// Public, unauthenticated reads for the Site Editor's admin-managed
// content — ads, homepage section visibility, page text blocks, and bank
// logo overrides. Same shape as useRatesApi.ts: plain $fetch, no auth,
// called from onMounted. Fetched at runtime (not baked into the static
// build) so an admin edit reaches visitors without a rebuild — see
// migrations/007_site_editor.sql for why.

export interface SiteAd {
  id: number
  title: string
  image_url: string
  target_url: string
}

export interface SiteSection {
  id: number
  page: string
  section_key: string
  label: string
  is_visible: boolean
  layout_variant: string
  background_image_url: string | null
  sort_order: number
}

export interface SiteContentBlock {
  id: number
  page: string
  content_key: string
  label: string
  body: string
}

export interface BankLogoOverride {
  bank_slug: string
  logo_url: string | null
  logo_small_url: string | null
}

interface ListResponse<T> {
  data: T[]
}

export function useSiteContent() {
  return {
    fetchAds: (slotKey: string) => $fetch<ListResponse<SiteAd>>(`/api/v1/site/ads?slot=${encodeURIComponent(slotKey)}`).then((r) => r.data || []),
    fetchSections: (page = 'home') => $fetch<ListResponse<SiteSection>>(`/api/v1/site/sections?page=${encodeURIComponent(page)}`).then((r) => r.data || []),
    fetchPageContent: (page: string) => $fetch<ListResponse<SiteContentBlock>>(`/api/v1/site/content?page=${encodeURIComponent(page)}`).then((r) => r.data || []),
    fetchBankLogoOverrides: () => $fetch<ListResponse<BankLogoOverride>>('/api/v1/site/bank-logos').then((r) => r.data || [])
  }
}
