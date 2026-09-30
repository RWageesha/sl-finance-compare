-- The skyscraper rails were a single "sitewide" pair of slots, shown
-- identically on every page with no way to target one page specifically
-- — every other ad placement (banners, native rows, overlays) already
-- lets an admin pick which page(s) it shows on via the wizard's
-- Placement step, since each real page has its own ad_slots rows. Adds
-- the same per-page slots for the vertical rails, one pair per real page
-- group. The original "sitewide" pair stays as an "every page" option
-- (SkyscraperRails.vue merges page-specific + sitewide candidates), just
-- relabeled for clarity now that page-specific ones exist alongside it.

UPDATE ad_slots SET label = 'Site-wide (all pages) — Skyscraper Left' WHERE key = 'skyscraper-left';
UPDATE ad_slots SET label = 'Site-wide (all pages) — Skyscraper Right' WHERE key = 'skyscraper-right';

INSERT INTO ad_slots (key, label, page_key, orientation, size) VALUES
    ('home-skyscraper-left',      'Home — Skyscraper Left',              'home',     'v', '160×600'),
    ('home-skyscraper-right',     'Home — Skyscraper Right',             'home',     'v', '160×600'),
    ('banks-skyscraper-left',     'Bank Profile — Skyscraper Left',      'banks',    'v', '160×600'),
    ('banks-skyscraper-right',    'Bank Profile — Skyscraper Right',     'banks',    'v', '160×600'),
    ('products-skyscraper-left',  'Product/Compare — Skyscraper Left',   'products', 'v', '160×600'),
    ('products-skyscraper-right', 'Product/Compare — Skyscraper Right',  'products', 'v', '160×600');
