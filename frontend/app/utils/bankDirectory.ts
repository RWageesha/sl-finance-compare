// Shared bank-directory data used by pages/banks.vue (the directory grid)
// and pages/banks/[slug].vue (individual profile pages).
//
// Only the three banks marked tracked: true are actually scraped
// (scraper/banks/) — the rest are listed so the directory/profile pages
// show the real shape of the Sri Lankan bank market, but they carry no
// invented statistics: no fabricated "data health" percentages, credit
// ratings, or corporate history. What we don't have, we don't make up.
import type { ProductRate } from '~/composables/useRatesApi'

export interface DirectoryBank {
  slug: string
  /** Exact bank_name value the API returns — only set for tracked banks,
   * and must match migrations/002_seed.sql exactly. */
  apiName?: string
  displayName: string
  type: 'Commercial Bank' | 'Savings Bank' | 'Licensed Commercial Bank'
  icon: 'bank' | 'savings' | 'moon'
  tracked: boolean
  /** The bank's real published rates page, taken directly from the
   * scraper's own source URL constants — only set for tracked banks. */
  sourceUrl?: string
  /** File extension of this bank's real logo in public/banks/<slug>.<ext>,
   * or undefined to fall back to the generic `icon` glyph above. A logo
   * here is just the bank's public brand mark — unrelated to whether we
   * track their rates, so untracked banks can have one too. */
  logoExt?: 'png' | 'jpeg' | 'svg'
}

export const BANK_TYPES = ['Commercial Bank', 'Savings Bank', 'Licensed Commercial Bank'] as const

export const DIRECTORY_BANKS: DirectoryBank[] = [
  {
    slug: 'hnb',
    apiName: 'Hatton National Bank',
    displayName: 'Hatton National Bank (HNB)',
    type: 'Commercial Bank',
    icon: 'bank',
    tracked: true,
    sourceUrl: 'https://www.hnb.lk/fixed-deposits-interest-rates',
    logoExt: 'png'
  },
  {
    slug: 'combank',
    apiName: 'Commercial Bank of Ceylon',
    displayName: 'Commercial Bank of Ceylon',
    type: 'Commercial Bank',
    icon: 'bank',
    tracked: true,
    sourceUrl: 'https://www.combank.lk/rates-tariff',
    logoExt: 'png'
  },
  {
    slug: 'boc',
    apiName: 'Bank of Ceylon',
    displayName: 'Bank of Ceylon (BOC)',
    type: 'Commercial Bank',
    icon: 'bank',
    tracked: true,
    sourceUrl: 'https://www.boc.lk/rates-tariff',
    logoExt: 'jpeg'
  },
  { slug: 'sampath', displayName: 'Sampath Bank', type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'ndb', displayName: 'National Development Bank (NDB)', type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'seylan', displayName: 'Seylan Bank', type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'dfcc', displayName: 'DFCC Bank', type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'peoples', displayName: "People's Bank", type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'nsb', displayName: 'National Savings Bank (NSB)', type: 'Savings Bank', icon: 'savings', tracked: false, logoExt: 'png' },
  { slug: 'panasia', displayName: 'Pan Asia Banking Corporation', type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'union', displayName: 'Union Bank of Colombo', type: 'Commercial Bank', icon: 'bank', tracked: false, logoExt: 'png' },
  { slug: 'amana', displayName: 'Amana Bank', type: 'Licensed Commercial Bank', icon: 'moon', tracked: false, logoExt: 'png' }
]

export function findDirectoryBank(slug: string): DirectoryBank | undefined {
  return DIRECTORY_BANKS.find((b) => b.slug === slug)
}

export type TaggedRow = ProductRate & { kind: 'fd' | 'savings' | 'loans' }

export interface BankStats {
  productCount: number
  categoriesTracked: number
  lastUpdated: string | null
  sourceCount: number
}
export const EMPTY_BANK_STATS: BankStats = { productCount: 0, categoriesTracked: 0, lastUpdated: null, sourceCount: 0 }

export function computeBankStats(rows: TaggedRow[], apiName: string | undefined): BankStats {
  if (!apiName) return EMPTY_BANK_STATS
  const bankRows = rows.filter((r) => r.bank_name === apiName)
  if (bankRows.length === 0) return EMPTY_BANK_STATS
  const kinds = new Set(bankRows.map((r) => r.kind))
  const sources = new Set(bankRows.map((r) => r.source_url).filter(Boolean))
  const latest = bankRows.reduce<string | null>((max, r) => (!max || r.scraped_at > max ? r.scraped_at : max), null)
  return {
    productCount: bankRows.length,
    categoriesTracked: kinds.size,
    lastUpdated: latest,
    sourceCount: sources.size
  }
}

// Groups a bank's rows into the five product categories the profile page
// shows, each with a real min–max rate range and a deep link back to the
// filtered rates table — no single "the" rate is cherry-picked, since a
// bank can have several FD/savings sub-products at very different rates.
export interface ProductGroup {
  label: string
  count: number
  minRate: number | null
  maxRate: number | null
  href: string
}

const PRODUCT_GROUP_DEFS: { label: string; tab: string; category?: string; match: (r: TaggedRow) => boolean }[] = [
  { label: 'Fixed Deposits', tab: 'fd', match: (r) => r.kind === 'fd' },
  { label: 'Savings Accounts', tab: 'savings', match: (r) => r.kind === 'savings' },
  { label: 'Housing Loans', tab: 'loans', category: 'HOUSING_LOAN', match: (r) => r.kind === 'loans' && r.category_code === 'HOUSING_LOAN' },
  { label: 'Personal Loans', tab: 'loans', category: 'PERSONAL_LOAN', match: (r) => r.kind === 'loans' && r.category_code === 'PERSONAL_LOAN' },
  { label: 'Gold Loans', tab: 'loans', category: 'GOLD_LOAN', match: (r) => r.kind === 'loans' && r.category_code === 'GOLD_LOAN' }
]

export function computeProductGroups(rows: TaggedRow[], apiName: string | undefined): ProductGroup[] {
  const bankRows = apiName ? rows.filter((r) => r.bank_name === apiName) : []
  return PRODUCT_GROUP_DEFS.map((def) => {
    const matched = bankRows.filter(def.match)
    const rates = matched.map((r) => r.interest_rate)
    let href = `/rates?tab=${def.tab}`
    if (apiName) href += `&bank=${encodeURIComponent(apiName)}`
    if (def.category) href += `&category=${def.category}`
    return {
      label: def.label,
      count: matched.length,
      minRate: rates.length ? Math.min(...rates) : null,
      maxRate: rates.length ? Math.max(...rates) : null,
      href
    }
  })
}
