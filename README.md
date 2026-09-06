# FindRate LK

Scrapes, normalizes, and compares Sri Lankan bank financial product rates.
Tracks Fixed Deposit, Savings Account, and Loan rates from Hatton National
Bank (HNB), Commercial Bank of Ceylon (ComBank), Bank of Ceylon (BOC), and
National Development Bank (NDB).

**Target stack:** Nuxt.js frontend · Go backend · PostgreSQL · Python for
data collection. All four now match.

## Project layout

```
frontend/                   Nuxt 3 app — source of truth for the frontend;
                             see "Frontend (Nuxt.js)" below
  app/pages/{index,rates}.vue  Homepage and the rate comparison tool
  app/components/            ProductCard, CompareCard, RecentRatesTable, RatesTable
  app/composables/useRatesApi.ts  Fetches the Go API's three rate endpoints
  app/utils/format.ts         fmtDate/fmtTenure/formatCategoryLabel

web/                        Generated static output of `frontend/` (nuxt generate) —
                             build artifact, never hand-edit; served as-is by cmd/api

scraper/                    Python data collection — see "Running the scraper" below
  main.py                   Orchestrates every bank/product-type scrape
  db.py                     Postgres access via psycopg
  normalize.py              Maps each bank's raw output onto the normalized schema
  validate.py                Plausibility checks applied before a rate is stored
  ratetext.py                Shared tenure/rate string parsing
  banks/{hnb,combank,boc}.py  Per-bank fetch + parse

cmd/api/main.go            REST API (net/http) — GET /api/v1/fixed-deposits,
                            /api/v1/savings-rates, /api/v1/loan-rates; serves web/
internal/db                Postgres access via pgx (Go API side only)
internal/models            Shared domain types, including the normalized
                            ProductCategory/Product/ProductRate schema
internal/scrapers/*        Original Go scrapers — superseded by scraper/, kept
                            for reference; internal/normalize, internal/validate,
                            and cmd/scraper/main.go likewise unused now
migrations                 SQL migrations, applied in filename order
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

- Go 1.22+ (API server)
- Python 3.12+ (data collection)
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
   psql "$DATABASE_URL" -f migrations/005_admin_auth.sql
   ```

   Migration 004 replaces `fixed_deposit_rates`/`savings_rates`/
   `loan_rates` (from 001/003) with the normalized schema above — it drops
   those three tables, so this is only safe to run against a database
   whose scraped data you're fine losing (the next scrape run repopulates
   everything anyway). Migration 005 adds the admin panel's auth tables
   (`admin_users`, `admin_sessions`, `audit_logs`, `user_reports`) plus a
   few admin-only columns on the existing tables — it's additive and safe
   to run on a database with real scraped data already in it.

5. Create the initial super admin account (one-time — the command
   refuses to run if a super_admin already exists):

   ```sh
   SEED_ADMIN_USERNAME=youradmin SEED_ADMIN_EMAIL=you@example.com SEED_ADMIN_PASSWORD=a-strong-password go run ./cmd/seed-admin
   ```

   Log in at `/admin/login` with those credentials, then change the
   password from Settings. Never commit real values for these — only
   `.env.example` is tracked; `.env` is gitignored.

## Running the scraper

```sh
cd scraper
python -m venv .venv
.venv/Scripts/activate        # Windows; use `source .venv/bin/activate` on Linux/macOS
pip install -r requirements.txt
python main.py                # reads DATABASE_URL from ../.env
```

Runs every configured bank/product scraper in turn, normalizes each result
onto the shared schema (see `normalize.py`), and inserts it into
`product_rates`. Each run adds new rows (rather than overwriting), so the
database accumulates a rate history over time, and also records one
`scrape_runs` row per (bank, product type) attempt for basic operational
visibility. One bank or product type failing (site down, markup changed)
doesn't block the others — each scraper's outcome is logged individually,
and the process only exits non-zero if at least one failed.

- **HNB**: its rates page is a client-rendered React app with no
  server-side HTML table, so `banks/hnb.py` talks directly to the same
  JSON API the page itself calls
  (`https://venus.hnb.lk/api/get_interest_rates_contents`). That one API
  response contains all of HNB's published rates, so it's fetched once and
  parsed three ways — fixed deposits, savings accounts, and loans. Loan
  rates that aren't a single flat percentage (floating-rate formulas like
  `AWPLR + 2.50%`, ranges, foreign currency loans) are skipped rather than
  surfaced as a misleading number.

- **ComBank**: `banks/combank.py` parses its FD page and a separate
  `rates-tariff` hub page (Savings + Loans) with BeautifulSoup — both are
  server-rendered HTML with real tables.

- **BOC**: `banks/boc.py` parses its `rates-tariff` page
  (`https://www.boc.lk/rates-tariff`), which bundles many unrelated
  sections (exchange rates, loan rates, a suspended senior citizen scheme,
  etc.) — the scraper picks out only specific sections by title (e.g.
  "Rupee Fixed Deposits", "Housing Loans"). Cells are trilingual
  (Sinhala/Tamil/English, `<br>`-separated); the parser keeps only the
  English segment.

### Running the scraper on a schedule

The scraper is a one-shot script, meant to be invoked periodically by the
OS scheduler rather than run as a long-lived process. On Windows, register
a scheduled task pointing at a small wrapper batch file that activates the
venv and runs `python main.py`; on Linux/macOS, a cron entry calling
`scraper/.venv/bin/python scraper/main.py` from the project directory
works the same way. In CI, `.github/workflows/scrape.yml` runs it on a
cron schedule via `actions/setup-python`.

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

## Frontend (Nuxt.js)

`frontend/` is the source of truth for the site — a Nuxt 3 app statically
generated (no SSR server) into `web/`, which `cmd/api` serves unchanged.
**Never hand-edit files under `web/`** — they're a build artifact and get
overwritten by the next `npm run generate`.

Dev workflow (hot-reload, run alongside the Go API):

```sh
go run ./cmd/api          # in one terminal — the API on :8080
cd frontend
npm install
npm run dev                # in another terminal — Nuxt on :3000
```

`nuxt.config.ts`'s `nitro.devProxy` forwards `/api/*` requests from the
Nuxt dev server to the Go API, so every page fetches the same relative
`/api/v1/...` paths in dev and in production (same-origin once built).

Building for deploy:

```sh
cd frontend
npm run generate           # writes static output into ../web
```

Commit the resulting `web/` changes along with your `frontend/` changes —
Render's deploy just builds and runs the Go binary (no Node build step),
so the generated static files need to already be in the repo.

## Verifying data in Postgres

```sh
psql "$DATABASE_URL" -c "SELECT c.code, count(*) FROM product_rates r JOIN products p ON p.id = r.product_id JOIN product_categories c ON c.id = p.category_id GROUP BY c.code ORDER BY c.code;"
psql "$DATABASE_URL" -c "SELECT b.code, p.name, r.tenure_value, r.tenure_label, r.rate_label, r.interest_rate FROM product_rates r JOIN products p ON p.id = r.product_id JOIN banks b ON b.id = p.bank_id ORDER BY r.scraped_at DESC LIMIT 20;"
psql "$DATABASE_URL" -c "SELECT ds.label, sr.status, sr.records_found, sr.error_message FROM scrape_runs sr JOIN data_sources ds ON ds.id = sr.source_id ORDER BY sr.id DESC LIMIT 20;"
```

## Deploying to Render + Supabase

This is a single deployable service: the Go binary serves both the API
and the prerendered static site (including `/admin`) from one process,
so there's one Render web service and one Supabase Postgres database —
no separate frontend host, no separate worker.

**1. Create the Supabase database**

- Create a project at [supabase.com](https://supabase.com).
- Once provisioned, go to **Project Settings → Database → Connection
  string** and copy the **direct connection** (port 5432), not the
  pooler/pgbouncer one on port 6543 — pgx's extended query protocol
  doesn't play well with pgbouncer's transaction pooling mode without
  extra config, and this project's traffic doesn't need pgbouncer's
  concurrency benefits anyway. It looks like:

  ```
  postgresql://postgres:[YOUR-PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres
  ```

- Apply the migrations against it, in order, from your machine (append
  `?sslmode=require` — Supabase requires SSL):

  ```sh
  export DATABASE_URL="postgresql://postgres:...@db.....supabase.co:5432/postgres?sslmode=require"
  psql "$DATABASE_URL" -f migrations/001_init.sql
  psql "$DATABASE_URL" -f migrations/002_seed.sql
  psql "$DATABASE_URL" -f migrations/003_savings_and_loans.sql
  psql "$DATABASE_URL" -f migrations/004_normalized_schema.sql
  psql "$DATABASE_URL" -f migrations/005_admin_auth.sql
  ```

- Seed your real super admin (pick your own values — never reuse a
  local test password in production):

  ```sh
  SEED_ADMIN_USERNAME=... SEED_ADMIN_EMAIL=... SEED_ADMIN_PASSWORD=... go run ./cmd/seed-admin
  ```

**2. Deploy the Go service on Render**

- In the Render dashboard: **New → Blueprint**, connect the
  `RWageesha/sl-finance-compare` GitHub repo. Render reads `render.yaml`
  at the repo root and provisions the `findrate-lk` web service
  automatically.
- On first deploy, Render will prompt for `DATABASE_URL` (it's declared
  `sync: false` in `render.yaml` so it's never committed) — paste the
  same Supabase connection string from step 1.
- Render injects `PORT` automatically; `cmd/api/main.go` already binds
  to it, no extra config needed.
- Once live, the whole site (public pages, the API, and `/admin`) is
  served from the one Render URL — e.g.
  `https://findrate-lk.onrender.com/admin/login`.

**3. Keep the scraper running**

The Python scraper (`scraper/`) is a separate concern from this
service — it writes directly to Postgres and isn't invoked by the Go
API in production (only in local dev, via the admin panel's "Run
Manual"/"Run All Scrapers" buttons — see their code comments). Point its
own `DATABASE_URL` at the same Supabase database, and schedule it
however you're already running it (GitHub Actions manual-dispatch or a
Render Cron Job) — nothing about this deploy changes that setup.
