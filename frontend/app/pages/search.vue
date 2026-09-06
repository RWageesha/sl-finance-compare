<script setup lang="ts">
import { isFdRow, productDetailHref } from '~/utils/fdCompare'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import { bankColor, bankInitial } from '~/utils/bankColors'
import type { ProductRate } from '~/composables/useRatesApi'

type Kind = 'fd' | 'savings' | 'loans'
type Row = ProductRate & { kind: Kind }

const route = useRoute()
const searchInput = ref(String(route.query.q ?? ''))

useHead({ title: computed(() => (route.query.q ? `Search: ${route.query.q} — FindRate LK` : 'Search — FindRate LK')) })

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()
const allRows = ref<Row[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const [fd, savings, loans] = await Promise.all([
      fetchFixedDeposits().catch(() => []),
      fetchSavings().catch(() => []),
      fetchLoans().catch(() => [])
    ])
    allRows.value = [
      ...fd.map((r) => ({ ...r, kind: 'fd' as const })),
      ...savings.map((r) => ({ ...r, kind: 'savings' as const })),
      ...loans.map((r) => ({ ...r, kind: 'loans' as const }))
    ]
  } finally {
    loading.value = false
  }
})

function rowLabel(r: Row): string {
  if (r.kind === 'fd' && isFdRow(r)) return `${r.tenure_label || fmtTenure(r.tenure_value ?? 0)} ${formatCategoryLabel(r.category_code)}`
  return r.product_name || r.tenure_label || r.rate_label || formatCategoryLabel(r.category_code)
}
const CATEGORY_LABEL: Record<Kind, string> = { fd: 'Fixed Deposits', savings: 'Savings', loans: 'Loans' }

const query = ref(String(route.query.q ?? ''))
function runSearch() {
  query.value = searchInput.value
  navigateTo({ path: '/search', query: { q: searchInput.value } })
}

const matched = computed(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) return []
  return allRows.value.filter((r) => r.bank_name.toLowerCase().includes(q) || rowLabel(r).toLowerCase().includes(q) || formatCategoryLabel(r.category_code).toLowerCase().includes(q))
})

// --- Sidebar filters, counted against the query-matched set ---
const bankFilter = ref<Set<string>>(new Set())
const categoryFilter = ref<Set<Kind>>(new Set())
const minRate = ref('')
const maxRate = ref('')

const bankCounts = computed(() => {
  const counts = new Map<string, number>()
  for (const r of matched.value) counts.set(r.bank_name, (counts.get(r.bank_name) ?? 0) + 1)
  return counts
})
const categoryCounts = computed(() => {
  const counts = new Map<Kind, number>()
  for (const r of matched.value) counts.set(r.kind, (counts.get(r.kind) ?? 0) + 1)
  return counts
})

const filtered = computed(() => {
  const lo = Number(minRate.value) || 0
  const hi = Number(maxRate.value) || Infinity
  return matched.value.filter((r) => {
    if (bankFilter.value.size && !bankFilter.value.has(r.bank_name)) return false
    if (categoryFilter.value.size && !categoryFilter.value.has(r.kind)) return false
    if (r.interest_rate < lo || r.interest_rate > hi) return false
    return true
  })
})

function toggleBank(name: string) {
  bankFilter.value.has(name) ? bankFilter.value.delete(name) : bankFilter.value.add(name)
  bankFilter.value = new Set(bankFilter.value)
}
function toggleCategory(k: Kind) {
  categoryFilter.value.has(k) ? categoryFilter.value.delete(k) : categoryFilter.value.add(k)
  categoryFilter.value = new Set(categoryFilter.value)
}
function clearFilters() {
  bankFilter.value = new Set()
  categoryFilter.value = new Set()
  minRate.value = ''
  maxRate.value = ''
}

const sortBy = ref('Highest Rate')
const sorted = computed(() => {
  const rows = filtered.value.slice()
  if (sortBy.value === 'Highest Rate') rows.sort((a, b) => b.interest_rate - a.interest_rate)
  else if (sortBy.value === 'Lowest Rate') rows.sort((a, b) => a.interest_rate - b.interest_rate)
  return rows
})

const PAGE_SIZE = 8
const currentPage = ref(1)
watch([query, bankFilter, categoryFilter, sortBy], () => { currentPage.value = 1 }, { deep: true })
const pageCount = computed(() => Math.max(1, Math.ceil(sorted.value.length / PAGE_SIZE)))
const paged = computed(() => sorted.value.slice((currentPage.value - 1) * PAGE_SIZE, currentPage.value * PAGE_SIZE))

function bankFor(r: Row) {
  return DIRECTORY_BANKS.find((b) => b.apiName === r.bank_name)
}

const SUGGESTIONS = ['Fixed Deposits', 'Housing Loan', 'Savings Account']
function trySuggestion(term: string) {
  searchInput.value = term
  runSearch()
}
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1080px]">
        <form class="flex gap-2" @submit.prevent="runSearch">
          <input
            v-model="searchInput"
            type="text"
            placeholder="Search banks, products, or rates..."
            class="w-full rounded-lg border border-card-border bg-white px-4 py-2.5 text-sm text-navy"
          >
          <button type="submit" class="shrink-0 rounded-lg bg-primary px-5 py-2.5 text-sm font-bold text-white transition hover:bg-primary/90">Search</button>
        </form>

        <div v-if="query" class="mt-4">
          <p class="text-lg text-muted">Showing results for '<strong class="text-navy">{{ query }}</strong>'</p>
          <p class="text-xs text-muted">{{ loading ? 'Searching…' : `${sorted.length} results found` }}</p>
        </div>

        <template v-if="!loading && query && sorted.length === 0">
          <div class="mt-8 flex flex-col items-center rounded-card border border-card-border bg-card p-12 text-center">
            <span class="text-3xl text-muted">&#128269;</span>
            <p class="mt-3 text-lg font-bold text-navy">No results found for '{{ query }}'</p>
            <p class="mt-1 text-xs text-muted">Try checking your spelling or searching for a broader term.</p>
            <div class="mt-4 flex flex-wrap justify-center gap-2">
              <button
                v-for="s in SUGGESTIONS"
                :key="s"
                type="button"
                class="rounded-pill border border-card-border px-3.5 py-1.5 text-xs font-semibold text-navy transition hover:border-primary hover:text-primary"
                @click="trySuggestion(s)"
              >
                {{ s }}
              </button>
            </div>
          </div>
        </template>

        <div v-else-if="query" class="mt-5 grid grid-cols-1 gap-5 lg:grid-cols-[240px_1fr]">
          <aside class="flex flex-col gap-5 rounded-card border border-card-border bg-card p-5">
            <div>
              <p class="mb-2 text-xs font-bold uppercase tracking-wide text-muted">Filter by Bank</p>
              <label v-for="[name, count] in bankCounts" :key="name" class="mb-1.5 flex items-center justify-between gap-2 text-[13px] text-navy">
                <span class="flex items-center gap-2"><input type="checkbox" :checked="bankFilter.has(name)" class="h-3.5 w-3.5 accent-primary" @change="toggleBank(name)">{{ name }}</span>
                <span class="text-xs text-muted">({{ count }})</span>
              </label>
            </div>
            <div class="border-t border-card-border pt-4">
              <p class="mb-2 text-xs font-bold uppercase tracking-wide text-muted">Filter by Category</p>
              <label v-for="[k, count] in categoryCounts" :key="k" class="mb-1.5 flex items-center justify-between gap-2 text-[13px] text-navy">
                <span class="flex items-center gap-2"><input type="checkbox" :checked="categoryFilter.has(k)" class="h-3.5 w-3.5 accent-primary" @change="toggleCategory(k)">{{ CATEGORY_LABEL[k] }}</span>
                <span class="text-xs text-muted">({{ count }})</span>
              </label>
            </div>
            <div class="border-t border-card-border pt-4">
              <p class="mb-2 text-xs font-bold uppercase tracking-wide text-muted">Filter by Rate Range</p>
              <div class="flex items-center gap-2">
                <input v-model="minRate" type="number" placeholder="Min" class="w-full rounded-lg border border-card-border px-2 py-1.5 text-xs">
                <span class="text-xs text-muted">to</span>
                <input v-model="maxRate" type="number" placeholder="Max" class="w-full rounded-lg border border-card-border px-2 py-1.5 text-xs">
              </div>
            </div>
            <div class="border-t border-card-border pt-3">
              <button type="button" class="text-xs font-semibold text-muted hover:text-primary" @click="clearFilters">Clear All Filters</button>
            </div>
          </aside>

          <div>
            <div class="mb-3 flex justify-end">
              <label class="inline-flex items-center gap-1.5 text-xs text-muted">
                Sort by:
                <select v-model="sortBy" class="rounded-lg border border-card-border bg-white px-2 py-1 text-xs text-navy">
                  <option>Highest Rate</option>
                  <option>Lowest Rate</option>
                </select>
              </label>
            </div>

            <div class="divide-y divide-card-border rounded-card border border-card-border bg-card">
              <div v-for="r in paged" :key="`${r.kind}-${r.id}`" class="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex min-w-0 items-center gap-3">
                  <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white" :style="{ backgroundColor: bankColor(r.bank_name) }">{{ bankInitial(r.bank_name) }}</span>
                  <div class="min-w-0">
                    <p class="truncate text-sm font-semibold text-navy">{{ r.bank_name }}</p>
                    <p class="truncate text-xs text-muted">{{ rowLabel(r) }}</p>
                  </div>
                  <span class="hidden shrink-0 rounded-pill bg-badge-bg px-2.5 py-0.5 text-[11px] font-bold text-primary sm:inline-block">{{ CATEGORY_LABEL[r.kind] }}</span>
                </div>
                <div class="flex shrink-0 items-center gap-4">
                  <strong class="text-base text-primary">{{ r.interest_rate.toFixed(2) }}%</strong>
                  <NuxtLink v-if="productDetailHref(r)" :to="productDetailHref(r)!" class="text-xs font-bold text-primary hover:underline">View Details &rarr;</NuxtLink>
                </div>
              </div>
              <p v-if="!loading && paged.length === 0" class="p-6 text-center text-sm text-muted">No results match the current filters.</p>
            </div>

            <div v-if="pageCount > 1" class="mt-4 flex items-center justify-center gap-3">
              <button type="button" class="text-xs font-semibold text-muted disabled:opacity-40" :disabled="currentPage === 1" @click="currentPage--">Previous</button>
              <span class="text-xs font-semibold text-navy">Page {{ currentPage }} of {{ pageCount }}</span>
              <button type="button" class="text-xs font-semibold text-primary disabled:opacity-40" :disabled="currentPage === pageCount" @click="currentPage++">Next</button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
