CREATE TABLE IF NOT EXISTS banks (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every scrape run inserts new rows rather than overwriting, so we keep a
-- history of rate changes over time. The API surfaces the latest row per
-- (bank, tenure, rate_type).
CREATE TABLE IF NOT EXISTS fixed_deposit_rates (
    id BIGSERIAL PRIMARY KEY,
    bank_id INTEGER NOT NULL REFERENCES banks(id) ON DELETE CASCADE,
    tenure_months INTEGER NOT NULL CHECK (tenure_months > 0),
    min_amount NUMERIC(14, 2) CHECK (min_amount IS NULL OR min_amount >= 0),
    interest_rate NUMERIC(5, 2) NOT NULL CHECK (interest_rate >= 0 AND interest_rate <= 100),
    rate_type TEXT NOT NULL DEFAULT 'normal',
    source_url TEXT,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fd_rates_bank_id ON fixed_deposit_rates (bank_id);

-- Supports the "latest rate per bank/tenure/type" lookup used by the API.
CREATE INDEX IF NOT EXISTS idx_fd_rates_latest
    ON fixed_deposit_rates (bank_id, tenure_months, rate_type, scraped_at DESC);
