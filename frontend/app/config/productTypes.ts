// Single source of truth for every /products/[slug] directory page AND
// every /compare/[slug] page. FilterBar/ProductResultList/CompareResultList
// render entirely from these definitions — adding a new product type (or
// changing an existing one) never touches those component files, only
// this config.
//
// Every row carries a stable `id` and is defined exactly once per logical
// product — Directory and Compare pages read the SAME row array for a
// given type (housing-loans is the one deliberate exception: its Compare
// view needs a flat-rate number to compute an honest EMI, which an
// "AWPLR + 2.5%" disclosed rate can't give it — see the note below). A
// row's `id` is how /products/sample/[id] resolves a working "View
// Details" page for every row shown anywhere in the app, even though none
// of this config is backed by the live API yet.
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
  payment?: string
  tier?: string
  annualFee?: string
  atmLimit?: string
  terms?: string
  verified?: string
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

// --- Fixed Deposits: one shared row array — Directory and Compare show
// the same products, just through different columns. ---
const FD_ROWS: SampleRow[] = [
  { id: 'hnb-fd-12', bank: 'HNB', product: '12-Month Fixed Deposit', rate: '8.25%', minDeposit: 'Rs. 25,000', payment: 'At Maturity', verified: '3h ago' },
  { id: 'combank-fd-12', bank: 'Commercial Bank', product: 'Regular Fixed Deposit', rate: '8.10%', minDeposit: 'Rs. 50,000', payment: 'At Maturity', verified: '4h ago' },
  { id: 'sampath-fd-12', bank: 'Sampath Bank', product: 'Sarasa Fixed Deposit', rate: '8.00%', minDeposit: 'Rs. 25,000', payment: 'Monthly', verified: '4h ago' },
  { id: 'boc-fd-12', bank: 'Bank of Ceylon', product: '12-Month Standard Deposit', rate: '7.90%', minDeposit: 'Rs. 10,000', payment: 'At Maturity', verified: '5h ago' },
  { id: 'seylan-fd-12', bank: 'Seylan Bank', product: 'Seylan 12-Month FD', rate: '7.75%', minDeposit: 'Rs. 10,000', payment: 'Annually', verified: '5h ago' },
  { id: 'peoples-fd-12', bank: "People's Bank", product: "People's Term Deposit", rate: '7.50%', minDeposit: 'Rs. 5,000', payment: 'At Maturity', verified: '5h ago' }
]

// --- Savings Accounts: same idea — one shared row array. ---
const SAVINGS_ROWS: SampleRow[] = [
  { id: 'hnb-savings-senior', bank: 'HNB', product: 'HNB Singithi Senior Savings', rate: '3.00% - 4.50%', tiered: true, minBalance: 'Rs. 1,000', category: 'Senior' },
  { id: 'combank-savings-children', bank: 'Commercial Bank', product: "Arunalu Children's Savings", rate: '3.25%', tiered: false, minBalance: 'Rs. 500', category: "Children's" },
  { id: 'sampath-savings-digital', bank: 'Sampath Bank', product: 'Sampath Vishwa Digital Savings', rate: '2.75% - 3.75%', tiered: true, minBalance: 'Rs. 0', category: 'Digital' },
  { id: 'peoples-savings-women', bank: "People's Bank", product: "Vanitha Vasana Women's Savings", rate: '3.00%', tiered: false, minBalance: 'Rs. 1,000', category: "Women's" },
  { id: 'boc-savings-regular', bank: 'Bank of Ceylon', product: 'BOC Regular Savings', rate: '2.50% - 3.25%', tiered: true, minBalance: 'Rs. 5,000', category: 'Regular' },
  { id: 'nsb-savings-regular', bank: 'National Savings Bank', product: 'NSB Savings', rate: '2.75%', tiered: false, minBalance: 'Rs. 1,000', category: 'Regular' }
]

// --- Housing Loans: deliberately TWO representations of the same six
// products, sharing ids so /products/sample/[id] can merge them into one
// honest detail page. Directory shows each bank's real disclosed rate
// (AWPLR-linked, the way these loans actually work); Compare needs a flat
// number to compute an EMI, which "AWPLR + 2.5%" can't honestly give
// without a real AWPLR figure we don't track, so it uses a separate,
// clearly-labeled illustrative flat rate instead of silently pretending
// the disclosed rate was used.
const HOUSING_DIRECTORY_ROWS: SampleRow[] = [
  { id: 'hnb-housing-dream', bank: 'HNB', product: 'Dream Home Loan', rate: 'AWPLR + 2.5%', maxAmount: 'Rs. 50M', tenure: '25 Years', verified: '6h ago' },
  { id: 'combank-housing-cbc', bank: 'Commercial Bank', product: 'CBC Home Loan', rate: 'AWPLR + 2.0%', maxAmount: 'Rs. 75M', tenure: '20 Years', verified: '4h ago' },
  { id: 'sampath-housing-sevana', bank: 'Sampath Bank', product: 'Sevana Housing Loan', rate: 'AWPLR + 2.0%', maxAmount: 'Rs. 40M', tenure: '25 Years', verified: '3h ago' },
  { id: 'ndb-housing-home', bank: 'NDB', product: 'NDB Home Loan', rate: 'Max 14.00%', maxAmount: 'Rs. 10M', tenure: '20 Years', verified: '8h ago' },
  { id: 'boc-housing-boc', bank: 'Bank of Ceylon', product: 'BOC Housing Loan', rate: 'AWPLR + 2.5%', maxAmount: 'Rs. 60M', tenure: '25 Years', verified: '5h ago' },
  { id: 'peoples-housing', bank: "People's Bank", product: 'Housing Loan', rate: 'AWPLR + 3.0%', maxAmount: 'Rs. 30M', tenure: '20 Years', verified: '7h ago' }
]
const HOUSING_COMPARE_ROWS: SampleRow[] = [
  { id: 'hnb-housing-dream', bank: 'HNB', product: 'Dream Home Loan', rate: '13.50%', rateType: 'Reducing', maxAmount: 'Rs. 50M', verified: '6h ago' },
  { id: 'combank-housing-cbc', bank: 'Commercial Bank', product: 'CBC Home Loan', rate: '13.00%', rateType: 'Reducing', maxAmount: 'Rs. 75M', verified: '4h ago' },
  { id: 'sampath-housing-sevana', bank: 'Sampath Bank', product: 'Sevana Housing Loan', rate: '13.25%', rateType: 'Reducing', maxAmount: 'Rs. 40M', verified: '3h ago' },
  { id: 'ndb-housing-home', bank: 'NDB', product: 'NDB Home Loan', rate: '14.00%', rateType: 'Reducing', maxAmount: 'Rs. 10M', verified: '8h ago' },
  { id: 'boc-housing-boc', bank: 'Bank of Ceylon', product: 'BOC Housing Loan', rate: '13.75%', rateType: 'Reducing', maxAmount: 'Rs. 60M', verified: '5h ago' },
  { id: 'peoples-housing', bank: "People's Bank", product: 'Housing Loan', rate: '14.25%', rateType: 'Reducing', maxAmount: 'Rs. 30M', verified: '7h ago' }
]

// --- Personal Loans: one shared row array (the one AWPLR row here gets
// an explicit rateValue override instead of a second dataset, since it's
// a single row rather than the whole type). ---
const PERSONAL_ROWS: SampleRow[] = [
  { id: 'hnb-personal', bank: 'HNB', product: 'HNB Personal Loan', rate: '14.50%', rateType: 'Reducing', maxAmount: 'Rs. 5,000,000', tenure: '5 Years', verified: '2h ago' },
  { id: 'combank-personal-flexi', bank: 'Commercial Bank', product: 'ComBank FlexiLoan', rate: 'AWPLR + 2.00%', rateValue: 13.5, rateType: 'Floating', maxAmount: 'Rs. 10,000,000', tenure: '5 Years', verified: '3h ago' },
  { id: 'sampath-personal-salary', bank: 'Sampath Bank', product: 'Sampath Salary Loan', rate: '13.75%', rateType: 'Reducing', maxAmount: 'Rs. 3,000,000', tenure: '5 Years', verified: '3h ago' },
  { id: 'ndb-personal-advance', bank: 'NDB', product: 'NDB Personal Advance', rate: '15.25%', rateType: 'Reducing', maxAmount: 'Rs. 7,500,000', tenure: '5 Years', verified: '4h ago' },
  { id: 'seylan-personal-cash', bank: 'Seylan Bank', product: 'Seylan Quick Cash', rate: '14.00%', rateType: 'Reducing', maxAmount: 'Rs. 4,000,000', tenure: '5 Years', verified: '5h ago' },
  { id: 'boc-personal', bank: 'Bank of Ceylon', product: 'BOC Personal Loan', rate: '15.75%', rateType: 'Reducing', maxAmount: 'Rs. 5,000,000', tenure: '5 Years', verified: '6h ago' }
]

const GOLD_ROWS: SampleRow[] = [
  { id: 'peoples-gold-swarna', bank: "People's Bank", product: 'Swarna Pradeepa', rate: '15.50%', terms: 'Penalty after 12mo', verified: '4h ago' },
  { id: 'sampath-gold-randiriya', bank: 'Sampath Bank', product: 'Randiriya', rate: '16.00%', terms: 'Standard terms', verified: '6h ago' },
  { id: 'hnb-gold', bank: 'HNB', product: 'Gold Loan', rate: '16.50%', terms: 'Standard terms', verified: '5h ago' },
  { id: 'boc-gold-pawning', bank: 'Bank of Ceylon', product: 'Pawning Advance', rate: '15.75%', terms: 'Standard terms', verified: '7h ago' },
  { id: 'combank-gold', bank: 'Commercial Bank', product: 'Gold Loan', rate: '16.25%', terms: 'Standard terms', verified: '3h ago' }
]

const CREDIT_CARD_ROWS: SampleRow[] = [
  { id: 'hnb-card-signature', bank: 'HNB', product: 'Signature Card', tier: 'Signature', annualFee: 'Rs. 4,000', rate: '28.00%', verified: '5h ago' },
  { id: 'sampath-card-platinum', bank: 'Sampath Bank', product: 'Platinum Card', tier: 'Platinum', annualFee: 'Rs. 2,900', rate: '24.00%', verified: '3h ago' },
  { id: 'combank-card-gold', bank: 'Commercial Bank', product: 'Gold Card', tier: 'Gold', annualFee: 'Rs. 2,500', rate: '36.00%', verified: '6h ago' },
  { id: 'boc-card-classic', bank: 'Bank of Ceylon', product: 'Visa Classic', tier: 'Classic', annualFee: 'Rs. 1,500', rate: '30.00%', verified: '8h ago' },
  { id: 'hnb-card-classic', bank: 'HNB', product: 'Classic Card', tier: 'Classic', annualFee: 'Rs. 1,000', rate: '28.00%', verified: '4h ago' },
  { id: 'sampath-card-world', bank: 'Sampath Bank', product: 'World Card', tier: 'World', annualFee: 'Rs. 4,000', rate: '24.00%', verified: '2h ago' }
]

const DEBIT_CARD_ROWS: SampleRow[] = [
  { id: 'combank-debit-visa', bank: 'Commercial Bank', product: 'Visa Debit', annualFee: 'Rs. 300', atmLimit: 'Rs. 300,000', verified: '5h ago' },
  { id: 'hnb-debit-classic', bank: 'HNB', product: 'Classic Debit', annualFee: 'Rs. 250', atmLimit: 'Rs. 250,000', verified: '6h ago' },
  { id: 'sampath-debit-visa', bank: 'Sampath Bank', product: 'Visa Debit', annualFee: 'Rs. 300', atmLimit: 'Rs. 300,000', verified: '4h ago' },
  { id: 'boc-debit', bank: 'Bank of Ceylon', product: 'Debit Card', annualFee: 'Rs. 200', atmLimit: 'Rs. 200,000', verified: '7h ago' },
  { id: 'peoples-debit', bank: "People's Bank", product: 'Debit Card', annualFee: 'Rs. 250', atmLimit: 'Rs. 250,000', verified: '6h ago' }
]

export const PRODUCT_TYPES: Record<string, ProductTypeConfig> = {
  'fixed-deposits': {
    slug: 'fixed-deposits',
    switcherLabel: 'Fixed Deposits',
    eyebrow: 'FIXED DEPOSITS',
    title: 'Fixed Deposit Rates',
    subtitle: 'Compare fixed deposit rates from every licensed bank in Sri Lanka. Find the highest yield for your savings timeline.',
    filters: [
      { type: 'input', key: 'amount', label: 'Deposit Amount', placeholder: 'Deposit Amount: Rs. 1,000,000' },
      { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['Any', '1 Month', '3 Months', '6 Months', '12 Months', '24 Months', '36 Months', '60 Months'] },
      { type: 'dropdown', key: 'payment', label: 'Interest Payment', options: ['Any', 'At Maturity', 'Monthly', 'Annually'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare FD',
    compareHref: '/compare/fixed-deposits',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'minDeposit', label: 'Min. Deposit', type: 'text' },
      { key: 'payment', label: 'Payment', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: FD_ROWS,
    compare: {
      heading: 'Compare Fixed Deposit Rates',
      subtitle: 'Estimate and compare rates of return across licensed retail banks in Sri Lanka. Fill in the deposit variables to project net maturity value.',
      filters: [
        { type: 'input', key: 'amount', label: 'Deposit Amount (LKR)', placeholder: 'Rs. 1,000,000' },
        { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['12 Months', '1 Month', '3 Months', '6 Months', '24 Months', '36 Months', '60 Months'] },
        { type: 'dropdown', key: 'payment', label: 'Interest Payment', options: ['At Maturity', 'Monthly', 'Annually'] }
      ],
      defaultLabel: 'Compare FD',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'maturity', label: 'Est. Maturity Value', type: 'computed', compute: 'maturity', align: 'right' },
        { key: 'minDeposit', label: 'Min. Deposit', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: FD_ROWS
    }
  },

  'savings-accounts': {
    slug: 'savings-accounts',
    switcherLabel: 'Savings',
    eyebrow: 'SAVINGS ACCOUNTS',
    title: 'Savings Account Rates',
    subtitle: 'Compare savings account interest rates and balance tiers across Sri Lankan banks. Maximize daily earnings.',
    filters: [
      { type: 'dropdown', key: 'accountType', label: 'Account Type', options: ['Any', 'Senior', "Children's", "Women's", 'Digital', 'Regular'] },
      { type: 'dropdown', key: 'minBalance', label: 'Minimum Balance', options: ['Any', 'Under Rs. 5,000', 'Rs. 5,000 – 50,000', 'Above Rs. 50,000'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare Savings',
    compareHref: '/compare/savings-accounts',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate Range (P.A.)', type: 'rateRange', align: 'right' },
      { key: 'minBalance', label: 'Min. Balance', type: 'text' },
      { key: 'category', label: 'Category', type: 'text' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: SAVINGS_ROWS,
    compare: {
      heading: 'Compare Savings Account Rates',
      subtitle: 'Compare interest rates and balance tiers across Sri Lankan savings accounts.',
      filters: [
        { type: 'dropdown', key: 'accountType', label: 'Account Type', options: ['Any', 'Senior', "Children's", "Women's", 'Digital', 'Regular'] },
        { type: 'dropdown', key: 'minBalance', label: 'Minimum Balance', options: ['Any', 'Under Rs. 5,000', 'Rs. 5,000 – 50,000', 'Above Rs. 50,000'] }
      ],
      defaultLabel: 'Compare Savings',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate Range (P.A.)', type: 'rateRange', align: 'right' },
        { key: 'minBalance', label: 'Min. Balance', type: 'text' },
        { key: 'category', label: 'Category', type: 'text' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: SAVINGS_ROWS
    }
  },

  'housing-loans': {
    slug: 'housing-loans',
    switcherLabel: 'Housing Loans',
    eyebrow: 'HOUSING LOANS',
    title: 'Housing Loan Rates',
    subtitle: 'Compare housing loan rates and terms from licensed banks and financial institutions in Sri Lanka.',
    filters: [
      { type: 'input', key: 'amount', label: 'Loan Amount', placeholder: 'Loan Amount' },
      { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['Any', '5 Years', '10 Years', '15 Years', '20 Years', '25 Years'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare Loan',
    compareHref: '/compare/housing-loans',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'maxAmount', label: 'Max Loan Amount', type: 'text' },
      { key: 'tenure', label: 'Tenure', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: HOUSING_DIRECTORY_ROWS,
    // See HOUSING_COMPARE_ROWS' comment above: same six products as the
    // directory list (matching ids), but a clean flat rate instead of the
    // AWPLR-linked disclosed rate, since an EMI can't be honestly computed
    // from "AWPLR + 2.0%" without a real AWPLR figure we don't track.
    compare: {
      heading: 'Compare Housing Loan Rates',
      subtitle: 'Estimate monthly repayments and compare housing loan terms across Sri Lankan banks.',
      filters: [
        { type: 'input', key: 'amount', label: 'Loan Amount (LKR)', placeholder: 'Rs. 8,000,000' },
        { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['15 Years', '5 Years', '10 Years', '20 Years', '25 Years'] }
      ],
      defaultLabel: 'Compare Loan',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'emi', label: 'Est. Monthly EMI', type: 'computed', compute: 'emi', align: 'right' },
        { key: 'maxAmount', label: 'Max Loan Amount', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: HOUSING_COMPARE_ROWS
    }
  },

  'personal-loans': {
    slug: 'personal-loans',
    switcherLabel: 'Personal Loans',
    eyebrow: 'PERSONAL LOANS',
    title: 'Personal Loan Rates',
    subtitle: 'Compare unsecured personal loan rates and terms from banks across Sri Lanka.',
    filters: [
      { type: 'input', key: 'amount', label: 'Loan Amount', placeholder: 'Loan Amount' },
      { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['Any', '1 Year', '2 Years', '3 Years', '5 Years', '7 Years'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare Loan',
    compareHref: '/compare/personal-loans',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Product', type: 'text' },
      { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
      { key: 'maxAmount', label: 'Max Loan Amount', type: 'text' },
      { key: 'tenure', label: 'Tenure', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: PERSONAL_ROWS,
    compare: {
      heading: 'Compare Personal Loan Rates',
      subtitle: 'Estimate monthly repayments and compare loan terms across Sri Lankan banks.',
      filters: [
        { type: 'input', key: 'amount', label: 'Loan Amount (LKR)', placeholder: 'Rs. 2,000,000' },
        { type: 'dropdown', key: 'tenure', label: 'Tenure', options: ['3 Years', '1 Year', '2 Years', '5 Years', '7 Years'] }
      ],
      defaultLabel: 'Compare Loan',
      updateLabel: 'Update Results',
      prefillMessageTemplate: "Showing plans similar to {bank}'s {product} at {rate}% p.a.",
      columns: [
        { key: 'bank', label: 'Bank', type: 'bank' },
        { key: 'product', label: 'Product', type: 'text' },
        { key: 'rate', label: 'Rate (P.A.)', type: 'rate', align: 'right' },
        { key: 'emi', label: 'Est. Monthly EMI', type: 'computed', compute: 'emi', align: 'right' },
        { key: 'maxAmount', label: 'Max Loan Amount', type: 'text' },
        { key: 'verified', label: 'Verification', type: 'verification' },
        { key: 'action', label: 'Action', type: 'action' }
      ],
      rows: PERSONAL_ROWS
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
      { key: 'terms', label: 'Terms', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: GOLD_ROWS
  },

  'credit-cards': {
    slug: 'credit-cards',
    switcherLabel: 'Credit Cards',
    eyebrow: 'CREDIT CARDS',
    title: 'Credit Card Rates & Fees',
    subtitle: 'Compare annual fees, interest rates, and rewards across credit cards from Sri Lankan banks.',
    filters: [
      { type: 'dropdown', key: 'tier', label: 'Tier', options: ['Any', 'Classic', 'Gold', 'Platinum', 'Signature', 'World'] },
      { type: 'dropdown', key: 'annualFeeRange', label: 'Annual Fee Range', options: ['Any', 'Under Rs. 2,000', 'Rs. 2,000 – 5,000', 'Above Rs. 5,000'] },
      { type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }
    ],
    compareLabel: 'Compare Cards',
    compareHref: '#',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Card Name', type: 'text' },
      { key: 'tier', label: 'Tier', type: 'text' },
      { key: 'annualFee', label: 'Annual Fee', type: 'text' },
      // Same "rate" cell renderer as FD/Loans — only the column label
      // changes, which is all "APR vs Rate (P.A.)" ever needed to be.
      { key: 'rate', label: 'APR', type: 'rate', align: 'right' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: CREDIT_CARD_ROWS
  },

  'debit-cards': {
    slug: 'debit-cards',
    switcherLabel: 'Debit Cards',
    eyebrow: 'DEBIT CARDS',
    title: 'Debit Card Fees',
    subtitle: 'Compare debit card issuance, annual, and transaction fees across Sri Lankan banks.',
    filters: [{ type: 'multiselect', key: 'bank', label: 'Bank', placeholder: 'All Banks' }],
    compareLabel: 'Compare Debit Cards',
    compareHref: '#',
    columns: [
      { key: 'bank', label: 'Bank', type: 'bank' },
      { key: 'product', label: 'Card Name', type: 'text' },
      { key: 'annualFee', label: 'Annual Fee', type: 'text' },
      { key: 'atmLimit', label: 'ATM Daily Limit', type: 'text' },
      { key: 'verified', label: 'Verification', type: 'verification' },
      { key: 'action', label: 'Action', type: 'action' }
    ],
    rows: DEBIT_CARD_ROWS
  }
}
