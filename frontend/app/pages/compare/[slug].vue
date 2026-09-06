<script setup lang="ts">
import { PRODUCT_TYPES } from '~/config/productTypes'

const route = useRoute()
const slug = computed(() => String(route.params.slug))
const config = computed(() => PRODUCT_TYPES[slug.value])
const compare = computed(() => config.value?.compare)

useHead({
  title: computed(() => (compare.value ? `${compare.value.heading} — FindRate LK` : 'Compare page not found — FindRate LK'))
})

// --- Prefill from the query string (Bank Profile / Product Detail Compare
// buttons) --- `route.query` is replaced (not mutated) on every
// navigation, so this has to stay a computed — a plain `const query =
// route.query` snapshot goes stale the moment a client-side nav (e.g.
// "Clear filters") changes only the query on this same matched route,
// since Vue Router reuses the component instance rather than remounting it.
const query = computed(() => route.query)
const hasPrefill = computed(() => !!(query.value.fromBank && query.value.fromProduct))
const highlight = computed(() => (hasPrefill.value ? { bank: String(query.value.fromBank), product: String(query.value.fromProduct) } : null))

const prefillMessage = computed(() => {
  if (!compare.value || !hasPrefill.value) return ''
  return compare.value.prefillMessageTemplate
    .replace('{bank}', String(query.value.fromBank))
    .replace('{product}', String(query.value.fromProduct))
    .replace('{rate}', String(query.value.rate ?? ''))
})

// --- Filter state: seeded from query params when prefilled, else the
// config's own first-option defaults ---
function defaultFilterValues(): Record<string, string> {
  const values: Record<string, string> = {}
  if (!compare.value) return values
  const q = query.value
  for (const f of compare.value.filters) {
    if (f.key === 'amount' && q.amount) {
      values[f.key] = `Rs. ${Number(q.amount).toLocaleString('en-LK')}`
    } else if (f.key === 'tenure' && q.tenure) {
      // tenure arrives as a month count; match it back to the closest option label
      const months = Number(q.tenure)
      const match = f.options?.find((o) => parseTenureLabelToMonths(o) === months)
      values[f.key] = match ?? f.options?.[0] ?? ''
    } else if (f.key === 'payment' && q.payment && f.options?.includes(String(q.payment))) {
      values[f.key] = String(q.payment)
    } else {
      values[f.key] = f.type === 'input' ? (f.placeholder ?? '') : (f.options?.[0] ?? '')
    }
  }
  return values
}

const filterValues = ref<Record<string, string>>({})
watch([compare, query], () => { filterValues.value = defaultFilterValues() }, { immediate: true })

function applyFilters(values: Record<string, string>) {
  filterValues.value = values
}

const amount = computed(() => {
  const raw = filterValues.value.amount ?? ''
  const n = Number(raw.replace(/[^0-9]/g, ''))
  return Number.isFinite(n) && n > 0 ? n : 1000000
})
const tenureMonths = computed(() => parseTenureLabelToMonths(filterValues.value.tenure ?? '') || 12)
const paymentLabel = computed(() => (slug.value === 'fixed-deposits' ? filterValues.value.payment : undefined))

const buttonLabel = computed(() => (compare.value ? (hasPrefill.value ? compare.value.updateLabel : compare.value.defaultLabel) : ''))

// --- Filtering: only FD's payment and Savings' accountType/minBalance
// dropdowns actually narrow the row list — amount/tenure are calculator
// inputs (used above for the computed maturity/EMI column), not filters.
function minBalanceInRange(raw: string, rangeLabel: string): boolean {
  const amount = Number(raw.replace(/[^0-9]/g, ''))
  if (rangeLabel === 'Under Rs. 5,000') return amount < 5000
  if (rangeLabel === 'Rs. 5,000 – 50,000') return amount >= 5000 && amount <= 50000
  if (rangeLabel === 'Above Rs. 50,000') return amount > 50000
  return true
}
const filteredRows = computed(() => {
  if (!compare.value) return []
  return compare.value.rows.filter((row) => {
    // FD's "Interest Payment" dropdown has no neutral "Any" option — it's
    // a shared scenario label shown under every row (via paymentLabel
    // below), not a row filter, so every FD row stays visible regardless
    // of which payment frequency is selected.
    if (slug.value === 'savings-accounts') {
      const accountType = filterValues.value.accountType
      if (accountType && accountType !== 'Any' && row.category !== accountType) return false
      const minBalance = filterValues.value.minBalance
      if (minBalance && minBalance !== 'Any' && typeof row.minBalance === 'string' && !minBalanceInRange(row.minBalance, minBalance)) return false
    }
    return true
  })
})

const sortBy = ref('Highest Rate')
function rowRateValue(row: (typeof filteredRows.value)[number]): number {
  if (row.rateValue !== undefined) return row.rateValue
  const n = Number.parseFloat(String(row.rate ?? ''))
  return Number.isFinite(n) ? n : 0
}
const sortedRows = computed(() => {
  const rows = filteredRows.value.slice()
  if (sortBy.value === 'Highest Rate') rows.sort((a, b) => rowRateValue(b) - rowRateValue(a))
  else if (sortBy.value === 'Lowest Minimum') {
    const minOf = (r: (typeof rows)[number]) => Number(String(r.minDeposit ?? r.minBalance ?? '').replace(/[^0-9]/g, '')) || Infinity
    rows.sort((a, b) => minOf(a) - minOf(b))
  }
  return rows
})
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1080px]">
        <template v-if="compare && config">
          <CompareHeader :switcher-label="config.switcherLabel" :heading="compare.heading" :subtitle="compare.subtitle" />
          <PrefillBanner v-if="hasPrefill" :message="prefillMessage" :base-href="`/compare/${slug}`" />

          <div class="mt-4">
            <CompareFilterBar
              :filters="compare.filters"
              :initial-values="filterValues"
              :button-label="buttonLabel"
              :sort-by="sortBy"
              @apply="applyFilters"
              @update:sort-by="sortBy = $event"
            />
          </div>

          <p class="mt-4 text-sm font-semibold text-navy">{{ sortedRows.length }} Products Found</p>

          <div class="mt-3">
            <CompareResultList
              :columns="compare.columns"
              :rows="sortedRows"
              :amount="amount"
              :tenure-months="tenureMonths"
              :payment-label="paymentLabel"
              :highlight="highlight"
            />
          </div>
        </template>

        <div v-else class="rounded-card border border-card-border bg-card p-10 text-center">
          <p class="text-lg font-bold text-navy">Compare page not found</p>
          <p class="mt-1 text-sm text-muted">"{{ slug }}" doesn't have a comparison view yet.</p>
          <NuxtLink to="/products" class="mt-4 inline-block text-sm font-bold text-primary hover:underline">
            Browse all product types &rarr;
          </NuxtLink>
        </div>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
