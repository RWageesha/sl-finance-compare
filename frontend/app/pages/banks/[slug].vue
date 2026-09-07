<script setup lang="ts">
import { findDirectoryBank, computeBankStats } from '~/utils/bankDirectory'
import type { TaggedRow } from '~/utils/bankDirectory'
import { productDetailHref } from '~/utils/fdCompare'

const route = useRoute()
const slug = String(route.params.slug)
const bank = findDirectoryBank(slug)

if (!bank) {
  throw createError({ statusCode: 404, statusMessage: 'Bank not found', fatal: true })
}

useHead({ title: `${bank.displayName} — FindRate LK` })

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()

const allRows = ref<TaggedRow[]>([])
const loading = ref(true)

onMounted(async () => {
  if (!bank.tracked) {
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
      ...fd.map((r) => ({ ...r, kind: 'fd' as const })),
      ...savings.map((r) => ({ ...r, kind: 'savings' as const })),
      ...loans.map((r) => ({ ...r, kind: 'loans' as const }))
    ]
  } finally {
    loading.value = false
  }
})

const bankRows = computed(() => allRows.value.filter((r) => r.bank_name === bank.apiName))
const stats = computed(() => computeBankStats(allRows.value, bank.apiName))

const updatedLabel = computed(() => {
  if (!bank.tracked) return 'Not yet integrated'
  if (loading.value) return 'Loading…'
  return stats.value.lastUpdated ? fmtRelativeDate(stats.value.lastUpdated) : 'No data yet'
})

// "Cards" has no real tracked category anywhere in this codebase — kept as
// a visible, honestly-empty tab rather than left out, matching the
// homepage's "Coming soon" convention for the same product type.
const TABS = [
  { key: 'fd', label: 'Fixed Deposits', kind: 'fd' as const },
  { key: 'savings', label: 'Savings', kind: 'savings' as const },
  { key: 'loans', label: 'Loans', kind: 'loans' as const },
  { key: 'cards', label: 'Cards', kind: null }
]
type TabKey = (typeof TABS)[number]['key']

const tabCount = (key: TabKey) => {
  const def = TABS.find((t) => t.key === key)
  if (!def?.kind) return 0
  return bankRows.value.filter((r) => r.kind === def.kind).length
}

// Only shows a tab once its data has loaded and either it has rows, or
// it's the Cards tab — which has no tracked category anywhere in this
// codebase yet, so it stays visible as an honest "coming soon" (see the
// TABS comment above) rather than disappearing for every single bank.
const visibleTabs = computed(() => {
  if (loading.value) return TABS
  return TABS.filter((t) => t.key === 'cards' || tabCount(t.key) > 0)
})
watch(visibleTabs, (tabs) => {
  if (!tabs.some((t) => t.key === activeTab.value) && tabs.length) activeTab.value = tabs[0].key
})

const activeTab = ref<TabKey>('fd')
const tenureFilter = ref('all')
const categoryFilter = ref('all')

const PAGE_SIZE = 5
const currentPage = ref(1)

watch([activeTab, tenureFilter, categoryFilter], () => {
  currentPage.value = 1
})

function tenureLabelFor(row: TaggedRow): string {
  return row.tenure_label || (row.tenure_value ? fmtTenure(row.tenure_value) : '')
}

const tabRows = computed(() => {
  const def = TABS.find((t) => t.key === activeTab.value)
  if (!def?.kind) return []
  return bankRows.value.filter((r) => r.kind === def.kind)
})

const tenureOptions = computed(() => {
  const set = new Set<string>()
  for (const r of tabRows.value) {
    const l = tenureLabelFor(r)
    if (l) set.add(l)
  }
  return Array.from(set)
})

const categoryOptions = computed(() => {
  const set = new Set<string>()
  for (const r of tabRows.value) set.add(r.category_code)
  return Array.from(set)
})

const filteredRows = computed(() =>
  tabRows.value
    .filter((r) => tenureFilter.value === 'all' || tenureLabelFor(r) === tenureFilter.value)
    .filter((r) => categoryFilter.value === 'all' || r.category_code === categoryFilter.value)
    .slice()
    .sort((a, b) => (a.tenure_value ?? 9999) - (b.tenure_value ?? 9999) || b.interest_rate - a.interest_rate)
)

const pageCount = computed(() => Math.max(1, Math.ceil(filteredRows.value.length / PAGE_SIZE)))
const pagedRows = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return filteredRows.value.slice(start, start + PAGE_SIZE)
})

// Compressed page-number list: first, last, current ± 1, with an ellipsis
// marker filling any gap — avoids a wall of buttons once a bank has many
// tenures/categories worth of pages.
const pageNumbers = computed<(number | 'ellipsis')[]>(() => {
  const total = pageCount.value
  const current = currentPage.value
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const pages: (number | 'ellipsis')[] = [1]
  if (current > 3) pages.push('ellipsis')
  for (let p = Math.max(2, current - 1); p <= Math.min(total - 1, current + 1); p++) pages.push(p)
  if (current < total - 2) pages.push('ellipsis')
  pages.push(total)
  return pages
})

function rowTitle(r: TaggedRow): string {
  const tenure = tenureLabelFor(r)
  if (r.kind === 'fd') return tenure ? `${tenure} Fixed Deposit` : formatCategoryLabel(r.category_code)
  return r.product_name || formatCategoryLabel(r.category_code)
}

// Pill tags show only fields the schema actually has (tenure, category,
// rate label) — no minimum-deposit or payout-frequency pill, since that
// data isn't tracked anywhere and this site doesn't invent numbers.
function rowTags(r: TaggedRow): string[] {
  const tags: string[] = []
  const tenure = tenureLabelFor(r)
  if (tenure) tags.push(tenure)
  tags.push(formatCategoryLabel(r.category_code))
  if (r.rate_label) tags.push(r.rate_label)
  return tags
}

// Which new /compare/[slug] page this row belongs to — the 'loans' tab
// mixes housing/personal/gold under one tab, so the compare target has to
// come from the row's own category, not the tab key.
function compareSlugFor(r: TaggedRow): string | null {
  if (r.kind === 'fd') return 'fixed-deposits'
  if (r.kind === 'savings') return 'savings-accounts'
  if (r.category_code === 'HOUSING_LOAN') return 'housing-loans'
  if (r.category_code === 'PERSONAL_LOAN') return 'personal-loans'
  return null
}

// Context-aware: prefills the Compare page from this bank's own top
// result in the active tab, where a real /compare/[slug] page exists for
// it. Gold loans (and anything else with no new Compare page yet) falls
// back to the old /rates query-filter route.
const compareHref = computed(() => {
  const rep = filteredRows.value[0]
  const targetSlug = rep ? compareSlugFor(rep) : null
  if (!rep || !targetSlug) {
    return `/rates?tab=${activeTab.value}&bank=${encodeURIComponent(bank!.apiName ?? '')}`
  }
  const params = new URLSearchParams({
    fromBank: bank!.slug,
    fromProduct: rowTitle(rep),
    rate: rep.interest_rate.toFixed(2)
  })
  if (rep.tenure_value) params.set('tenure', String(rep.tenure_value))
  // Real field, only sent when the scraper actually captured it — this
  // schema has no minimum-deposit data for most rows, so it's routinely
  // absent (see fdCompare.ts's note on what this dataset does and
  // doesn't track). Never invented when missing.
  if (rep.min_amount) params.set('amount', String(rep.min_amount))
  return `/compare/${targetSlug}?${params.toString()}`
})
const compareLabel = computed(() => {
  if (activeTab.value === 'fd') return 'Compare FD'
  if (activeTab.value === 'savings') return 'Compare Savings'
  if (activeTab.value === 'loans') return 'Compare Loans'
  return 'Compare'
})
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1080px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted sm:text-sm" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&rsaquo;</span>
          <NuxtLink to="/banks" class="hover:text-primary">Banks</NuxtLink>
          <span>&rsaquo;</span>
          <span class="font-semibold text-primary">{{ bank.displayName }}</span>
        </nav>

        <!-- Header panel -->
        <section class="flex flex-col gap-5 rounded-card bg-badge-bg p-5 sm:flex-row sm:items-start sm:justify-between sm:p-7">
          <div class="flex items-start gap-4">
            <span class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-card-border bg-white p-2.5 sm:h-24 sm:w-24">
              <BankLogo :bank="bank" />
            </span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h1 class="text-xl font-extrabold text-navy sm:text-2xl">{{ bank.displayName }}</h1>
                <span class="rounded-pill bg-primary/10 px-2.5 py-0.5 text-xs font-bold text-primary">{{ bank.type }}</span>
              </div>
              <div class="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs sm:text-sm">
                <a v-if="bank.sourceUrl" :href="bank.sourceUrl" target="_blank" rel="noopener" class="font-semibold text-primary hover:underline">
                  Official site &nearr;
                </a>
                <span v-if="bank.sourceUrl" class="text-muted">&middot;</span>
                <span class="inline-flex items-center gap-1 text-muted">
                  <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  Last verified {{ updatedLabel }}
                </span>
              </div>
            </div>
          </div>

          <div v-if="bank.tracked" class="flex shrink-0 gap-2 sm:gap-3">
            <div class="min-w-[84px] rounded-lg border border-card-border bg-white px-3 py-2 text-center sm:min-w-[96px] sm:px-4">
              <p class="text-lg font-extrabold text-navy sm:text-xl">{{ stats.productCount }}</p>
              <p class="text-[10px] text-muted sm:text-xs">Products Tracked</p>
            </div>
            <div class="min-w-[84px] rounded-lg border border-card-border bg-white px-3 py-2 text-center sm:min-w-[96px] sm:px-4">
              <p class="flex items-center justify-center gap-1 text-lg font-extrabold text-navy sm:text-xl">
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />{{ stats.sourceCount }}
              </p>
              <p class="text-[10px] text-muted sm:text-xs">Verified Sources</p>
            </div>
            <div class="min-w-[84px] rounded-lg border border-card-border bg-white px-3 py-2 text-center sm:min-w-[96px] sm:px-4">
              <p class="text-lg font-extrabold text-navy sm:text-xl">{{ stats.categoriesTracked }}</p>
              <p class="text-[10px] text-muted sm:text-xs">Rate Categories</p>
            </div>
          </div>
        </section>

        <template v-if="bank.tracked">
          <!-- Tabs -->
          <div class="mt-5 flex flex-wrap gap-2">
            <button
              v-for="tab in visibleTabs"
              :key="tab.key"
              type="button"
              class="rounded-pill px-4 py-2 text-sm font-semibold transition"
              :class="activeTab === tab.key ? 'bg-primary text-white' : 'border border-card-border bg-white text-navy hover:border-primary'"
              @click="activeTab = tab.key"
            >
              {{ tab.label }} <span class="opacity-70">({{ tabCount(tab.key) }})</span>
            </button>
          </div>

          <!-- Filter bar -->
          <div v-if="activeTab !== 'cards'" class="mt-4 flex flex-wrap items-center gap-3 rounded-card border border-card-border bg-card p-4 shadow-sm">
            <select v-model="tenureFilter" class="rounded-lg border border-card-border px-3 py-2 text-sm text-navy">
              <option value="all">Tenure: All</option>
              <option v-for="t in tenureOptions" :key="t" :value="t">{{ t }}</option>
            </select>
            <select v-model="categoryFilter" class="rounded-lg border border-card-border px-3 py-2 text-sm text-navy">
              <option value="all">Category: All</option>
              <option v-for="c in categoryOptions" :key="c" :value="c">{{ formatCategoryLabel(c) }}</option>
            </select>
            <NuxtLink :to="compareHref" class="ml-auto rounded-lg bg-primary px-5 py-2 text-sm font-bold text-white transition hover:bg-primary/90">
              {{ compareLabel }}
            </NuxtLink>
          </div>

          <!-- Result list -->
          <div v-if="activeTab === 'cards'" class="mt-4 rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
            Credit card tariffs aren't tracked yet — this tab is shown to reflect the full shape of the market.
          </div>
          <p v-else-if="loading" class="mt-4 rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">Loading…</p>
          <p v-else-if="filteredRows.length === 0" class="mt-4 rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
            No {{ activeTab }} rates match these filters yet.
          </p>
          <div v-else class="mt-4 divide-y divide-card-border rounded-card border border-card-border bg-card">
            <div v-for="r in pagedRows" :key="r.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5">
              <div class="min-w-0">
                <h3 class="font-bold text-navy">{{ rowTitle(r) }}</h3>
                <div class="mt-1.5 flex flex-wrap gap-1.5">
                  <span v-for="tag in rowTags(r)" :key="tag" class="rounded-pill bg-page px-2.5 py-0.5 text-[11px] font-medium text-muted">
                    {{ tag }}
                  </span>
                </div>
              </div>
              <div class="flex shrink-0 items-center justify-between gap-4 sm:justify-end">
                <div class="text-right">
                  <p class="text-xl font-extrabold text-primary">{{ r.interest_rate.toFixed(2) }}%</p>
                  <p class="text-[11px] text-muted">Interest Rate (p.a.)</p>
                  <p class="mt-0.5 inline-flex items-center gap-1 text-[11px] text-muted">
                    <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />Verified {{ fmtRelativeDate(r.scraped_at) }}
                  </p>
                </div>
                <NuxtLink
                  v-if="productDetailHref(r)"
                  :to="productDetailHref(r)!"
                  class="shrink-0 rounded-pill bg-badge-bg px-4 py-2 text-xs font-bold text-primary transition hover:bg-primary hover:text-white"
                >
                  Details &rarr;
                </NuxtLink>
              </div>
            </div>
          </div>

          <div v-if="filteredRows.length > 0" class="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm text-muted">
            <span>
              Showing {{ (currentPage - 1) * PAGE_SIZE + 1 }}-{{ Math.min(currentPage * PAGE_SIZE, filteredRows.length) }}
              of {{ filteredRows.length }} results
            </span>
            <div v-if="pageCount > 1" class="flex items-center gap-1">
              <button
                type="button"
                aria-label="Previous page"
                class="flex h-8 w-8 items-center justify-center rounded-lg border border-card-border text-navy transition hover:border-primary disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-card-border"
                :disabled="currentPage === 1"
                @click="currentPage--"
              >
                &larr;
              </button>
              <template v-for="(p, i) in pageNumbers" :key="i">
                <span v-if="p === 'ellipsis'" class="px-1">&hellip;</span>
                <button
                  v-else
                  type="button"
                  class="flex h-8 w-8 items-center justify-center rounded-lg text-sm font-semibold transition"
                  :class="p === currentPage ? 'bg-primary text-white' : 'border border-card-border text-navy hover:border-primary'"
                  @click="currentPage = p"
                >
                  {{ p }}
                </button>
              </template>
              <button
                type="button"
                aria-label="Next page"
                class="flex h-8 w-8 items-center justify-center rounded-lg border border-card-border text-navy transition hover:border-primary disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-card-border"
                :disabled="currentPage === pageCount"
                @click="currentPage++"
              >
                &rarr;
              </button>
            </div>
          </div>
        </template>

        <div v-else class="mt-5 rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
          FindRate doesn't have a live data feed for {{ bank.displayName }} yet. It's listed here to show the full shape of the
          market — {{ bank.type.toLowerCase() }} coverage may be added later.
        </div>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
