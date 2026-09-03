// Shared helpers for the Fixed Deposit comparison flow:
//   /compare/fixed-deposits          (input)
//   /compare/fixed-deposits/results  (cards)
//   /compare/fixed-deposits/table    (side-by-side table)
//   /products/fixed-deposits/[slug]          (product detail)
//   /products/fixed-deposits/[slug]/history  (rate history)
//
// We only have one field set per row: bank, category_code, tenure_value,
// interest_rate, source_url, scraped_at. There's no minimum-deposit,
// interest-payment-frequency, fee, or eligibility data anywhere in the
// schema, and (so far) only one scrape run exists — so this file has no
// "fabricate the rest" fallback. Callers show honest gaps instead.
import type { ProductRate } from '~/composables/useRatesApi'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'

export function isFdRow(r: ProductRate): boolean {
  return r.category_code.endsWith('_FD')
}

export function bankSlugForApiName(apiName: string): string | undefined {
  return DIRECTORY_BANKS.find((b) => b.apiName === apiName)?.slug
}

// hnb-standard_fd-12 — bank slugs and category codes never contain a
// hyphen (slugs are single words, category codes use underscores), so
// splitting on "-" is unambiguous.
export function fdSlug(bankSlug: string, categoryCode: string, tenureMonths: number): string {
  return `${bankSlug}-${categoryCode.toLowerCase()}-${tenureMonths}`
}

export interface ParsedFdSlug {
  bankSlug: string
  categoryCode: string
  tenure: number
}

export function parseFdSlug(slug: string): ParsedFdSlug | null {
  const parts = slug.split('-')
  if (parts.length !== 3) return null
  const tenure = Number(parts[2])
  if (!Number.isFinite(tenure)) return null
  return { bankSlug: parts[0], categoryCode: parts[1].toUpperCase(), tenure }
}

export function findFdRow(rows: ProductRate[], parsed: ParsedFdSlug): ProductRate | undefined {
  const bank = DIRECTORY_BANKS.find((b) => b.slug === parsed.bankSlug)
  if (!bank?.apiName) return undefined
  return rows.find(
    (r) => r.bank_name === bank.apiName && r.category_code === parsed.categoryCode && r.tenure_value === parsed.tenure
  )
}

// Simple (non-compounded) interest prorated by tenure — same formula as
// FdCalculatorModal.vue, kept consistent across the site.
export function maturityValue(principal: number, annualRatePct: number, months: number): number {
  return principal * (1 + (annualRatePct / 100) * (months / 12))
}

// Standard reducing-balance EMI formula — same as LoanCalculatorModal.vue.
// Used on the loan comparison pages to project a monthly payment at a
// bank's real disclosed rate for a user-chosen amount/tenure; it's a
// projection tied to their rate, not a claim about the bank's own terms
// (we don't have processing fees, LTV limits, or eligibility data).
export function emiPayment(principal: number, annualRatePct: number, months: number): number {
  if (months <= 0) return 0
  const r = annualRatePct / 100 / 12
  if (r === 0) return principal / months
  const factor = Math.pow(1 + r, months)
  return (principal * r * factor) / (factor - 1)
}

export function fmtLkr(n: number): string {
  return 'Rs. ' + Math.round(n).toLocaleString('en-LK')
}

// Product-detail link for any row, used by RatesTable.vue and the Product
// Directory page. FD rows use the stable bank+category+tenure slug (the
// tenure genuinely identifies one logical product across re-scrapes).
// Savings and loan rows can't use that scheme: within the same bank and
// category, multiple real rows often share an identical (or blank)
// tenure_label/rate_label with different rates — e.g. BOC has three
// STANDARD_SAVINGS rows with no distinguishing label at all. Their detail
// pages are keyed by the row's real database id instead — unambiguous now,
// though a link can go stale after a future scrape replaces that row
// (handled by each detail page's "not in current dataset" state).
export function productDetailHref(row: ProductRate): string | null {
  if (isFdRow(row)) {
    const bankSlug = bankSlugForApiName(row.bank_name)
    if (!bankSlug) return null
    return `/products/fixed-deposits/${fdSlug(bankSlug, row.category_code, row.tenure_value ?? 0)}`
  }
  if (row.category_code.endsWith('_SAVINGS')) return `/products/savings/${row.id}`
  return `/products/loans/${row.id}`
}
