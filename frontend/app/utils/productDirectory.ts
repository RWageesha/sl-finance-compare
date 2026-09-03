// Shared product-directory data used by pages/products.vue. Every entry
// marked tracked: true maps to a real category_code we scrape; Credit
// Cards and Debit Cards aren't scraped at all (fee/APR-based products, not
// interest rates), so they're listed for market-scope completeness but
// carry no invented stats — same principle as bankDirectory.ts.
export interface ProductEntry {
  label: string
  tab: 'fd' | 'savings' | 'loans'
  /** category_code filter; omitted means "the whole tab" (FD, Savings). */
  category?: string
  tracked: boolean
}

export interface ProductGroupDef {
  label: string
  items: ProductEntry[]
}

export const PRODUCT_DIRECTORY: ProductGroupDef[] = [
  {
    label: 'Deposits',
    items: [
      { label: 'Fixed Deposits', tab: 'fd', tracked: true },
      { label: 'Savings Accounts', tab: 'savings', tracked: true }
    ]
  },
  {
    label: 'Loans',
    items: [
      { label: 'Housing Loans', tab: 'loans', category: 'HOUSING_LOAN', tracked: true },
      { label: 'Personal Loans', tab: 'loans', category: 'PERSONAL_LOAN', tracked: true },
      { label: 'Vehicle Loans', tab: 'loans', category: 'LEASE', tracked: true },
      { label: 'Gold/Pawning Loans', tab: 'loans', category: 'GOLD_LOAN', tracked: true }
    ]
  },
  {
    label: 'Cards',
    items: [
      { label: 'Credit Cards', tab: 'loans', tracked: false },
      { label: 'Debit Cards', tab: 'loans', tracked: false }
    ]
  }
]

// Every product type shown on the homepage's first section has its own
// dedicated compare page now; Vehicle Loans (LEASE) doesn't, so it still
// goes to the general filtered rates table.
const DEDICATED_HREF: Record<string, string> = {
  fd: '/compare/fixed-deposits',
  'loans:HOUSING_LOAN': '/compare/housing-loans',
  'loans:PERSONAL_LOAN': '/compare/personal-loans',
  'loans:GOLD_LOAN': '/compare/gold-loans',
  savings: '/compare/savings'
}

export function productHref(item: ProductEntry): string {
  const key = item.category ? `${item.tab}:${item.category}` : item.tab
  if (DEDICATED_HREF[key]) return DEDICATED_HREF[key]
  let href = `/rates?tab=${item.tab}`
  if (item.category) href += `&category=${item.category}`
  return href
}
