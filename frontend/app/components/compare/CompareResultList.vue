<script setup lang="ts">
import type { ColumnDef, CompareRow } from '~/config/productTypes'
import { bankColor, bankInitial, bankKey } from '~/utils/bankColors'

const props = defineProps<{
  columns: ColumnDef[]
  rows: CompareRow[]
  amount: number
  tenureMonths: number
  /** FD only: every row shares one payout-frequency sub-label under the
   * rate (the currently-selected Interest Payment filter value). Loan
   * rows carry their own per-row rateType ("Reducing"/"Floating")
   * instead, which takes precedence when present. */
  paymentLabel?: string
  /** Prefilled-from-product-page match, for the highlight + "Your
   * Selection" badge — null when this page was opened with no prefill. */
  highlight: { bank: string; product: string } | null
}>()

const selected = ref<Set<string>>(new Set())
function toggleSelected(id: string) {
  selected.value.has(id) ? selected.value.delete(id) : selected.value.add(id)
  selected.value = new Set(selected.value)
}

// Matches by bank alone, not bank+product: every one of these mock
// Compare datasets has exactly one row per bank, and the incoming
// fromProduct string comes from whichever REAL product a live-data page
// (Bank Profile, Product Detail) picked — which rarely matches this
// dataset's one illustrative product name for that bank. Bank-only
// matching is unambiguous here and actually highlights the right row.
function isHighlighted(row: CompareRow): boolean {
  if (!props.highlight) return false
  return bankKey(row.bank) === bankKey(props.highlight.bank)
}

// A row's rate is a display string ("8.25%" or "AWPLR + 2.00%"); the
// numeric value used for computation is either an explicit override
// (the one AWPLR row) or parsed straight off the display string.
function numericRate(row: CompareRow): number {
  if (row.rateValue !== undefined) return row.rateValue
  const n = Number.parseFloat(row.rate)
  return Number.isFinite(n) ? n : 0
}

function computedValue(row: CompareRow, col: ColumnDef): string {
  // Always the filter's live tenure, not a per-row default — the whole
  // point of this calculator is "at MY amount/tenure, what would each
  // bank's rate yield," so every row must be projected over the same
  // tenure to stay a fair comparison.
  const months = props.tenureMonths
  const rate = numericRate(row)
  if (col.compute === 'maturity') return fmtLkr(maturityValue(props.amount, rate, months))
  if (col.compute === 'emi') return fmtLkr(emiPayment(props.amount, rate, months))
  return ''
}

const bankCol = computed(() => props.columns.find((c) => c.type === 'bank'))
const productCol = computed(() => props.columns.find((c) => c.key === 'product'))
const rateCol = computed(() => props.columns.find((c) => c.type === 'rate' || c.type === 'rateRange'))
const computedCol = computed(() => props.columns.find((c) => c.type === 'computed'))
const verifiedCol = computed(() => props.columns.find((c) => c.type === 'verification'))
const restCols = computed(() =>
  props.columns.filter(
    (c) => ![bankCol.value?.key, productCol.value?.key, rateCol.value?.key, computedCol.value?.key, verifiedCol.value?.key, 'action'].includes(c.key)
  )
)
const restPairs = computed(() => {
  const pairs: ColumnDef[][] = []
  for (let i = 0; i < restCols.value.length; i += 2) pairs.push(restCols.value.slice(i, i + 2))
  return pairs
})

const PAGE_SIZE = 4
const currentPage = ref(1)
const pageCount = computed(() => Math.max(1, Math.ceil(props.rows.length / PAGE_SIZE)))
const pagedRows = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return props.rows.slice(start, start + PAGE_SIZE)
})
watch(
  () => props.rows,
  () => { currentPage.value = 1 }
)
</script>

<template>
  <div>
    <!-- Desktop / tablet table -->
    <div v-if="rows.length" class="hidden overflow-x-auto rounded-card border border-card-border bg-card shadow-sm md:block">
      <table class="w-full min-w-[760px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] font-bold uppercase tracking-wide text-muted">
            <th class="w-10 px-4 py-3"></th>
            <th v-for="col in columns" :key="col.key" class="px-4 py-3" :class="col.align === 'right' ? 'text-right' : 'text-left'">
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in pagedRows"
            :key="row.id"
            class="border-b border-card-border last:border-none"
            :class="isHighlighted(row) ? 'bg-badge-bg' : ''"
          >
            <td class="px-4 py-3.5">
              <input type="checkbox" class="h-4 w-4 accent-primary" :checked="selected.has(row.id)" @change="toggleSelected(row.id)">
            </td>
            <td v-for="col in columns" :key="col.key" class="px-4 py-3.5" :class="col.align === 'right' ? 'text-right' : 'text-left'">
              <span v-if="col.type === 'bank'" class="flex items-center gap-2.5">
                <span
                  class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                  :style="{ backgroundColor: bankColor(row.bank) }"
                >{{ bankInitial(row.bank) }}</span>
                <span class="font-semibold text-navy">{{ row.bank }}</span>
                <span v-if="isHighlighted(row)" class="rounded-pill bg-primary px-2 py-0.5 text-[10px] font-bold text-white">Your Selection</span>
              </span>

              <span v-else-if="col.type === 'rate'">
                <span class="text-[15px] font-bold text-primary">{{ row.rate }}</span>
                <span v-if="row.rateType || paymentLabel" class="block text-[10.5px] text-muted">{{ row.rateType || paymentLabel }}</span>
              </span>

              <span v-else-if="col.type === 'rateRange'">
                <span v-if="row.tiered" class="inline-flex items-center gap-1.5">
                  <span class="text-[15px] font-bold text-primary">{{ row.rate }}</span>
                  <span class="rounded-pill bg-badge-bg px-2 py-0.5 text-[10px] font-bold text-primary">Tiered</span>
                </span>
                <span v-else class="text-[15px] font-bold text-navy">{{ row.rate }}</span>
              </span>

              <span v-else-if="col.type === 'computed'" class="text-[15px] font-bold text-primary">{{ computedValue(row, col) }}</span>

              <span v-else-if="col.type === 'verification'" class="inline-flex items-center gap-1.5 text-muted">
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />{{ row.verified }}
              </span>

              <NuxtLink v-else-if="col.type === 'action'" :to="row.detailHref ?? `/products/sample/${row.id}`" class="font-bold text-primary hover:underline">View Details &rarr;</NuxtLink>

              <span v-else class="text-muted">{{ (row as Record<string, unknown>)[col.key] }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile cards -->
    <div v-if="rows.length" class="flex flex-col gap-3 md:hidden">
      <div
        v-for="row in pagedRows"
        :key="row.id"
        class="rounded-xl border p-3.5 shadow-sm"
        :class="isHighlighted(row) ? 'border-primary bg-badge-bg' : 'border-card-border bg-card'"
      >
        <div class="flex items-center justify-between gap-3">
          <span class="flex min-w-0 items-center gap-2">
            <input type="checkbox" class="h-4 w-4 shrink-0 accent-primary" :checked="selected.has(row.id)" @change="toggleSelected(row.id)">
            <span
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
              :style="{ backgroundColor: bankColor(row.bank) }"
            >{{ bankInitial(row.bank) }}</span>
            <span class="truncate font-semibold text-navy">{{ row.bank }}</span>
          </span>
          <span class="shrink-0 text-right">
            <span class="text-[15px] font-bold" :class="rateCol?.type === 'rateRange' && !row.tiered ? 'text-navy' : 'text-primary'">{{ row.rate }}</span>
            <span v-if="rateCol?.type === 'rateRange' && row.tiered" class="ml-1 rounded-pill bg-badge-bg px-1.5 py-0.5 text-[9px] font-bold text-primary">Tiered</span>
          </span>
        </div>

        <span v-if="isHighlighted(row)" class="mt-1.5 inline-block rounded-pill bg-primary px-2 py-0.5 text-[10px] font-bold text-white">Your Selection</span>
        <p v-if="productCol" class="mt-1.5 text-[13px] text-muted">{{ row.product }}</p>
        <p v-if="row.rateType || paymentLabel" class="text-[11px] text-muted">{{ row.rateType || paymentLabel }}</p>

        <p v-if="computedCol" class="mt-1.5 text-[12.5px] text-muted">
          {{ computedCol.label }}: <span class="font-bold text-primary">{{ computedValue(row, computedCol) }}</span>
        </p>

        <div v-for="(pair, pi) in restPairs" :key="pi" class="mt-1.5 flex items-center justify-between text-[12.5px] text-muted">
          <span v-for="col in pair" :key="col.key">{{ col.label }}: {{ (row as Record<string, unknown>)[col.key] }}</span>
        </div>

        <div class="mt-2.5 flex items-center justify-between border-t border-card-border pt-2.5">
          <span v-if="verifiedCol" class="inline-flex items-center gap-1.5 text-xs text-muted">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />{{ row.verified }}
          </span>
          <span v-else />
          <NuxtLink :to="row.detailHref ?? `/products/sample/${row.id}`" class="text-[13px] font-bold text-primary">View Details &rarr;</NuxtLink>
        </div>
      </div>
    </div>

    <p v-if="rows.length === 0" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
      No products match your filters.
    </p>

    <Pagination v-if="pageCount > 1" v-model="currentPage" :page-count="pageCount" />
  </div>
</template>
