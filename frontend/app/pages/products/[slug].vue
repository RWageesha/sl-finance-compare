<script setup lang="ts">
import { PRODUCT_TYPES } from '~/config/productTypes'

const route = useRoute()
const slug = computed(() => String(route.params.slug))
const config = computed(() => PRODUCT_TYPES[slug.value])

useHead({
  title: computed(() => (config.value ? `${config.value.title} — FindRate LK` : 'Product type not found — FindRate LK'))
})

const filterValues = ref<Record<string, string>>({})
function onFilterValues(v: Record<string, string>) {
  filterValues.value = v
}
watch(slug, () => { filterValues.value = {} })

function parseRupees(s: string): number {
  return Number(s.replace(/[^0-9]/g, '')) || 0
}
function inRange(raw: string, rangeLabel: string): boolean {
  const amt = parseRupees(raw)
  if (rangeLabel.startsWith('Under')) return amt < parseRupees(rangeLabel)
  if (rangeLabel.startsWith('Above')) return amt > parseRupees(rangeLabel)
  const [lo, hi] = rangeLabel.replace(/[^\d–-]/g, '').split(/[–-]/).map((n) => Number(n))
  return amt >= lo && amt <= hi
}

const filteredRows = computed(() => {
  if (!config.value) return []
  return config.value.rows.filter((row) => {
    for (const f of config.value!.filters) {
      const v = filterValues.value[f.key]
      if (!v || f.key === 'amount') continue
      if (f.type !== 'dropdown') {
        // Free-text (input/multiselect) fields — 'bank' is the only one
        // in practice, matched as a case-insensitive substring.
        const target = String(row[f.key] ?? (f.key === 'bank' ? row.bank : '')).toLowerCase()
        if (!target.includes(v.toLowerCase())) return false
        continue
      }
      if (v === (f.options?.[0] ?? '')) continue // first option = "no filter"
      if (f.key === 'accountType') { if (row.category !== v) return false; continue }
      if (f.key === 'minBalance' && typeof row.minBalance === 'string') { if (!inRange(row.minBalance, v)) return false; continue }
      if (f.key === 'annualFeeRange' && typeof row.annualFee === 'string') { if (!inRange(row.annualFee, v)) return false; continue }
      if (row[f.key] !== undefined && String(row[f.key]) !== v) return false
    }
    return true
  })
})

const sortBy = ref('Highest Rate')
function rowRateValue(row: (typeof filteredRows.value)[number]): number {
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
          <FilterBar
            :filters="config.filters"
            :compare-label="config.compareLabel"
            :compare-href="config.compareHref"
            @update:values="onFilterValues"
          />

          <div class="mt-4 flex items-center justify-between">
            <p class="text-sm font-semibold text-navy">{{ sortedRows.length }} Products Found</p>
            <label class="inline-flex items-center gap-1.5 text-xs text-muted">
              Sort by:
              <select v-model="sortBy" class="rounded-lg border border-card-border bg-white px-2 py-1 text-xs text-navy">
                <option>Highest Rate</option>
                <option>Lowest Minimum</option>
              </select>
            </label>
          </div>

          <ProductResultList :columns="config.columns" :rows="sortedRows" />
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
