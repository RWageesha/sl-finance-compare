# SL Finance Compare

Scrapes, normalizes, and compares Sri Lankan bank financial product rates.
Currently tracks Fixed Deposit rates from Hatton National Bank (HNB),
Commercial Bank of Ceylon (ComBank), and Bank of Ceylon (BOC).

## Project layout

```
cmd/api/main.go            REST API (net/http) — GET /api/v1/fixed-deposits, serves web/
cmd/scraper/main.go        One-shot scraper pipeline runner (all banks)
internal/db                Postgres access via pgx
internal/models            Shared domain types
internal/scrapers/hnb      HNB Fixed Deposit scraper (JSON API client)
internal/scrapers/combank  ComBank Fixed Deposit scraper (HTML/goquery)
internal/scrapers/boc      BOC Fixed Deposit scraper (HTML/goquery)
internal/scrapers/ratetext Shared tenure/rate string parsing, used by all scrapers
migrations                 SQL migrations, applied in filename order
web/index.html             Static frontend — sortable/filterable rate comparison table
```

## Prerequisites

- Go 1.22+
- A native PostgreSQL 16 install
- `psql` (for applying migrations and inspecting the database directly)

## Setup

1. Copy the environment file and adjust if needed:

   ```sh
   cp .env.example .env
   ```

2. Install Go dependencies:

   ```sh
   go mod tidy
   ```

3. Install PostgreSQL 16 natively, then create the role/database
   `.env.example` expects:

   ```sh
   psql -U postgres -c "CREATE ROLE slfinance LOGIN PASSWORD 'slfinance';"
   psql -U postgres -c "CREATE DATABASE slfinance OWNER slfinance;"
   ```

4. Apply migrations (in order):

   ```sh
   psql "$DATABASE_URL" -f migrations/001_init.sql
   psql "$DATABASE_URL" -f migrations/002_seed.sql
   ```

## Running the scraper

```sh
go run ./cmd/scraper
```

Runs every configured bank's scraper in turn and inserts results into the
`fixed_deposit_rates` table. Each run adds new rows (rather than
overwriting), so the database accumulates a rate history over time. One
bank failing (site down, markup changed) doesn't block the others — each
scraper's outcome is logged individually, and the process only exits
non-zero if at least one bank failed.

- **HNB**: its rates page is a client-rendered React app with no
  server-side HTML table, so the scraper talks directly to the same JSON
  API the page itself calls
  (`https://venus.hnb.lk/api/get_interest_rates_contents`). Debug with:

  ```sh
  HNB_DEBUG_DUMP_JSON=1 go run ./cmd/scraper
  ```

  writes the fetched response to `hnb_fixed_deposits.debug.json`.

- **ComBank**: its rates page is server-rendered HTML with a real table, so
  the scraper parses it directly with goquery. Debug with:

  ```sh
  COMBANK_DEBUG_DUMP_HTML=1 go run ./cmd/scraper
  ```

  writes the fetched page to `combank_fixed_deposits.debug.html`.

- **BOC**: its rates & tariff page (`https://www.boc.lk/rates-tariff`) is
  server-rendered HTML bundling many unrelated sections (exchange rates,
  loan rates, a suspended senior citizen scheme, etc.) — the scraper picks
  out only the "Rupee Fixed Deposits" table by section title. Cells are
  trilingual (Sinhala/Tamil/English, `<br>`-separated); the parser keeps
  only the English segment. Debug with:

  ```sh
  BOC_DEBUG_DUMP_HTML=1 go run ./cmd/scraper
  ```

  writes the fetched page to `boc_fixed_deposits.debug.html`.

### Running the scraper on a schedule

The scraper is a one-shot binary, meant to be invoked periodically by the
OS scheduler rather than run as a long-lived daemon. Build it once:

```sh
go build -o bin/scraper.exe ./cmd/scraper
```

On Windows, register a scheduled task pointing at `bin/run_scraper.bat`
(which `cd`s into the project root first so `.env` is found, and appends
output to `scraper.log`):

```powershell
schtasks /create /tn "SLFinanceCompare-Scraper" /tr "E:\SL-Finance_Compare\bin\run_scraper.bat" /sc HOURLY /mo 12 /st 06:00
```

On Linux/macOS, a cron entry calling the compiled binary from the project
directory works the same way.

## Running the API

```sh
go run ./cmd/api
```

The API and static frontend are served from the same process:

- `http://localhost:8080/` — sortable/filterable HTML comparison table
- `http://localhost:8080/api/v1/fixed-deposits` — the same data as JSON

```sh
curl http://localhost:8080/api/v1/fixed-deposits
```

Returns the latest known rate per (bank, tenure, rate type) as JSON.

## Verifying data in Postgres

```sh
psql "$DATABASE_URL" -c "SELECT b.code, r.tenure_months, r.interest_rate, r.scraped_at FROM fixed_deposit_rates r JOIN banks b ON b.id = r.bank_id ORDER BY r.scraped_at DESC LIMIT 20;"
```
