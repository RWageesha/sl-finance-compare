// Single source of truth for every /products/[slug] directory page AND
// every /compare/[slug] page. FilterBar/ProductResultList/CompareResultList
// render entirely from these definitions — adding a new product type (or
// changing an existing one) never touches those component files, only
// this config.
//
// `rows`/`compare.rows` are always [] here — for the five product types a
// scraper actually backs (fixed-deposits, savings-accounts, and the three
// loan categories), the Directory/Compare pages fetch real data at
// runtime and map it onto this shape via utils/realProductRows.ts, keyed
// by each row's real database `id` (see fdCompare.ts's productDetailHref
// for how that resolves to a working detail page). Credit/debit cards
// have no scraper anywhere in this codebase, so their `live: false` tells
// those pages to show an honest "not tracked yet" state instead of ever
// rendering a product row for them. Columns/filters below only reference
// fields the real schema actually has — no minimum-deposit, payout
// frequency, or max-loan-amount data exists anywhere, so none of those
// are faked here (same convention pages/banks/[slug].vue already follows).
export type FilterFieldType = 'input' | 'dropdown' | 'multiselect'

export interface FilterField {
  type: FilterFieldType
  key: string
  label: string
  placeholder?: string
  /** dropdown/multiselect only — first option is the "no filter" default. */
  options?: string[]
}

export type ColumnType = 'bank' | 'text' | 'rate' | 'rateRange' | 'computed' | 'verification' | 'action'

export interface ColumnDef {
  key: string
  label: string
  type: ColumnType
  align?: 'left' | 'right'
  /** 'computed' columns only — which formula to run against the current
   * amount/tenure filter values (both live in ~/utils/fdCompare.ts,
   * shared with the FD and Loan EMI calculators — not reimplemented here). */
  compute?: 'maturity' | 'emi'
}

// A loose bag of fields keyed by whatever a product type's columns need —
// 'id' and 'bank' are the only keys every row is guaranteed to have.
export interface SampleRow {
  id: string
  bank: string
  product?: string
  action?: string
  rate?: string
  rateValue?: number
  rateType?: string
  tiered?: boolean
  minDeposit?: string
  minBalance?: string
  category?: string
  maxAmount?: string
  tenure?: string
  /** Real tenure in months, for the Directory page's Tenure filter — kept
   * separate from `tenure` (a display string) since real tenure labels
   * vary in wording bank to bank ("01 month" vs "12 Months (Triple A)")
   * in a way that can't be exact-string-matched against a filter option. */
  tenureMonths?: number
  payment?: string
  tier?: string
  annualFee?: string
  atmLimit?: string
  terms?: string
  verified?: string
  /** Real rows only — routes "View Details" to the real product detail
   * page (see utils/realProductRows.ts) instead of the mock /products/
   * sample/[id] page. */
  detailHref?: string
  [key: string]: string | number | boolean | undefined
}
export type ProductRow = SampleRow
export type CompareRow = SampleRow

export interface CompareConfig {
  heading: string
  subtitle: string
  filters: FilterField[]
  columns: ColumnDef[]
  defaultLabel: string
  updateLabel: string
  /** e.g. "Showing plans similar to {bank}'s {product} at {rate}% p.a." */
  prefillMessageTemplate: string
  rows: CompareRow[]
}

export interface ProductTypeConfig {
  slug: string
  switcherLabel: string
  eyebrow: string
  title: string
  subtitle: string
  filters: FilterField[]
  compareLabel: string
  // The real /compare/* page for this type, where one already exists
  // (fixed-deposits, savings, gold-loans, and the two loan types all
  // predate this config). Credit and debit cards have no compare page
  // yet, so their button is a visible but inert '#' for now.
  compareHref: string
  columns: ColumnDef[]
  rows: ProductRow[]
  /** False only for credit-cards/debit-cards — no scraper tracks either
   * anywhere in this codebase, so the Directory/Compare pages show an
   * honest "not tracked yet" state for them instead of fetching rows. */
  live: boolean
  /** Only set for types that have a real /compare/[slug] page (currently
   * fixed-deposits, savings-accounts, housing-loans, personal-loans). */
  compare?: CompareConfig
}

// Order here is the order the ProductSwitcher pills render in.
export const PRODUCT_TYPE_ORDER = [
  'fixed-deposits',
  'savings-accounts',
  'housing-loans',
  'personal-loans',
  'gold-loans',
  'credit-cards',
  'debit-cards'
] as const

// No static row data below — fixed-deposits, savings-accounts, and the
// three loan categories are `live: true` and get their rows from the real
// API at runtime (see utils/realProductRows.ts). credit-cards/debit-cards
// are `live: false` and never render a rows array at all.

export const PRODUCT_TYPES: Record<string, ProductTypeConfig> = {
  'fixed-deposits': {
    slug: 'fixed-deposits',
    switcherLabel: 'Fixed Deposits',
    eyebrow: 'FIXED DEPOSITS',
    title: 'Fixed Deposit Rates',
    subtitle: 'Compare fixed deposit rates from every licensed bank in Sri Lanka. Find the highest yield for your savings timeline.',
    filters: [
      { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['Any', '1 Month', '3 Months', '6 Months', '12 Months', '24 Months', '36 Months', '60 Months'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare FD',
    compareHref: '/compare/fixed-deposits',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'category', label: 'Category', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: [],
    live: true,
    compare: {
      heading: 'Compare Fixed Deposit Rates',
      subtitle: 'Estimate and compare rates of return across licensed retail banks in Sri Lanka. Fill in the deposit variables to project net maturity value.',
      filters: [
        { type: 'input', key: 'amount', label: 'Deposit Amount (LKR)', placeholder: 'Rs. 1,000,000' },
        { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['12 Months', '1 Month', '3 Months', '6 Months', '24 Months', '36 Months', '60 Months'] }
      ],
      defaultLabel: 'Compare FD',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'maturity', label: 'Est. Maturity Value', type: 'computed', compute: 'maturity', align: 'right' },
        { key: 'category', label: 'Category', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: []
    }
  },

  'savings-accounts': {
    slug: 'savings-accounts',
    switcherLabel: 'Savings',
    eyebrow: 'SAVINGS ACCOUNTS',
    title: 'Savings Account Rates',
    subtitle: 'Compare savings account interest rates across Sri Lankan banks. Maximize daily earnings.',
    filters: [
      { type: 'dropdown', key: 'category', label: 'Account Type', options: ['Any', 'Standard Savings', 'Senior Savings', 'Teen Savings', 'Womens Savings', 'Minor Savings'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare Savings',
    compareHref: '/compare/savings-accounts',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'category', label: 'Category', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: [],
    live: true,
    compare: {
      heading: 'Compare Savings Account Rates',
      subtitle: 'Compare interest rates across Sri Lankan savings accounts.',
      filters: [
        { type: 'dropdown', key: 'category', label: 'Account Type', options: ['Any', 'Standard Savings', 'Senior Savings', 'Teen Savings', 'Womens Savings', 'Minor Savings'] }
      ],
      defaultLabel: 'Compare Savings',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'category', label: 'Category', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: []
    }
  },

  'housing-loans': {
    slug: 'housing-loans',
    switcherLabel: 'Housing Loans',
    eyebrow: 'HOUSING LOANS',
    title: 'Housing Loan Rates',
    subtitle: 'Compare housing loan rates from licensed banks and financial institutions in Sri Lanka.',
    // No tenure filter — real loan tenure comes as free-text bank wording
    // ("Maximum up to 15 years"), not a clean month count to match a
    // dropdown against.
    filters: [{ type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }],
    compareLabel: 'Compare Loan',
    compareHref: '/compare/housing-loans',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'tenure', label: 'Terms', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: [],
    live: true,
    compare: {
      heading: 'Compare Housing Loan Rates',
      subtitle: 'Estimate monthly repayments and compare housing loan rates across Sri Lankan banks.',
      filters: [{ type: 'input', key: 'amount', label: 'Loan Amount (LKR)', placeholder: 'Rs. 8,000,000' }],
      defaultLabel: 'Compare Loan',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'emi', label: 'Est. Monthly EMI (12mo)', type: 'computed', compute: 'emi', align: 'right' },
        { key: 'tenure', label: 'Terms', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: []
    }
  },

  'personal-loans': {
    slug: 'personal-loans',
    switcherLabel: 'Personal Loans',
    eyebrow: 'PERSONAL LOANS',
    title: 'Personal Loan Rates',
    subtitle: 'Compare unsecured personal loan rates from banks across Sri Lanka.',
    filters: [{ type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }],
    compareLabel: 'Compare Loan',
    compareHref: '/compare/personal-loans',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'tenure', label: 'Terms', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: [],
    live: true,
    compare: {
      heading: 'Compare Personal Loan Rates',
      subtitle: 'Estimate monthly repayments and compare loan rates across Sri Lankan banks.',
      filters: [{ type: 'input', key: 'amount', label: 'Loan Amount (LKR)', placeholder: 'Rs. 2,000,000' }],
      defaultLabel: 'Compare Loan',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'emi', label: 'Est. Monthly EMI (3yr)', type: 'computed', compute: 'emi', align: 'right' },
        { key: 'tenure', label: 'Terms', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: []
    }
  },

  'gold-loans': {
    slug: 'gold-loans',
    switcherLabel: 'Gold Loans',
    eyebrow: 'GOLD LOANS',
    title: 'Gold Loan (Pawning) Rates',
    subtitle: 'Compare gold-backed loan rates from Sri Lankan banks — advances against gold jewelry.',
    // Deliberately no amount/tenure calculator fields — there's no
    // loan-to-value data to compute against, so none is faked here.
    filters: [{ type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }],
    compareLabel: 'Compare Gold Loan',
    compareHref: '/compare/gold-loans',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'tenure', label: 'Terms', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: [],
    live: true
  },

  'credit-cards': {
    slug: 'credit-cards',
    switcherLabel: 'Credit Cards',
    eyebrow: 'CREDIT CARDS',
    title: 'Credit Card Rates & Fees',
    subtitle: 'Compare annual fees, interest rates, and rewards across credit cards from Sri Lankan banks.',
    filters: [],
    compareLabel: 'Compare Cards',
    compareHref: '#',
    columns: [],
    rows: [],
    live: false
  },

  'debit-cards': {
    slug: 'debit-cards',
    switcherLabel: 'Debit Cards',
    eyebrow: 'DEBIT CARDS',
    title: 'Debit Card Fees',
    subtitle: 'Compare debit card issuance, annual, and transaction fees across Sri Lankan banks.',
    filters: [],
    compareLabel: 'Compare Debit Cards',
    compareHref: '#',
    columns: [],
    rows: [],
    live: false
  }
}
