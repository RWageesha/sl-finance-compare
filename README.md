# SL Finance Compare

Scrapes, normalizes, and compares Sri Lankan bank financial product rates.
Tracks Fixed Deposit rates from Hatton National Bank (HNB), Commercial Bank
of Ceylon (ComBank), and Bank of Ceylon (BOC), plus Savings Account and Loan
rates from HNB.

## Project layout

```
cmd/api/main.go            REST API (net/http) — GET /api/v1/fixed-deposits,
                            /api/v1/savings-rates, /api/v1/loan-rates; serves web/
cmd/scraper/main.go        One-shot scraper pipeline runner (all banks/products)
internal/db                Postgres access via pgx
internal/models            Shared domain types, including the normalized
                            ProductCategory/Product/ProductRate schema
internal/normalize         Maps each scraper's raw output (FixedDepositRate/
                            SavingsRate/LoanRate) onto the normalized schema
internal/validate           Plausibility checks applied before a rate is stored
internal/scrapers/hnb      HNB scraper (JSON API client) — fixed deposits,
                            savings accounts, and loans
internal/scrapers/combank  ComBank Fixed Deposit scraper (HTML/goquery)
internal/scrapers/boc      BOC Fixed Deposit scraper (HTML/goquery)
internal/scrapers/ratetext Shared tenure/rate string parsing, used by all scrapers
migrations                 SQL migrations, applied in filename order
web/index.html             Static frontend — tabbed, sortable/filterable rate
                            comparison table (Fixed Deposits / Savings / Loans)
```

### Database schema

Rates are stored in a normalized, bank-agnostic schema rather than one
table per product type:

```
banks
product_categories   -- taxonomy: DEPOSIT > FIXED_DEPOSIT > STANDARD_FD, etc.
products              -- one row per (bank, category, bank's own product name)
product_rates         -- dated rate records; append-only, so this table is
                          itself the rate history — the API surfaces only the
                          latest row per (product, tenure, rate label)
data_sources           -- one row per (bank, scraper) the scraper fetches from
scrape_runs             -- one row per scrape attempt (status, row count, error)
```

Every scraper's `Parse*` function still returns the same typed structs it
always has (`FixedDepositRate`, `SavingsRate`, `LoanRate`) — the
`internal/normalize` package is what maps those onto `product_rates`,
resolving/creating the right category and product rows along the way.

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
   psql "$DATABASE_URL" -f migrations/003_savings_and_loans.sql
   psql "$DATABASE_URL" -f migrations/004_normalized_schema.sql
   ```

   Migration 004 replaces `fixed_deposit_rates`/`savings_rates`/
   `loan_rates` (from 001/003) with the normalized schema above — it drops
   those three tables, so this is only safe to run against a database
   whose scraped data you're fine losing (the next scrape run repopulates
   everything anyway).

## Running the scraper

```sh
go run ./cmd/scraper
```

Runs every configured bank/product scraper in turn, normalizes each result
onto the shared schema (see `internal/normalize`), and inserts it into
`product_rates`. Each run adds new rows (rather than overwriting), so the
database accumulates a rate history over time, and also records one
`scrape_runs` row per (bank, product type) attempt for basic operational
visibility. One bank or product type failing (site down, markup changed)
doesn't block the others — each scraper's outcome is logged individually,
and the process only exits non-zero if at least one failed.

- **HNB**: its rates page is a client-rendered React app with no
  server-side HTML table, so the scraper talks directly to the same JSON
  API the page itself calls
  (`https://venus.hnb.lk/api/get_interest_rates_contents`). That one API
  response contains all of HNB's published rates, so it's fetched once and
  parsed three ways — fixed deposits, savings accounts, and loans — each
  inserted into its own table. Loan rates that aren't a single flat
  percentage (floating-rate formulas like `AWPLR + 2.50%`, ranges, foreign
  currency loans) are skipped rather than surfaced as a misleading number.
  Debug with:

  ```sh
  HNB_DEBUG_DUMP_JSON=1 go run ./cmd/scraper
  ```

  writes the fetched response to `hnb_rates.debug.json`.

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

- `http://localhost:8080/` — tabbed, sortable/filterable HTML comparison
  table (Fixed Deposits / Savings / Loans)
- `http://localhost:8080/api/v1/fixed-deposits` — Fixed Deposit rates as JSON
- `http://localhost:8080/api/v1/savings-rates` — Savings Account rates as JSON
- `http://localhost:8080/api/v1/loan-rates` — Loan rates as JSON

```sh
curl http://localhost:8080/api/v1/fixed-deposits
curl http://localhost:8080/api/v1/savings-rates
curl http://localhost:8080/api/v1/loan-rates
```

Each endpoint queries `product_rates` filtered to a top-level category
group (`FIXED_DEPOSIT`, `SAVINGS`, or `LOAN`) and returns the latest row
per (product, tenure, rate label) as JSON.

## Verifying data in Postgres

```sh
psql "$DATABASE_URL" -c "SELECT c.code, count(*) FROM product_rates r JOIN products p ON p.id = r.product_id JOIN product_categories c ON c.id = p.category_id GROUP BY c.code ORDER BY c.code;"
psql "$DATABASE_URL" -c "SELECT b.code, p.name, r.tenure_value, r.tenure_label, r.rate_label, r.interest_rate FROM product_rates r JOIN products p ON p.id = r.product_id JOIN banks b ON b.id = p.bank_id ORDER BY r.scraped_at DESC LIMIT 20;"
psql "$DATABASE_URL" -c "SELECT ds.label, sr.status, sr.records_found, sr.error_message FROM scrape_runs sr JOIN data_sources ds ON ds.id = sr.source_id ORDER BY sr.id DESC LIMIT 20;"
```
