-- Savings account and loan rates, mirroring fixed_deposit_rates: every
-- scrape adds new rows rather than overwriting, so rate history
-- accumulates over time. The API surfaces the latest row per natural key.

CREATE TABLE IF NOT EXISTS savings_rates (
    id BIGSERIAL PRIMARY KEY,
    bank_id INTEGER NOT NULL REFERENCES banks(id) ON DELETE CASCADE,
    account_name TEXT NOT NULL,
    balance_tier TEXT NOT NULL DEFAULT '',
    interest_rate NUMERIC(5, 2) NOT NULL CHECK (interest_rate >= 0 AND interest_rate <= 100),
    source_url TEXT,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_savings_rates_bank_id ON savings_rates (bank_id);

CREATE INDEX IF NOT EXISTS idx_savings_rates_latest
    ON savings_rates (bank_id, account_name, balance_tier, scraped_at DESC);

CREATE TABLE IF NOT EXISTS loan_rates (
    id BIGSERIAL PRIMARY KEY,
    bank_id INTEGER NOT NULL REFERENCES banks(id) ON DELETE CASCADE,
    loan_category TEXT NOT NULL,
    loan_product TEXT NOT NULL,
    rate_label TEXT NOT NULL DEFAULT '',
    tenure TEXT NOT NULL DEFAULT '',
    interest_rate NUMERIC(5, 2) NOT NULL CHECK (interest_rate >= 0 AND interest_rate <= 100),
    source_url TEXT,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_loan_rates_bank_id ON loan_rates (bank_id);

CREATE INDEX IF NOT EXISTS idx_loan_rates_latest
    ON loan_rates (bank_id, loan_category, loan_product, rate_label, tenure, scraped_at DESC);
