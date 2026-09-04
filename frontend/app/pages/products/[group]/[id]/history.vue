<script setup lang="ts">
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
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

const { fetchSavings, fetchLoans } = useRatesApi()
const rows = ref<ProductRate[]>([])
const loading = ref(true)
const row = ref<ProductRate | null>(null)

onMounted(async () => {
  try {
    rows.value = await (group === 'savings' ? fetchSavings() : fetchLoans()).catch(() => [])
    row.value = rows.value.find((r) => r.id === id) ?? null
  } finally {
    loading.value = false
  }
})

const bank = computed(() => (row.value ? DIRECTORY_BANKS.find((b) => b.apiName === row.value!.bank_name) : undefined))

function productLabel(r: ProductRate): string {
  return r.tenure_label || r.rate_label || formatCategoryLabel(r.category_code)
}

useHead({
  title: computed(() => (row.value ? `Rate History — ${row.value.bank_name} ${formatCategoryLabel(row.value.category_code)} — FindRate LK` : 'Rate History — FindRate LK'))
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
          <NuxtLink :to="meta.directoryHref">{{ meta.directoryLabel }}</NuxtLink>
          <span>/</span>
          <NuxtLink :to="`/products/${group}/${id}`">{{ row ? `${row.bank_name} ${formatCategoryLabel(row.category_code)}` : 'Product' }}</NuxtLink>
          <span>/</span>
          <span>Rate History</span>
        </nav>

        <p v-if="loading" class="empty">Loading…</p>
        <p v-else-if="!row" class="empty">
          This product isn't in our current dataset.
          <NuxtLink :to="meta.directoryHref">Browse {{ meta.directoryLabel }} &rarr;</NuxtLink>
        </p>

        <template v-else>
          <section class="header-card">
            <div>
              <h1>{{ bank?.displayName ?? row.bank_name }} &mdash; {{ formatCategoryLabel(row.category_code) }}</h1>
              <p v-if="row.tenure_label || row.rate_label" class="product-sub">{{ productLabel(row) }}</p>
              <p class="sub">Rate history for this product. Every scrape is stored, so this view grows more useful over time.</p>
            </div>
            <div class="rate-box">
              <p class="rate-label">Current Rate</p>
              <p class="rate-value">{{ row.interest_rate.toFixed(2) }}% <span class="unit">p.a.</span></p>
            </div>
          </section>

          <section class="panel">
            <h2>Rate Trend</h2>
            <div class="trend-empty">
              <p><strong>Not enough history yet for a trend chart.</strong></p>
              <p>Only one scrape has been recorded for this product so far. Scrapes run twice daily (around 06:00 and 18:00 Sri Lanka time) — a chart will appear here once at least two data points exist, since every scrape is stored rather than overwritten.</p>
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
                  <tr>
                    <td>{{ fmtDate(row.scraped_at) }}</td>
                    <td><strong>{{ row.interest_rate.toFixed(2) }}%</strong></td>
                    <td>
                      <a v-if="row.source_url" :href="row.source_url" target="_blank" rel="noopener">Official page &rarr;</a>
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
  margin: 0 0 0.3rem;
  font-size: 1.3rem;
  font-weight: 800;
}
.product-sub {
  margin: 0 0 0.4rem;
  font-size: 0.85rem;
  color: var(--muted);
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
