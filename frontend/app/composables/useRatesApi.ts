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

export function useRatesApi() {
  return {
    fetchFixedDeposits: () => fetchRates('/api/v1/fixed-deposits'),
    fetchSavings: () => fetchRates('/api/v1/savings-rates'),
    fetchLoans: () => fetchRates('/api/v1/loan-rates')
  }
}
