-- Normalized schema: banks -> products -> product_rates, with a shared
-- product_categories taxonomy so every bank's data lands in the same
-- comparable shape regardless of what the bank itself calls the product.
-- Replaces fixed_deposit_rates / savings_rates / loan_rates.

ALTER TABLE banks ADD COLUMN IF NOT EXISTS slug TEXT;
UPDATE banks SET slug = lower(code) WHERE slug IS NULL;
ALTER TABLE banks ALTER COLUMN slug SET NOT NULL;
ALTER TABLE banks ADD CONSTRAINT banks_slug_unique UNIQUE (slug);

CREATE TABLE product_categories (
    id SERIAL PRIMARY KEY,
    parent_id INTEGER REFERENCES product_categories(id),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    bank_id INTEGER NOT NULL REFERENCES banks(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES product_categories(id),
    name TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'LKR',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bank_id, category_id, name)
);

-- Every scrape adds new rows rather than overwriting, so this table is
-- itself the rate history — the API surfaces only the latest row per
-- (product, tenure, rate_label) via a DISTINCT ON query.
CREATE TABLE product_rates (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tenure_value INTEGER,
    tenure_unit TEXT,
    tenure_label TEXT NOT NULL DEFAULT '',
    rate_label TEXT NOT NULL DEFAULT '',
    min_amount NUMERIC(14, 2) CHECK (min_amount IS NULL OR min_amount >= 0),
    interest_rate NUMERIC(6, 3) NOT NULL CHECK (interest_rate >= 0 AND interest_rate <= 100),
    source_url TEXT,
    confidence TEXT NOT NULL DEFAULT 'high',
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_product_rates_product_id ON product_rates (product_id);

CREATE INDEX idx_product_rates_latest
    ON product_rates (product_id, tenure_value, tenure_label, rate_label, scraped_at DESC);

CREATE TABLE data_sources (
    id SERIAL PRIMARY KEY,
    bank_id INTEGER NOT NULL REFERENCES banks(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    source_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bank_id, label)
);

CREATE TABLE scrape_runs (
    id BIGSERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL,
    records_found INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE INDEX idx_scrape_runs_source_id ON scrape_runs (source_id, started_at DESC);

DROP TABLE IF EXISTS fixed_deposit_rates;
DROP TABLE IF EXISTS savings_rates;
DROP TABLE IF EXISTS loan_rates;

-- Taxonomy seed: only the categories the current scrapers actually
-- produce. Extend as new product shapes are added.
INSERT INTO product_categories (code, name, parent_id) VALUES
    ('DEPOSIT', 'Deposits', NULL),
    ('LOAN', 'Loans', NULL)
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, parent_id)
SELECT 'FIXED_DEPOSIT', 'Fixed Deposits', id FROM product_categories WHERE code = 'DEPOSIT'
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, parent_id)
SELECT 'SAVINGS', 'Savings Accounts', id FROM product_categories WHERE code = 'DEPOSIT'
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, parent_id)
SELECT code, name, id FROM (
    SELECT id FROM product_categories WHERE code = 'FIXED_DEPOSIT'
) fd
CROSS JOIN (VALUES
    ('STANDARD_FD', 'Standard Fixed Deposit'),
    ('SENIOR_CITIZEN_FD', 'Senior Citizen Fixed Deposit'),
    ('SPECIAL_FD', 'Special Fixed Deposit'),
    ('SATHKARA_FD', 'Sathkara Fixed Deposit'),
    ('DIGITAL_FD', 'Digital Fixed Deposit')
) AS t(code, name)
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, parent_id)
SELECT code, name, id FROM (
    SELECT id FROM product_categories WHERE code = 'SAVINGS'
) sv
CROSS JOIN (VALUES
    ('STANDARD_SAVINGS', 'Standard Savings'),
    ('SENIOR_SAVINGS', 'Senior Citizen Savings'),
    ('TEEN_SAVINGS', 'Teen Savings'),
    ('WOMENS_SAVINGS', 'Women''s Savings'),
    ('MINOR_SAVINGS', 'Minor Savings')
) AS t(code, name)
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, parent_id)
SELECT code, name, id FROM (
    SELECT id FROM product_categories WHERE code = 'LOAN'
) ln
CROSS JOIN (VALUES
    ('HOUSING_LOAN', 'Housing Loan'),
    ('PERSONAL_LOAN', 'Personal Loan'),
    ('EDUCATION_LOAN', 'Education Loan'),
    ('GOLD_LOAN', 'Gold Loan / Pawning'),
    ('LEASE', 'Leasing'),
    ('PENSIONER_LOAN', 'Pensioner Loan'),
    ('OTHER_LOAN', 'Other Loan')
) AS t(code, name)
ON CONFLICT (code) DO NOTHING;
