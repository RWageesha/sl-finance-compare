<script setup lang="ts">
import { PRODUCT_TYPES } from '~/config/productTypes'
import { realRowsFor, type Kind, type TaggedRow } from '~/utils/realProductRows'
import { bankKey } from '~/utils/bankColors'

const route = useRoute()
const slug = computed(() => String(route.params.slug))
const config = computed(() => PRODUCT_TYPES[slug.value])

useHead({
  title: computed(() => (config.value ? `${config.value.title} — FindRate LK` : 'Product type not found — FindRate LK'))
})

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()
const allRows = ref<TaggedRow[]>([])
const loading = ref(true)

onMounted(async () => {
  if (!config.value?.live) {
    loading.value = false
    return
  }
  try {
    const [fd, savings, loans] = await Promise.all([
      fetchFixedDeposits().catch(() => []),
      fetchSavings().catch(() => []),
      fetchLoans().catch(() => [])
    ])
    allRows.value = [
      ...fd.map((r) => ({ ...r, kind: 'fd' as Kind })),
      ...savings.map((r) => ({ ...r, kind: 'savings' as Kind })),
      ...loans.map((r) => ({ ...r, kind: 'loans' as Kind }))
    ]
  } finally {
    loading.value = false
  }
})

const liveRows = computed(() => realRowsFor(slug.value, allRows.value))

const filterValues = ref<Record<string, string>>({})
function onFilterValues(v: Record<string, string>) {
  filterValues.value = v
}
watch(slug, () => { filterValues.value = {} })

const filteredRows = computed(() => {
  if (!config.value) return []
  return liveRows.value.filter((row) => {
    for (const f of config.value!.filters) {
      const v = filterValues.value[f.key]
      if (!v || f.key === 'amount') continue
      if (f.type !== 'dropdown') {
        // Free-text (input/multiselect) fields — 'bank' is the only one
        // in practice. Real bank names are full legal names ("Hatton
        // National Bank"), but people search by the common short form
        // ("HNB") used everywhere else on the site — bankKey's short form
        // is folded into the match target so both spellings work.
        if (f.key === 'bank') {
          const target = `${row.bank} ${bankKey(row.bank)}`.toLowerCase()
          if (!target.includes(v.toLowerCase())) return false
          continue
        }
        const target = String(row[f.key] ?? '').toLowerCase()
        if (!target.includes(v.toLowerCase())) return false
        continue
      }
      if (v === (f.options?.[0] ?? '')) continue // first option = "no filter"
      if (f.key === 'tenure') { if (parseTenureLabelToMonths(v) !== row.tenureMonths) return false; continue }
      if (row[f.key] !== undefined && String(row[f.key]) !== v) return false
    }
    return true
  })
})

const sortBy = ref('Highest Rate')
function rowRateValue(row: (typeof filteredRows.value)[number]): number {
  return row.rateValue ?? 0
}
const sortedRows = computed(() => {
  const rows = filteredRows.value.slice()
  if (sortBy.value === 'Highest Rate') rows.sort((a, b) => rowRateValue(b) - rowRateValue(a))
  return rows
})
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1080px]">
        <template v-if="config">
          <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted sm:text-sm" aria-label="Breadcrumb">
            <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
            <span>&rsaquo;</span>
            <NuxtLink to="/products" class="hover:text-primary">Products</NuxtLink>
            <span>&rsaquo;</span>
            <span class="font-semibold text-primary">{{ config.switcherLabel }}</span>
          </nav>

          <ProductPageHeader :eyebrow="config.eyebrow" :title="config.title" :subtitle="config.subtitle" />
          <ProductSwitcher :active="config.slug" />

          <template v-if="!config.live">
            <div class="mt-4 rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
              FindRate doesn't have a live data feed for {{ config.switcherLabel.toLowerCase() }} yet — it's listed here to show
              the full shape of the market, but no bank's rates are tracked for this product type.
            </div>
          </template>
          <template v-else>
            <FilterBar
              :filters="config.filters"
              :compare-label="config.compareLabel"
              :compare-href="config.compareHref"
              @update:values="onFilterValues"
            />

            <div class="mt-4 flex items-center justify-between">
              <p class="text-sm font-semibold text-navy">{{ loading ? 'Loading…' : `${sortedRows.length} Products Found` }}</p>
              <label class="inline-flex items-center gap-1.5 text-xs text-muted">
                Sort by:
                <select v-model="sortBy" class="rounded-lg border border-card-border bg-white px-2 py-1 text-xs text-navy">
                  <option>Highest Rate</option>
                </select>
              </label>
            </div>

            <p v-if="loading" class="mt-4 rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">Loading…</p>
            <ProductResultList v-else :columns="config.columns" :rows="sortedRows" />
          </template>
        </template>

        <div v-else class="rounded-card border border-card-border bg-card p-10 text-center">
          <p class="text-lg font-bold text-navy">Product type not found</p>
          <p class="mt-1 text-sm text-muted">"{{ slug }}" isn't a product type FindRate LK tracks.</p>
          <NuxtLink to="/products" class="mt-4 inline-block text-sm font-bold text-primary hover:underline">
            Browse all product types &rarr;
          </NuxtLink>
        </div>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
