-- Aligns every ad slot's declared size with a real IAB standard ad size,
-- matching the actual pixel dimensions each placement now renders at
-- (frontend/app/pages/index.vue, AppFooter.vue, banks/[slug].vue,
-- ProductResultList.vue, CompareResultList.vue) — previously these were
-- arbitrary stretch-to-fill boxes with no real creative size behind
-- them, so an uploaded ad was silently cropped/stretched (object-fit:
-- cover) rather than displayed at its native resolution. Only the label
-- text changes here; no schema change.

UPDATE ad_slots SET size = '970×250 (Billboard)' WHERE key = 'home-after-hero';
UPDATE ad_slots SET size = '728×90 (Leaderboard)' WHERE key = 'home-mid';
UPDATE ad_slots SET size = '970×90 (Large Leaderboard)' WHERE key = 'home-pre-footer';
UPDATE ad_slots SET size = '970×90 (Large Leaderboard)' WHERE key = 'footer-strip';
UPDATE ad_slots SET size = '728×90 (Leaderboard)' WHERE key = 'bank-profile';
UPDATE ad_slots SET size = '728×90 (Leaderboard)' WHERE key = 'product-list-native';
UPDATE ad_slots SET size = '120×600 (Skyscraper)' WHERE key IN
    ('skyscraper-left', 'skyscraper-right', 'home-skyscraper-left', 'home-skyscraper-right',
     'banks-skyscraper-left', 'banks-skyscraper-right', 'products-skyscraper-left', 'products-skyscraper-right');
