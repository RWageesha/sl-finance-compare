// Pure financial math for the dedicated calculator pages. Standard
// formulas only — nothing here is a claim about what a specific bank
// actually offers or charges (see each calculator page's disclaimer).
import { emiPayment } from '~/utils/fdCompare'

export type FdFrequency = 'maturity' | 'monthly' | 'annual'

export interface FdResult {
  principal: number
  /** Total interest over the full term, regardless of payout frequency —
   * frequency only changes how it's distributed, not the total (simple,
   * non-compounded interest, same convention as FdCalculatorModal.vue). */
  totalInterest: number
  tax: number
  /** Total value received over the term: principal + interest − tax. */
  netTotal: number
  /** Only set for 'monthly'/'annual': the gross payout per period. */
  periodicPayout?: number
  periodicPayoutAfterTax?: number
  periodsCount?: number
}

export function calcFd(principal: number, annualRatePct: number, months: number, whtPct: number, frequency: FdFrequency): FdResult {
  const totalInterest = principal * (annualRatePct / 100) * (months / 12)
  const tax = totalInterest * (whtPct / 100)
  const netTotal = principal + totalInterest - tax
  if (frequency === 'maturity') {
    return { principal, totalInterest, tax, netTotal }
  }
  const periodsPerYear = frequency === 'monthly' ? 12 : 1
  const periodsCount = Math.max(1, Math.round((months / 12) * periodsPerYear))
  const periodicPayout = totalInterest / periodsCount
  return {
    principal,
    totalInterest,
    tax,
    netTotal,
    periodicPayout,
    periodicPayoutAfterTax: periodicPayout * (1 - whtPct / 100),
    periodsCount
  }
}

export interface AmortizationYear {
  year: number
  principalPaid: number
  interestPaid: number
  remainingBalance: number
}

// Standard reducing-balance amortization, computed month by month and
// rolled up into yearly totals — the same EMI value each month, split
// differently between principal and interest as the balance shrinks.
export function amortizationSchedule(principal: number, annualRatePct: number, months: number, maxYears = 5): AmortizationYear[] {
  const r = annualRatePct / 100 / 12
  const emi = emiPayment(principal, annualRatePct, months)
  let balance = principal
  const years: AmortizationYear[] = []
  const totalYears = Math.min(maxYears, Math.ceil(months / 12))
  let monthIndex = 0
  for (let y = 1; y <= totalYears; y++) {
    let yearPrincipal = 0
    let yearInterest = 0
    for (let m = 0; m < 12 && monthIndex < months; m++, monthIndex++) {
      const interestPortion = balance * r
      let principalPortion = emi - interestPortion
      if (principalPortion > balance) principalPortion = balance
      balance = Math.max(balance - principalPortion, 0)
      yearPrincipal += principalPortion
      yearInterest += interestPortion
    }
    years.push({ year: y, principalPaid: yearPrincipal, interestPaid: yearInterest, remainingBalance: balance })
  }
  return years
}
