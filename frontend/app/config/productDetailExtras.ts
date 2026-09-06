// Extends the real, live-API-backed product rows with the extra detail
// fields the Product Detail page's design calls for (min/max deposit,
// early withdrawal, tax, eligibility, fees) that FindRate's schema
// doesn't capture yet (see fdCompare.ts's note on what it does and
// doesn't track). Only HNB's 12-month Fixed Deposit is populated for now
// — every other product simply has no entry here, and the cards that
// would show these fields don't render, rather than showing them empty.
//
// Keyed the same way each detail page already identifies one product:
// FD rows by their existing bankSlug-categoryCode-tenureMonths slug
// (see fdCompare.ts's fdSlug()); savings/loan rows by `${group}-${id}`.
export interface ProductDetailExtras {
  minDeposit?: string
  maxDeposit?: string
  paymentFrequency?: string
  earlyWithdrawal?: string
  tax?: string
  eligibility?: string[]
  feesConditions?: string[]
  sourceLabel?: string
}

export const PRODUCT_DETAIL_EXTRAS: Record<string, ProductDetailExtras> = {
  'hnb-standard_fd-12': {
    minDeposit: 'Rs. 100,000',
    maxDeposit: 'No limit',
    paymentFrequency: 'At Maturity',
    earlyWithdrawal: 'Penalty applies',
    tax: 'WHT 5% applicable',
    eligibility: [
      'Sri Lankan citizens above 18 years of age',
      'Joint account options available for up to 3 individuals',
      'Dual Citizens and Non-Resident Sri Lankans (FCBS accounts applicable)',
      'Completed application form, National Identity Card (NIC) / Passport copy, and proof of address are mandatory'
    ],
    feesConditions: [
      'No initial processing fees apply for deposit opening',
      'Premature withdrawal penalty: rate adjusted downwards by 1.50% from the original placement rate for the actual elapsed period',
      'Withholding Tax (WHT) of 5.0% automatically deducted from interest pay-out at maturity'
    ],
    sourceLabel: 'Official HNB Website'
  }
}

export function findDetailExtras(key: string): ProductDetailExtras | undefined {
  return PRODUCT_DETAIL_EXTRAS[key]
}
