-- Site Editor: ad slots/ads, homepage section visibility, page text
-- blocks, and bank logo overrides — the admin-editable content layer the
-- site never had before (everything was hardcoded in .vue files). Every
-- table here is read by a public, unauthenticated /api/v1/site/* endpoint
-- at runtime, since the frontend is a static build regenerated only on
-- deploy — admin edits have to reach visitors without a rebuild.

-- Fixed placements defined in the frontend (AdSlot.vue call sites), not
-- admin-creatable — this table exists so the admin UI has real slot ids
-- to attach ads to and a human label to show, not a random string field.
CREATE TABLE ad_slots (
    id         SERIAL PRIMARY KEY,
    key        TEXT NOT NULL UNIQUE,
    label      TEXT NOT NULL,
    kind       TEXT NOT NULL, -- 'skyscraper' | 'banner' | 'native'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO ad_slots (key, label, kind) VALUES
    ('skyscraper-left',    'Skyscraper — Left Rail',           'skyscraper'),
    ('skyscraper-right',   'Skyscraper — Right Rail',          'skyscraper'),
    ('home-after-hero',    'Homepage — After Hero',            'banner'),
    ('home-mid',           'Homepage — Between Banks & Calculators', 'banner'),
    ('home-pre-footer',    'Homepage — Before Transparency Banner',  'banner'),
    ('footer-strip',       'Site-wide — Above Footer',         'banner'),
    ('bank-profile',       'Bank Profile — Between Product Groups',  'banner'),
    ('product-list-native','Product/Compare Lists — In-Result Row',  'native');

-- One ad creative. Several rows can share a slot_id — the frontend
-- rotates between whichever are currently active. ends_at NULL means "no
-- end date" (runs until deactivated); starts_at/ends_at also cover
-- outsider sponsors bought for a fixed date range.
CREATE TABLE ads (
    id                  SERIAL PRIMARY KEY,
    slot_id             INT NOT NULL REFERENCES ad_slots(id) ON DELETE CASCADE,
    ad_type             TEXT NOT NULL DEFAULT 'house', -- 'house' | 'sponsor'
    title               TEXT NOT NULL,
    image_url           TEXT NOT NULL,
    target_url          TEXT NOT NULL,
    sort_order          INT NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    starts_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at             TIMESTAMPTZ,
    created_by_admin_id INT REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ads_slot_active ON ads (slot_id, is_active);

-- Homepage (and later other pages) section registry — visibility +
-- layout variant + display order. Seeded with the current
-- pages/index.vue sections so nothing changes visually until an admin
-- edits one.
CREATE TABLE site_sections (
    id                  SERIAL PRIMARY KEY,
    page                TEXT NOT NULL DEFAULT 'home',
    section_key         TEXT NOT NULL UNIQUE,
    label               TEXT NOT NULL,
    is_visible          BOOLEAN NOT NULL DEFAULT true,
    layout_variant      TEXT NOT NULL DEFAULT 'default',
    -- Only used by 'hero' today — lets an admin replace the homepage
    -- hero background without a redeploy. NULL = use the built-in
    -- default (public/hero/skyline.jpg).
    background_image_url TEXT,
    sort_order          INT NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO site_sections (section_key, label, sort_order) VALUES
    ('hero',                  'Hero',                      0),
    ('popular_products',      'Popular Financial Products', 1),
    ('popular_banks',         'Most Popular Banks',        2),
    ('quick_calculators',     'Quick Calculators',         3),
    ('head_to_head',          'Head-to-Head Comparisons',  4),
    ('recent_rates',          'Recently Updated Rates',    5),
    ('transparency_banner',   'Transparency Banner',       6);

-- Simple named text blocks — About Us / Contact Us body copy today,
-- reusable for any future "edit this paragraph" need. Not a rich CMS:
-- one plain-text/markdown-lite field per block.
CREATE TABLE site_content (
    id                  SERIAL PRIMARY KEY,
    page                TEXT NOT NULL,
    content_key         TEXT NOT NULL UNIQUE,
    label               TEXT NOT NULL,
    body                TEXT NOT NULL DEFAULT '',
    updated_by_admin_id INT REFERENCES admin_users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO site_content (page, content_key, label, body) VALUES
    ('about-us',   'about-us-body',   'About Us Body',   'FindRate LK helps Sri Lankans compare fixed deposit, savings, loan, and card rates across licensed banks and finance companies — content coming soon.'),
    ('contact-us', 'contact-us-body', 'Contact Us Body', 'Have a question or found an error in our data? Use the Report Incorrect Information form, or reach us on GitHub — content coming soon.');

-- Bank logo overrides only — bankDirectory.ts stays the source of truth
-- for name/type/tracked/etc.; this just lets an admin replace the two
-- logo image files for a bank without a code change + redeploy.
CREATE TABLE bank_logo_overrides (
    id                  SERIAL PRIMARY KEY,
    bank_slug           TEXT NOT NULL UNIQUE,
    logo_url            TEXT,
    logo_small_url      TEXT,
    updated_by_admin_id INT REFERENCES admin_users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
