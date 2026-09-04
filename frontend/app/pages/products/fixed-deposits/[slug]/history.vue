<script setup lang="ts">
import { isFdRow, parseFdSlug, findFdRow } from '~/utils/fdCompare'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import type { ProductRate } from '~/composables/useRatesApi'

const route = useRoute()
const slug = String(route.params.slug)
const parsed = parseFdSlug(slug)
if (!parsed) {
  throw createError({ statusCode: 404, statusMessage: 'Product not found', fatal: true })
}

const { fetchFixedDeposits, fetchRateHistory } = useRatesApi()
const fdRows = ref<ProductRate[]>([])
const loading = ref(true)
const row = ref<ProductRate | null>(null)
// Oldest first, as returned by the API — every scrape ever recorded for
// this exact product line, not just the latest one `row` holds.
const history = ref<ProductRate[]>([])

onMounted(async () => {
  try {
    fdRows.value = (await fetchFixedDeposits().catch(() => [])).filter(isFdRow)
    row.value = findFdRow(fdRows.value, parsed!) ?? null
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

const chartPoints = computed(() => history.value.map((r) => ({ date: r.scraped_at, rate: r.interest_rate })))
// Newest first for the table, so the most recent scrape reads at the top.
const historyDesc = computed(() => [...history.value].reverse())

const bank = computed(() => DIRECTORY_BANKS.find((b) => b.slug === parsed!.bankSlug))

useHead({
  title: computed(() =>
    row.value ? `Rate History — ${bank.value?.displayName ?? row.value.bank_name} ${fmtTenure(row.value.tenure_value ?? 0)} FD — FindRate LK` : 'Rate History — FindRate LK'
  )
})
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <nav class="breadcrumb" aria-label="Breadcrumb">
          <NuxtLink to="/">Home</NuxtLink>
          <span>/</span>
          <NuxtLink to="/compare/fixed-deposits">Fixed Deposits</NuxtLink>
          <span>/</span>
          <NuxtLink :to="`/products/fixed-deposits/${slug}`">{{ bank ? `${bank.displayName} ${row ? fmtTenure(row.tenure_value ?? 0) : ''} FD` : 'Product' }}</NuxtLink>
          <span>/</span>
          <span>Rate History</span>
        </nav>

        <p v-if="loading" class="empty">Loading…</p>
        <p v-else-if="!row" class="empty">
          This product isn't in our current dataset.
          <NuxtLink to="/compare/fixed-deposits">Browse Fixed Deposits &rarr;</NuxtLink>
        </p>

        <template v-else>
          <section class="header-card">
            <div>
              <h1>{{ bank?.displayName ?? row.bank_name }} {{ fmtTenure(row.tenure_value ?? 0) }} Fixed Deposit</h1>
              <p class="sub">Rate history for this {{ formatCategoryLabel(row.category_code) }} product. Every scrape is stored, so this view grows more useful over time.</p>
            </div>
            <div class="rate-box">
              <p class="rate-label">Current Rate</p>
              <p class="rate-value">{{ row.interest_rate.toFixed(2) }}% <span class="unit">p.a.</span></p>
            </div>
          </section>

          <section class="panel">
            <h2>Rate Trend</h2>
            <RateTrendChart v-if="chartPoints.length > 1" :points="chartPoints" />
            <div v-else class="trend-empty">
              <p><strong>Not enough history yet for a trend chart.</strong></p>
              <p>Only one scrape has been recorded for this product so far. Scrapes run four times daily (around 3:15am, 9:15am, 3:15pm, and 9:15pm Sri Lanka time) — a chart will appear here once at least two data points exist, since every scrape is stored rather than overwritten.</p>
            </div>
          </section>

          <section class="panel">
            <h2>Recorded Data Points</h2>
            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Scraped At</th>
                    <th>Rate</th>
                    <th>Source</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="h in (historyDesc.length ? historyDesc : [row])" :key="h.id">
                    <td>{{ fmtDate(h.scraped_at) }}</td>
                    <td><strong>{{ h.interest_rate.toFixed(2) }}%</strong></td>
                    <td>
                      <a v-if="h.source_url" :href="h.source_url" target="_blank" rel="noopener">Official page &rarr;</a>
                      <span v-else class="muted">&mdash;</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </template>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>

<style scoped>
.wrap {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 1.5rem;
}
main {
  padding: 0.5rem 0 1rem;
}
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.82rem;
  color: var(--muted);
  margin: 1.3rem 0 1.1rem;
  flex-wrap: wrap;
}
.breadcrumb a {
  color: var(--muted);
  text-decoration: none;
}
.breadcrumb a:hover {
  color: var(--accent);
}
.breadcrumb span:last-child {
  color: var(--text);
  font-weight: 600;
}
.empty {
  padding: 2.5rem;
  text-align: center;
  color: var(--muted);
}
.empty a {
  display: block;
  margin-top: 0.6rem;
  color: var(--accent);
  font-weight: 600;
  text-decoration: none;
}

.header-card {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.4rem 1.6rem;
  flex-wrap: wrap;
  margin-bottom: 1.2rem;
}
.header-card h1 {
  margin: 0 0 0.4rem;
  font-size: 1.3rem;
  font-weight: 800;
}
.sub {
  margin: 0;
  max-width: 520px;
  font-size: 0.85rem;
  color: var(--muted);
  line-height: 1.5;
}
.rate-box {
  text-align: right;
}
.rate-label {
  margin: 0 0 0.2rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}
.rate-value {
  margin: 0;
  font-size: 1.7rem;
  font-weight: 800;
  color: var(--accent);
}
.rate-value .unit {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--muted);
}

.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.2rem 1.3rem;
  margin-bottom: 1.2rem;
}
.panel h2 {
  margin: 0 0 0.8rem;
  font-size: 1rem;
  font-weight: 700;
}
.trend-empty {
  background: var(--bg);
  border: 1px dashed var(--border);
  border-radius: 8px;
  padding: 1.5rem;
  text-align: center;
}
.trend-empty p {
  margin: 0 0 0.4rem;
  font-size: 0.85rem;
  color: var(--muted);
  max-width: 480px;
  margin-left: auto;
  margin-right: auto;
}
.trend-empty p:last-child {
  margin-bottom: 0;
}

.table-wrap {
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  min-width: 400px;
}
th,
td {
  padding: 0.65rem 0.9rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
  font-size: 0.85rem;
}
th {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
}
tbody tr:last-child td {
  border-bottom: none;
}
td a {
  color: var(--accent);
  font-weight: 600;
  text-decoration: none;
}
td a:hover {
  text-decoration: underline;
}
.muted {
  color: var(--muted);
}
</style>
