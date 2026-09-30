// Maps a route path to the page grouping used by ad_slots.page_key
// (migrations/008_ad_wizard.sql, 009_page_skyscrapers.sql) — shared by
// every component that needs to know "which page am I on" for ad
// targeting: SkyscraperRails.vue (per-page rails) and AdOverlay.vue
// (per-page overlays). Returns null for pages with no dedicated page
// group (e.g. /data, /report-issue) — callers fall back to the
// "sitewide" slots for those.
export function resolvePageKey(path: string): string | null {
  if (path === '/') return 'home'
  if (path.startsWith('/banks')) return 'banks'
  if (path.startsWith('/products') || path.startsWith('/compare')) return 'products'
  return null
}
