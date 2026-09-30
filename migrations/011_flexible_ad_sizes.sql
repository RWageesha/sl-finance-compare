-- Ad size became a per-ad choice (settings.width/height, chosen in the
-- wizard's Format step) instead of a fixed property of the slot — any
-- horizontal slot can now host a Leaderboard, Large Leaderboard, or
-- Billboard depending on what the admin picks for that specific ad, and
-- any vertical rail can host a Skyscraper, Wide Skyscraper, Half Page, or
-- Portrait. The old single-size labels (e.g. "728×90 (Leaderboard)") are
-- no longer accurate for any slot, so they're replaced with a plain
-- description of what fits, matching how the Placement step actually
-- behaves now (frontend/app/components/AdSlot.vue, pages/admin/
-- site-editor/ads.vue's SIZE_OPTIONS).

UPDATE ad_slots SET size = 'Any horizontal size (Leaderboard, Large Leaderboard, Billboard, or mobile banner)' WHERE orientation = 'h';
UPDATE ad_slots SET size = 'Any vertical size (Skyscraper, Wide Skyscraper, Half Page, or Portrait)' WHERE orientation = 'v';
UPDATE ad_slots SET size = 'Covers the whole page — no fixed size' WHERE orientation = 'overlay';
