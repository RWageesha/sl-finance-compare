<script setup lang="ts">
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import { bankColor, bankInitial } from '~/utils/bankColors'
import type { ProductRate } from '~/composables/useRatesApi'

const GROUP_META = {
  savings: { directoryLabel: 'Savings Accounts', directoryHref: '/rates?tab=savings' },
  loans: { directoryLabel: 'Loans', directoryHref: '/rates?tab=loans' }
} as const

const route = useRoute()
const group = String(route.params.group) as keyof typeof GROUP_META
const id = Number(route.params.id)

if (!(group in GROUP_META) || !Number.isFinite(id)) {
  throw createError({ statusCode: 404, statusMessage: 'Product not found', fatal: true })
}
const meta = GROUP_META[group]

const { fetchSavings, fetchLoans, fetchRateHistory } = useRatesApi()
const rows = ref<ProductRate[]>([])
const loading = ref(true)
const row = ref<ProductRate | null>(null)
const history = ref<ProductRate[]>([])

onMounted(async () => {
  try {
    rows.value = await (group === 'savings' ? fetchSavings() : fetchLoans()).catch(() => [])
    row.value = rows.value.find((r) => r.id === id) ?? null
    if (row.value) {
      history.value = await fetchRateHistory({
        product_id: row.value.product_id,
        tenure_value: row.value.tenure_value,
        tenure_label: row.value.tenure_label,
        rate_label: row.value.rate_label
      }).catch(() => [])
    }
  } finally {
    loading.value = false
  }
})

function productLabel(r: ProductRate): string {
  return r.tenure_label || r.rate_label || formatCategoryLabel(r.category_code)
}

const bank = computed(() => (row.value ? DIRECTORY_BANKS.find((b) => b.apiName === row.value!.bank_name) : undefined))

useHead({
  title: computed(() => (row.value ? `Rate History — ${row.value.bank_name} ${formatCategoryLabel(row.value.category_code)} — FindRate LK` : 'Rate History — FindRate LK'))
})

const RANGE_OPTIONS = ['3M', '6M', '1Y', 'All'] as const
const activeRange = ref<(typeof RANGE_OPTIONS)[number]>('All')
function cutoffDate(range: string): Date | null {
  if (range === 'All') return null
  const now = new Date()
  now.setMonth(now.getMonth() - (range === '3M' ? 3 : range === '6M' ? 6 : 12))
  return now
}
const rangedHistory = computed(() => {
  const cutoff = cutoffDate(activeRange.value)
  return cutoff ? history.value.filter((r) => new Date(r.scraped_at) >= cutoff) : history.value
})
const chartPoints = computed(() => rangedHistory.value.map((r) => ({ date: r.scraped_at, rate: r.interest_rate })))

const historyDesc = computed(() => [...history.value].reverse())
const changesDesc = computed(() =>
  historyDesc.value.map((h, i) => {
    const prev = historyDesc.value[i + 1]
    if (!prev) return { dir: 'none' as const, amount: 0 }
    const delta = h.interest_rate - prev.interest_rate
    if (delta > 0) return { dir: 'up' as const, amount: delta }
    if (delta < 0) return { dir: 'down' as const, amount: Math.abs(delta) }
    return { dir: 'none' as const, amount: 0 }
  })
)

const reportUrl = computed(() => {
  if (!row.value) return '/report-issue'
  const params = new URLSearchParams({
    bank: bank.value?.slug ?? '',
    category: meta.directoryLabel,
    product: `${row.value.bank_name} ${formatCategoryLabel(row.value.category_code)}`,
    currentValue: `${row.value.interest_rate.toFixed(2)}% p.a.`
  })
  return `/report-issue?${params.toString()}`
})
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[900px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&rsaquo;</span>
          <NuxtLink to="/banks" class="hover:text-primary">Banks</NuxtLink>
          <span>&rsaquo;</span>
          <NuxtLink v-if="bank" :to="`/banks/${bank.slug}`" class="hover:text-primary">{{ bank.displayName }}</NuxtLink>
          <span>&rsaquo;</span>
          <NuxtLink :to="`/products/${group}/${id}`" class="hover:text-primary">{{ row ? formatCategoryLabel(row.category_code) : '' }}</NuxtLink>
          <span>&rsaquo;</span>
          <span class="font-semibold text-primary">History</span>
        </nav>

        <p v-if="loading" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">Loading…</p>
        <p v-else-if="!row" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
          This product isn't in our current dataset.
          <NuxtLink :to="meta.directoryHref" class="mt-2 block font-bold text-primary hover:underline">Browse {{ meta.directoryLabel }} &rarr;</NuxtLink>
        </p>

        <template v-else>
          <div class="mb-5 flex items-center gap-2.5">
            <span
              class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
              :style="{ backgroundColor: bankColor(bank?.displayName ?? row.bank_name) }"
            >{{ bankInitial(bank?.displayName ?? row.bank_name) }}</span>
            <NuxtLink v-if="bank" :to="`/banks/${bank.slug}`" class="text-sm font-semibold text-muted hover:text-primary">{{ bank.displayName }}</NuxtLink>
          </div>
          <h1 class="text-[26px] font-bold text-navy">{{ formatCategoryLabel(row.category_code) }}</h1>
          <p v-if="row.tenure_label || row.rate_label" class="mt-1 text-sm text-muted">{{ productLabel(row) }}</p>
          <span class="mt-2 inline-block rounded-pill bg-badge-bg px-3 py-1 text-xs font-bold text-primary">{{ meta.directoryLabel }}</span>

          <section class="mt-5 rounded-[16px] border border-card-border bg-card p-5 shadow-sm sm:p-7">
            <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
              <h2 class="text-lg font-bold text-navy">Rate History</h2>
              <div class="flex gap-1 rounded-lg bg-page p-1">
                <button
                  v-for="r in RANGE_OPTIONS"
                  :key="r"
                  type="button"
                  class="rounded-md px-3 py-1 text-xs font-bold transition"
                  :class="activeRange === r ? 'bg-primary text-white' : 'text-muted hover:text-navy'"
                  @click="activeRange = r"
                >
                  {{ r }}
                </button>
              </div>
            </div>

            <RateTrendChart v-if="chartPoints.length > 1" :points="chartPoints" large />
            <div v-else class="rounded-lg border border-dashed border-card-border p-6 text-center">
              <p class="text-sm font-bold text-navy">Only {{ chartPoints.length }} verified reading{{ chartPoints.length === 1 ? '' : 's' }} so far.</p>
              <p class="mx-auto mt-1 max-w-md text-xs leading-relaxed text-muted">
                History builds as more scrapes complete — every scrape is stored, never overwritten.
              </p>
            </div>
          </section>

          <section class="mt-4 rounded-[16px] border border-card-border bg-card p-5 shadow-sm sm:p-7">
            <p class="mb-3 text-xs font-bold uppercase tracking-wide text-muted">All Recorded Changes</p>
            <div class="overflow-x-auto">
              <table class="w-full min-w-[520px] text-sm">
                <thead>
                  <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
                    <th class="pb-2 font-semibold">Date</th>
                    <th class="pb-2 font-semibold">Rate</th>
                    <th class="pb-2 font-semibold">Change</th>
                    <th class="pb-2 font-semibold">Source</th>
                    <th class="pb-2 font-semibold">Verification</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(h, i) in historyDesc" :key="h.id" class="border-b border-card-border last:border-none">
                    <td class="py-2.5 text-muted">{{ fmtDate(h.scraped_at) }}</td>
                    <td class="py-2.5 font-bold text-navy">{{ h.interest_rate.toFixed(2) }}%</td>
                    <td class="py-2.5">
                      <span v-if="changesDesc[i].dir === 'up'" class="font-semibold text-emerald-600">&uarr; {{ changesDesc[i].amount.toFixed(2) }}%</span>
                      <span v-else-if="changesDesc[i].dir === 'down'" class="font-semibold text-red-600">&darr; {{ changesDesc[i].amount.toFixed(2) }}%</span>
                      <span v-else class="text-muted">&mdash;</span>
                    </td>
                    <td class="py-2.5">
                      <a v-if="h.source_url" :href="h.source_url" target="_blank" rel="noopener" class="font-semibold text-primary hover:underline">Official page &rarr;</a>
                      <span v-else class="text-muted">&mdash;</span>
                    </td>
                    <td class="py-2.5">
                      <span class="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">&#10003; Verified</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <div class="mt-4 text-center">
            <NuxtLink :to="reportUrl" class="inline-flex items-center gap-1.5 text-[13px] text-muted hover:text-primary">
              <span>&#9873;</span> Report Incorrect Information
            </NuxtLink>
          </div>
        </template>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
