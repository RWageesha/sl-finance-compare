<script setup lang="ts">
// Consolidated Data & Transparency page — replaces the earlier separate
// "Data Dashboard" and "Transparency Hub" drafts. Every number here is
// either read straight from the live API or explicitly noted as
// unavailable — see the Core Sourcing Principle callout below. No
// "Data Health %" or similar invented composite score: this codebase
// has an explicit rule against exactly that kind of number (see
// utils/bankDirectory.ts's own comment on DIRECTORY_BANKS), and there's
// no real scrape-success-log endpoint to build a literal activity feed
// from, so "Scrape Activity" below is built from real per-bank/category
// scrape timestamps and row counts instead of a fabricated log.
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import { isFdRow } from '~/utils/fdCompare'
import { bankColor } from '~/utils/bankColors'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Data & Transparency — FindRate LK' })

const { fetchFixedDeposits, fetchSavings, fetchLoans, fetchRateHistory } = useRatesApi()

const fd = ref<ProductRate[]>([])
const savings = ref<ProductRate[]>([])
const loans = ref<ProductRate[]>([])
const loading = ref(true)
const chartSeries = ref<{ label: string; color: string; points: { date: string; rate: number }[] }[]>([])

const trackedBanks = computed(() => DIRECTORY_BANKS.filter((b) => b.tracked))
const allRows = computed<ProductRate[]>(() => [...fd.value, ...savings.value, ...loans.value])

onMounted(async () => {
  try {
    ;[fd.value, savings.value, loans.value] = await Promise.all([
      fetchFixedDeposits().catch(() => []),
      fetchSavings().catch(() => []),
      fetchLoans().catch(() => [])
    ])

    // One comparable line per tracked bank: its 12-month standard FD,
    // where one exists in the current dataset.
    const series: typeof chartSeries.value = []
    for (const bank of trackedBanks.value) {
      const row = fd.value.filter(isFdRow).find((r) => r.bank_name === bank.apiName && r.tenure_value === 12)
      if (!row) continue
      const history = await fetchRateHistory({
        product_id: row.product_id,
        tenure_value: row.tenure_value,
        tenure_label: row.tenure_label,
        rate_label: row.rate_label
      }).catch(() => [])
      if (history.length === 0) continue
      series.push({
        label: bank.displayName.replace(/\s*\(.*\)/, ''),
        color: bankColor(bank.apiName ?? bank.displayName),
        points: history.map((r) => ({ date: r.scraped_at, rate: r.interest_rate }))
      })
    }
    chartSeries.value = series
  } finally {
    loading.value = false
  }
})

const banksTrackedLabel = computed(() => `${trackedBanks.value.length} of ${DIRECTORY_BANKS.length}`)
const productsTrackedCount = computed(() => allRows.value.length)
const lastScrapeAt = computed<string | null>(() => {
  if (allRows.value.length === 0) return null
  return allRows.value.reduce((max, r) => (!max || r.scraped_at > max ? r.scraped_at : max), null as string | null)
})
const verifiedSourcesCount = computed(() => new Set(allRows.value.map((r) => r.source_url).filter(Boolean)).size)

// One row per tracked bank + category that actually has data, sorted by
// most recently scraped — a real, derivable "what did we last pull"
// view, not a literal scrape-success/failure log we don't have.
const scrapeActivity = computed(() => {
  const cats: { key: string; label: string; rows: ProductRate[] }[] = [
    { key: 'fd', label: 'Fixed Deposits', rows: fd.value },
    { key: 'savings', label: 'Savings', rows: savings.value },
    { key: 'loans', label: 'Loans', rows: loans.value }
  ]
  const entries: { bank: string; category: string; count: number; lastScrape: string }[] = []
  for (const bank of trackedBanks.value) {
    for (const cat of cats) {
      const rows = cat.rows.filter((r) => r.bank_name === bank.apiName)
      if (rows.length === 0) continue
      const last = rows.reduce((max, r) => (r.scraped_at > max ? r.scraped_at : max), rows[0].scraped_at)
      entries.push({ bank: bank.displayName.replace(/\s*\(.*\)/, ''), category: cat.label, count: rows.length, lastScrape: last })
    }
  }
  return entries.sort((a, b) => (a.lastScrape > b.lastScrape ? -1 : 1)).slice(0, 6)
})

const dataSourceRows = computed(() =>
  trackedBanks.value.map((bank) => {
    const rows = allRows.value.filter((r) => r.bank_name === bank.apiName)
    const last = rows.length ? rows.reduce((max, r) => (r.scraped_at > max ? r.scraped_at : max), rows[0].scraped_at) : null
    return { bank, lastChecked: last }
  })
)
const untrackedCount = computed(() => DIRECTORY_BANKS.length - trackedBanks.value.length)
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1140px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&rsaquo;</span>
          <span class="font-semibold text-primary">Data</span>
        </nav>

        <!-- Hero -->
        <section class="rounded-[16px] bg-badge-bg p-6 sm:p-8">
          <span class="inline-block rounded-pill bg-primary/10 px-3 py-1 text-[11px] font-bold uppercase tracking-wider text-primary">Data &amp; Transparency</span>
          <h1 class="mt-3 text-[28px] font-bold text-navy sm:text-[32px]">Data &amp; Transparency</h1>
          <p class="mt-1.5 max-w-2xl text-sm text-muted">
            See exactly how FindRate LK collects, verifies, and updates every rate on this site. No invented numbers, ever.
          </p>
        </section>

        <!-- Stat row -->
        <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="rounded-card border border-card-border bg-card p-4">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Banks Tracked</p>
            <p class="mt-1 text-2xl font-bold text-navy">{{ banksTrackedLabel }}</p>
            <p class="mt-0.5 text-xs text-muted">Sri Lankan licensed banks</p>
          </div>
          <div class="rounded-card border border-card-border bg-card p-4">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Products Tracked</p>
            <p class="mt-1 text-2xl font-bold text-navy">{{ loading ? '—' : productsTrackedCount }}</p>
            <p class="mt-0.5 text-xs text-muted">Across all categories</p>
          </div>
          <div class="rounded-card border border-card-border bg-card p-4">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Last Scrape</p>
            <p class="mt-1 flex items-center gap-1.5 text-2xl font-bold text-navy">
              <span class="h-2 w-2 rounded-full bg-emerald-500" />
              <span class="text-base sm:text-xl">{{ lastScrapeAt ? fmtRelativeDate(lastScrapeAt) : '—' }}</span>
            </p>
            <p class="mt-0.5 text-xs text-muted">Automated, every 6 hours</p>
          </div>
          <div class="rounded-card border border-card-border bg-card p-4">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Verified Sources</p>
            <p class="mt-1 flex items-center gap-1.5 text-2xl font-bold text-navy">
              <span class="h-2 w-2 rounded-full bg-emerald-500" />{{ loading ? '—' : verifiedSourcesCount }}
            </p>
            <p class="mt-0.5 text-xs text-muted">Distinct bank source pages</p>
          </div>
        </div>

        <!-- Chart -->
        <section class="mt-4 rounded-[16px] border border-card-border bg-card p-5 shadow-sm sm:p-7">
          <h2 class="text-base font-bold text-navy">Fixed Deposit Rate Trends by Bank</h2>
          <p class="-mt-0.5 text-xs text-muted">12-month tenure, historical rates</p>

          <div class="mt-4">
            <MultiLineChart v-if="chartSeries.length" :series="chartSeries" />
            <p v-else-if="!loading" class="rounded-lg border border-dashed border-card-border p-6 text-center text-sm text-muted">
              Not enough rate history yet to plot a trend for any tracked bank's 12-month FD.
            </p>
          </div>

          <div v-if="chartSeries.length" class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5">
            <span v-for="s in chartSeries" :key="s.label" class="inline-flex items-center gap-1.5 text-xs text-muted">
              <span class="h-2 w-2 rounded-full" :style="{ backgroundColor: s.color }" />{{ s.label }}
              <span v-if="s.points.length < 3" class="text-[11px] text-muted">(Scrape recently started)</span>
            </span>
          </div>
          <p class="mt-2 text-xs text-muted">
            Showing verified data for {{ trackedBanks.length }} of {{ DIRECTORY_BANKS.length }} tracked banks. Additional banks are added as scraping coverage expands.
          </p>
        </section>

        <!-- Sources + Activity -->
        <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
          <section class="rounded-[16px] border border-card-border bg-card p-5 shadow-sm">
            <h2 class="mb-3 text-base font-bold text-navy">Data Sources</h2>
            <div class="flex flex-col divide-y divide-card-border">
              <div v-for="row in dataSourceRows" :key="row.bank.slug" class="flex items-center justify-between gap-3 py-2.5">
                <div class="flex min-w-0 items-center gap-2.5">
                  <span
                    class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[11px] font-bold text-white"
                    :style="{ backgroundColor: bankColor(row.bank.apiName ?? row.bank.displayName) }"
                  >{{ row.bank.displayName[0] }}</span>
                  <div class="min-w-0">
                    <p class="truncate text-sm font-semibold text-navy">{{ row.bank.displayName }}</p>
                    <a v-if="row.bank.sourceUrl" :href="row.bank.sourceUrl" target="_blank" rel="noopener" class="truncate text-xs text-primary hover:underline">{{ row.bank.sourceUrl }}</a>
                  </div>
                </div>
                <span class="shrink-0 text-xs text-muted">{{ row.lastChecked ? `Checked ${fmtRelativeDate(row.lastChecked)}` : 'No data yet' }}</span>
              </div>
            </div>
            <p v-if="untrackedCount > 0" class="mt-2.5 border-t border-card-border pt-2.5 text-xs text-muted">
              {{ untrackedCount }} more Sri Lankan banks pending automated scraping integration.
            </p>
          </section>

          <section class="rounded-[16px] border border-card-border bg-card p-5 shadow-sm">
            <h2 class="mb-3 text-base font-bold text-navy">Scrape Activity</h2>
            <div class="flex flex-col divide-y divide-card-border">
              <div v-for="a in scrapeActivity" :key="a.bank + a.category" class="flex items-center justify-between gap-3 py-2.5">
                <div class="min-w-0">
                  <p class="truncate text-sm font-semibold text-navy">{{ a.bank }} ({{ a.category }})</p>
                  <span class="mt-0.5 inline-block rounded-pill bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700">SUCCESS: {{ a.count }} rates</span>
                </div>
                <span class="shrink-0 text-xs text-muted">{{ fmtRelativeDate(a.lastScrape) }}</span>
              </div>
              <p v-if="!loading && scrapeActivity.length === 0" class="py-2.5 text-sm text-muted">No scrape activity recorded yet.</p>
            </div>
          </section>
        </div>

        <!-- Verification & Standards -->
        <h2 class="mb-3 mt-6 text-lg font-bold text-navy">Verification &amp; Standards</h2>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <section class="rounded-[16px] border border-card-border bg-card p-5 shadow-sm">
            <div class="mb-2 flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">1</span>
              <h3 class="text-sm font-bold text-navy">Data Sources</h3>
            </div>
            <ul class="flex flex-col gap-2 text-[13px] text-navy">
              <li class="flex items-start gap-2"><span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-sm bg-primary" />Official commercial bank websites operating in Sri Lanka.</li>
              <li class="flex items-start gap-2"><span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-sm bg-primary" />Official documents and public press releases issued by registered financial institutions.</li>
              <li class="flex items-start gap-2"><span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-sm bg-primary" />Published datasets and statistical summaries from the Central Bank of Sri Lanka (CBSL), where available.</li>
            </ul>
          </section>

          <section class="rounded-[16px] border border-card-border bg-card p-5 shadow-sm">
            <div class="mb-2 flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">2</span>
              <h3 class="text-sm font-bold text-navy">How Often Is Data Checked?</h3>
            </div>
            <table class="w-full text-xs">
              <thead>
                <tr class="text-left text-[10px] uppercase tracking-wide text-muted">
                  <th class="pb-1.5 font-semibold">Product Category</th>
                  <th class="pb-1.5 text-right font-semibold">Check Frequency</th>
                </tr>
              </thead>
              <tbody class="text-navy">
                <tr class="border-t border-card-border"><td class="py-1.5">Fixed Deposit Rates</td><td class="py-1.5 text-right font-semibold">Every 6 hours</td></tr>
                <tr class="border-t border-card-border"><td class="py-1.5">Savings Account Rates</td><td class="py-1.5 text-right font-semibold">Every 6 hours</td></tr>
                <tr class="border-t border-card-border"><td class="py-1.5">Personal &amp; Housing Loan Rates</td><td class="py-1.5 text-right font-semibold">Every 6 hours</td></tr>
              </tbody>
            </table>
          </section>

          <section class="rounded-[16px] border border-card-border bg-card p-5 shadow-sm">
            <div class="mb-2 flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">3</span>
              <h3 class="text-sm font-bold text-navy">What Does Verified Mean?</h3>
            </div>
            <p class="text-[13px] leading-relaxed text-muted">
              Every rate shown carries the timestamp of the scrape that captured it, straight from the bank's own published page — nothing is manually typed in or estimated.
            </p>
          </section>

          <section class="rounded-[16px] border border-card-border bg-card p-5 shadow-sm">
            <div class="mb-2 flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">4</span>
              <h3 class="text-sm font-bold text-navy">Data Limitations</h3>
            </div>
            <p class="text-[13px] leading-relaxed text-muted">
              Rates change frequently and banks may offer preferential terms not reflected here. All figures should be treated as indicative — always confirm directly with the bank before making a decision.
            </p>
          </section>
        </div>

        <!-- Methodology -->
        <section class="mt-4 rounded-[16px] border border-card-border bg-card p-5 shadow-sm sm:p-7">
          <h2 class="mb-4 text-base font-bold text-navy">Methodology</h2>
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-4">
            <div>
              <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">1</span>
              <p class="mt-2 text-sm font-bold text-navy">Collection</p>
              <p class="mt-1 text-xs leading-relaxed text-muted">Automated scraping of official bank websites and published rate sheets. Never third-party aggregators or user-submitted data.</p>
            </div>
            <div>
              <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">2</span>
              <p class="mt-2 text-sm font-bold text-navy">Validation</p>
              <p class="mt-1 text-xs leading-relaxed text-muted">Range checks flag unusual values before publishing. Out-of-range or malformed scrapes are quarantined for manual review.</p>
            </div>
            <div>
              <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">3</span>
              <p class="mt-2 text-sm font-bold text-navy">Verification</p>
              <p class="mt-1 text-xs leading-relaxed text-muted">Every published rate carries a timestamp of when it was last confirmed against the source. Stale rates are clearly marked.</p>
            </div>
            <div>
              <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">4</span>
              <p class="mt-2 text-sm font-bold text-navy">Publishing</p>
              <p class="mt-1 text-xs leading-relaxed text-muted">Rates appear on the site only after passing all checks. Accurate and honest data over completeness.</p>
            </div>
          </div>

          <p class="mt-5 rounded-lg bg-badge-bg px-4 py-3 text-sm font-bold text-primary">
            Core Sourcing Principle: No invented data, anywhere. If we do not have verified data for a specific bank or tenure, we explicitly say so. We never fill gaps with placeholder values.
          </p>
        </section>

        <!-- Report CTA -->
        <section class="mt-4 flex flex-col items-start gap-3 rounded-[14px] border border-card-border bg-card p-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-sm font-bold text-navy">Spot an incorrect rate?</p>
            <p class="mt-0.5 text-xs text-muted">Help us keep the financial ecosystem accurate. Report discrepancies directly to our review queue.</p>
          </div>
          <NuxtLink to="/report-issue" class="shrink-0 rounded-lg bg-primary px-5 py-2.5 text-sm font-bold text-white transition hover:bg-primary/90">
            Go to Report Form &rarr;
          </NuxtLink>
        </section>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
