// Thin wrapper around the Go API's three rate endpoints. Every field here
// mirrors models.ProductRate's JSON tags (internal/models/models.go) —
// see nuxt.config.ts's nitro.devProxy for how "/api/..." resolves in dev
// vs. the same-origin production static build.

export interface ProductRate {
  id: number
  product_id: number
  bank_name: string
  bank_code: string
  category_code: string
  product_name: string
  tenure_value?: number
  tenure_unit?: string
  tenure_label?: string
  rate_label?: string
  min_amount?: number
  interest_rate: number
  source_url?: string
  confidence?: string
  scraped_at: string
}

interface RatesResponse {
  count: number
  data: ProductRate[]
}

async function fetchRates(path: string): Promise<ProductRate[]> {
  const res = await $fetch<RatesResponse>(path)
  return res.data || []
}

// Identifies one exact logical product line the same way the Go API's
// GetLatestRates groups rows for the comparison tables: a product plus its
// tenure/rate_label. Pass the values straight off a ProductRate row (e.g.
// { product_id: row.product_id, tenure_value: row.tenure_value, ... }) to
// fetch every scrape ever recorded for that same line.
export interface RateHistoryParams {
  product_id: number
  tenure_value?: number
  tenure_label?: string
  rate_label?: string
}

function historyQuery(params: RateHistoryParams): string {
  const q = new URLSearchParams()
  q.set('product_id', String(params.product_id))
  if (params.tenure_value !== undefined) q.set('tenure_value', String(params.tenure_value))
  if (params.tenure_label) q.set('tenure_label', params.tenure_label)
  if (params.rate_label) q.set('rate_label', params.rate_label)
  return q.toString()
}

export function useRatesApi() {
  return {
    fetchFixedDeposits: () => fetchRates('/api/v1/fixed-deposits'),
    fetchSavings: () => fetchRates('/api/v1/savings-rates'),
    fetchLoans: () => fetchRates('/api/v1/loan-rates'),
    // Oldest first — every scrape ever recorded for one product line.
    fetchRateHistory: (params: RateHistoryParams) => fetchRates(`/api/v1/rate-history?${historyQuery(params)}`)
  }
}
