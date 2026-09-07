// Maps real API rows (ProductRate, tagged by product kind — see
// bankDirectory.ts's TaggedRow) onto the generic SampleRow shape that
// FilterBar/ProductResultList/CompareResultList already know how to
// render, for the Product Directory (/products/[slug]) and Compare
// (/compare/[slug]) pages. Fixed Deposits, Savings, and the three loan
// categories (Housing/Personal/Gold) all have real scraped data behind
// them; Credit/Debit Cards don't (no scraper tracks them anywhere in this
// codebase), so realRowsFor returns [] for those rather than inventing
// rows — callers show an honest "not tracked yet" state instead.
import type { ProductRate } from '~/composables/useRatesApi'
import type { SampleRow } from '~/config/productTypes'
import { productDetailHref } from '~/utils/fdCompare'
import { fmtTenure, formatCategoryLabel, fmtRelativeDate } from '~/utils/format'

export type Kind = 'fd' | 'savings' | 'loans'
export type TaggedRow = ProductRate & { kind: Kind }

export function tenureLabelFor(r: ProductRate): string {
  return r.tenure_label || (r.tenure_value ? fmtTenure(r.tenure_value) : '')
}

function titleFor(r: TaggedRow): string {
  if (r.kind === 'fd') {
    const t = tenureLabelFor(r)
    return t ? `${t} Fixed Deposit` : formatCategoryLabel(r.category_code)
  }
  return r.product_name || formatCategoryLabel(r.category_code)
}

function toSampleRow(r: TaggedRow): SampleRow {
  return {
    id: String(r.id),
    bank: r.bank_name,
    product: titleFor(r),
    rate: `${r.interest_rate.toFixed(2)}%`,
    rateValue: r.interest_rate,
    tenure: tenureLabelFor(r),
    tenureMonths: r.tenure_value,
    category: formatCategoryLabel(r.category_code),
    verified: fmtRelativeDate(r.scraped_at),
    detailHref: productDetailHref(r) ?? undefined
  }
}

const LOAN_CATEGORY_BY_SLUG: Record<string, string> = {
  'housing-loans': 'HOUSING_LOAN',
  'personal-loans': 'PERSONAL_LOAN',
  'gold-loans': 'GOLD_LOAN'
}

// The only product-type slugs backed by a real scraper anywhere in this
// codebase — used by the Directory/Compare pages to decide whether to
// fetch+render real rows or show the honest "not tracked" empty state.
export const LIVE_SLUGS = new Set(['fixed-deposits', 'savings-accounts', 'housing-loans', 'personal-loans', 'gold-loans'])

export function realRowsFor(slug: string, allRows: TaggedRow[]): SampleRow[] {
  if (slug === 'fixed-deposits') return allRows.filter((r) => r.kind === 'fd').map(toSampleRow)
  if (slug === 'savings-accounts') return allRows.filter((r) => r.kind === 'savings').map(toSampleRow)
  const category = LOAN_CATEGORY_BY_SLUG[slug]
  if (category) return allRows.filter((r) => r.kind === 'loans' && r.category_code === category).map(toSampleRow)
  return []
}
