-- Rebuilds the Ad Manager to match the approved wizard spec: 6 ad styles
-- (image/gif/video/slider/shared/overlay), an ad placed on several
-- page+slot combinations at once, several creative files per ad
-- (slider/shared), and weighted rotation when a slot has more than one
-- active ad. Safe as a clean replacement — production `ads` has 0 rows
-- (the only prior save attempt failed validation and was never
-- persisted), so no data migration is needed.

-- ad_slots gains page grouping + physical shape; 'kind' is replaced by
-- 'orientation', the thing that actually gates which ad styles/layouts
-- fit a given slot (a vertical ad can't go in a horizontal banner).
ALTER TABLE ad_slots ADD COLUMN page_key TEXT NOT NULL DEFAULT 'home';
ALTER TABLE ad_slots ADD COLUMN orientation TEXT NOT NULL DEFAULT 'h'; -- 'h' | 'v' | 'overlay'
ALTER TABLE ad_slots ADD COLUMN size TEXT NOT NULL DEFAULT '';
ALTER TABLE ad_slots DROP COLUMN kind;

-- Re-map the 8 existing physical placements onto real page groups.
-- skyscraper-left/right and footer-strip render on every page (app.vue /
-- AppFooter.vue), so they belong to a virtual 'sitewide' page, not 'home'.
UPDATE ad_slots SET page_key = 'sitewide', orientation = 'v', size = '160×600' WHERE key IN ('skyscraper-left', 'skyscraper-right');
UPDATE ad_slots SET page_key = 'sitewide', orientation = 'h', size = '1200×100' WHERE key = 'footer-strip';
UPDATE ad_slots SET page_key = 'home', orientation = 'h', size = '1200×100' WHERE key IN ('home-after-hero', 'home-mid', 'home-pre-footer');
UPDATE ad_slots SET page_key = 'banks', orientation = 'h', size = '1200×80' WHERE key = 'bank-profile';
UPDATE ad_slots SET page_key = 'products', orientation = 'h', size = '1200×64' WHERE key = 'product-list-native';

-- One overlay slot per page group — overlays cover the whole page, so
-- they don't compete for horizontal/vertical space, but still go through
-- the same ad_placements join as every other slot rather than a special case.
INSERT INTO ad_slots (key, label, page_key, orientation, size) VALUES
    ('home-overlay',     'Home — Overlay',             'home',     'overlay', ''),
    ('banks-overlay',    'Bank Profile — Overlay',     'banks',    'overlay', ''),
    ('products-overlay', 'Product/Compare — Overlay',  'products', 'overlay', ''),
    ('sitewide-overlay', 'Site-wide — Overlay',        'sitewide', 'overlay', '');

-- ads: slot_id/image_url/target_url move out — an ad can now span
-- several slots (ad_placements) and carry several files (ad_creatives).
ALTER TABLE ads DROP CONSTRAINT ads_slot_id_fkey;
ALTER TABLE ads DROP COLUMN slot_id;
ALTER TABLE ads DROP COLUMN image_url;
ALTER TABLE ads DROP COLUMN target_url;
ALTER TABLE ads ADD COLUMN advertiser TEXT; -- required for ad_type 'sponsored'/'partner', enforced in the API
ALTER TABLE ads ADD COLUMN layout TEXT NOT NULL DEFAULT 'horizontal'; -- 'horizontal' | 'vertical'
ALTER TABLE ads ADD COLUMN style TEXT NOT NULL DEFAULT 'image'; -- 'image'|'gif'|'video'|'slider'|'shared'|'overlay'
ALTER TABLE ads ADD COLUMN settings JSONB NOT NULL DEFAULT '{}'; -- style-specific: slide interval, tile gap, overlay timing
ALTER TABLE ads ADD COLUMN devices TEXT NOT NULL DEFAULT 'all'; -- 'all' | 'desktop' | 'mobile'
ALTER TABLE ads ADD COLUMN weight INT NOT NULL DEFAULT 5 CHECK (weight BETWEEN 1 AND 10);
COMMENT ON COLUMN ads.ad_type IS 'house, sponsored, or partner (was house/sponsor before the ad wizard)';

-- One row per file: position 1..N for slider (3-5) / shared (2-3),
-- exactly one row for image/gif/video/overlay.
CREATE TABLE ad_creatives (
    id         SERIAL PRIMARY KEY,
    ad_id      INT NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
    position   INT NOT NULL,
    media_url  TEXT NOT NULL,
    target_url TEXT NOT NULL,
    alt_text   TEXT,
    poster_url TEXT, -- video only — shown before the video loads
    UNIQUE (ad_id, position)
);

-- Which page+slot combinations this ad appears in. Many-to-many: one ad
-- can be placed on several slots; a slot can host several ads (rotation).
CREATE TABLE ad_placements (
    ad_id   INT NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
    slot_id INT NOT NULL REFERENCES ad_slots(id) ON DELETE CASCADE,
    PRIMARY KEY (ad_id, slot_id)
);
