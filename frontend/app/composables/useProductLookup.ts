// Shared "pick a real product, get its real rate" lookup used by the
// Report Issue form and the FD/Loan calculator modals — all three need
// the same thing: fetch every tagged row once, then let a Bank (+
// Category) selection narrow it down to a concrete list of real products
// with a human-readable label and rate string, instead of a free-text
// field the visitor has to fill in from memory.
import type { ProductRate } from '~/composables/useRatesApi'
import type { TaggedRow } from '~/utils/bankDirectory'
import { tenureLabelFor } from '~/utils/realProductRows'
import { fmtLkr } from '~/utils/fdCompare'

// Module-level (not inside useProductLookup) so every caller on the page
// shares one fetch — the Report Issue form and a calculator modal opened
// from the same page never need to hit the API twice for identical data.
const allRows = ref<TaggedRow[]>([])
const loaded = ref(false)
const loading = ref(false)

function tag<K extends TaggedRow['kind']>(rows: ProductRate[], kind: K): TaggedRow[] {
  return rows.map((r) => ({ ...r, kind }))
}

async function ensureProductsLoaded() {
  if (loaded.value || loading.value) return
  loading.value = true
  const { fetchFixedDeposits, fetchSavings, fetchLoans, fetchCards } = useRatesApi()
  try {
    const [fd, savings, loans, cards] = await Promise.all([
      fetchFixedDeposits(),
      fetchSavings(),
      fetchLoans(),
      fetchCards()
    ])
    allRows.value = [
      ...tag(fd, 'fd'),
      ...tag(savings, 'savings'),
      ...tag(loans, 'loans'),
      ...tag(cards, 'cards')
    ]
    loaded.value = true
  } finally {
    loading.value = false
  }
}

// Debit cards carry interest_rate: 0 as a literal fact (no interest
// mechanism) — same convention as utils/realProductRows.ts — so their
// "rate" is the annual fee instead of a misleading 0.00%.
export function productLabel(r: TaggedRow): string {
  if (r.kind === 'fd') {
    const t = tenureLabelFor(r)
    return t ? `${t} Fixed Deposit` : r.product_name
  }
  return r.product_name
}

export function productRateLabel(r: TaggedRow): string {
  if (r.category_code === 'DEBIT_CARD') {
    return r.annual_fee !== undefined ? `${fmtLkr(r.annual_fee)} annual fee` : 'No fee data'
  }
  return `${r.interest_rate.toFixed(2)}% p.a.`
}

export function useProductLookup() {
  return { allRows, loading, ensureProductsLoaded }
}
