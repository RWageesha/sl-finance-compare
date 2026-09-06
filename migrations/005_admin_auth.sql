-- Admin authentication, sessions, audit trail, and the schema additions
-- the admin panel needs on top of the existing normalized tables.
-- verification_status defaults to 'pending' for anything inserted from
-- here on (the scraper's INSERTs don't set it, so they'll always land as
-- 'pending' for the Verification Queue to review) — the UPDATE below
-- backfills every row that already existed before this migration to
-- 'verified', since those rates are already live on the public site and
-- were never meant to retroactively look unreviewed.

CREATE TABLE admin_users (
    id            SERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'admin', -- 'super_admin' | 'admin' | 'editor' | 'viewer'
    status        TEXT NOT NULL DEFAULT 'active', -- 'active' | 'disabled'
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

CREATE TABLE admin_sessions (
    id           SERIAL PRIMARY KEY,
    admin_id     INT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_admin_sessions_admin_id ON admin_sessions (admin_id);

CREATE TABLE audit_logs (
    id             SERIAL PRIMARY KEY,
    admin_id       INT REFERENCES admin_users(id) ON DELETE SET NULL,
    admin_username TEXT NOT NULL, -- denormalized so logs survive a deleted admin
    action         TEXT NOT NULL, -- e.g. 'Rate Approved', 'Source Disabled', 'Bank Added'
    record_ref     TEXT NOT NULL, -- e.g. 'Commercial Bank - Senior FD'
    old_value      TEXT,
    new_value      TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);

-- Public correction reports (from the site's Report Issue form) — a
-- distinct queue from the internal Verification Queue above, since these
-- come from anonymous visitors, not the scraper.
CREATE TABLE user_reports (
    id             SERIAL PRIMARY KEY,
    bank_name      TEXT NOT NULL,
    product_label  TEXT NOT NULL,
    issue_type     TEXT NOT NULL,
    current_value  TEXT,
    correct_value  TEXT NOT NULL,
    source_url     TEXT NOT NULL,
    description    TEXT,
    status         TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'resolved' | 'rejected'
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at    TIMESTAMPTZ,
    resolved_by    INT REFERENCES admin_users(id) ON DELETE SET NULL
);
CREATE INDEX idx_user_reports_status ON user_reports (status, created_at DESC);

-- Admin-facing metadata the public schema never needed: whether a bank is
-- actively managed, its market segment, and its public site.
ALTER TABLE banks ADD COLUMN status TEXT NOT NULL DEFAULT 'active'; -- 'active' | 'inactive'
ALTER TABLE banks ADD COLUMN bank_type TEXT NOT NULL DEFAULT 'Commercial Bank';
ALTER TABLE banks ADD COLUMN website TEXT;

-- Same idea for data sources: whether the scraper should still hit this
-- source, and what kind of page it is.
ALTER TABLE data_sources ADD COLUMN status TEXT NOT NULL DEFAULT 'active'; -- 'active' | 'disabled'
ALTER TABLE data_sources ADD COLUMN source_type TEXT NOT NULL DEFAULT 'HTML'; -- 'HTML' | 'PDF' | 'API'

ALTER TABLE product_rates ADD COLUMN verification_status TEXT NOT NULL DEFAULT 'pending'; -- 'pending' | 'verified' | 'rejected'
UPDATE product_rates SET verification_status = 'verified' WHERE verification_status = 'pending';
CREATE INDEX idx_product_rates_verification_status ON product_rates (verification_status);

-- Set only when an admin manually inserted this row (Rate Management's
-- "Edit" / "+ Add Manual Entry") — lets History show "Changed By: <admin>"
-- vs "Auto-detected" for a normal scrape, without guessing from timing.
ALTER TABLE product_rates ADD COLUMN created_by_admin_id INT REFERENCES admin_users(id) ON DELETE SET NULL;
