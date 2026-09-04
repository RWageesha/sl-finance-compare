<script setup lang="ts">
import { isFdRow, maturityValue, fmtLkr } from '~/utils/fdCompare'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Fixed Deposit Comparison — Table View — FindRate LK' })

const route = useRoute()
const { fetchFixedDeposits } = useRatesApi()

const fdRows = ref<ProductRate[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    fdRows.value = (await fetchFixedDeposits().catch(() => [])).filter(isFdRow)
  } finally {
    loading.value = false
  }
})

const amount = ref(Number(route.query.amount) || 1000000)
const tenure = ref(Number(route.query.tenure) || 12)

// Only the banks we actually scrape can appear as comparison columns —
// there's nothing real to put in a column for an untracked bank.
const TRACKED = DIRECTORY_BANKS.filter((b) => b.tracked)
const selected = ref<string[]>(TRACKED.map((b) => b.slug))

function toggle(slug: string) {
  selected.value = selected.value.includes(slug) ? selected.value.filter((s) => s !== slug) : [...selected.value, slug]
}

interface Column {
  slug: string
  displayName: string
  row: ProductRate | null
  otherTenures: number[]
}

const columns = computed<Column[]>(() => {
  return TRACKED.filter((b) => selected.value.includes(b.slug)).map((b) => {
    const bankRows = fdRows.value.filter((r) => r.bank_name === b.apiName)
    // Prefer the standard product; fall back to whatever's offered at this
    // tenure so a bank without a STANDARD_FD row still shows something real.
    const row =
      bankRows.find((r) => r.category_code === 'STANDARD_FD' && r.tenure_value === tenure.value) ??
      bankRows.find((r) => r.tenure_value === tenure.value) ??
      null
    const otherTenures = [...new Set(bankRows.filter((r) => r.category_code === (row?.category_code ?? 'STANDARD_FD')).map((r) => r.tenure_value))]
      .filter((v): v is number => v != null)
      .sort((a, b2) => a - b2)
    return { slug: b.slug, displayName: b.displayName, row, otherTenures }
  })
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
          <NuxtLink :to="`/compare/fixed-deposits/results?amount=${amount}&tenure=${tenure}`">Compare</NuxtLink>
          <span>/</span>
          <span>Fixed Deposits</span>
        </nav>

        <p class="eyebrow">Product Comparison</p>
        <h1>Fixed Deposit Comparison &mdash; Table View</h1>
        <p class="sub">Side-by-side breakdown of {{ fmtTenure(tenure) }} fixed deposits at Rs. {{ amount.toLocaleString('en-LK') }}, across the banks FindRate LK tracks.</p>

        <div class="bank-picker">
          <label v-for="b in TRACKED" :key="b.slug" class="bank-check">
            <input type="checkbox" :checked="selected.includes(b.slug)" @change="toggle(b.slug)">
            {{ b.displayName }}
          </label>
        </div>

        <p v-if="loading" class="empty">Loading…</p>
        <p v-else-if="columns.length === 0" class="empty">Select at least one bank to compare.</p>

        <div v-else class="table-wrap">
          <table>
            <thead>
              <tr>
                <th class="param-col">Parameter</th>
                <th v-for="c in columns" :key="c.slug">{{ c.displayName }}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="param">Rate ({{ fmtTenure(tenure) }})</td>
                <td v-for="c in columns" :key="c.slug" class="value">
                  <strong v-if="c.row">{{ c.row.interest_rate.toFixed(2) }}% p.a.</strong>
                  <span v-else class="muted">Not offered at this tenure</span>
                </td>
              </tr>
              <tr>
                <td class="param">Category</td>
                <td v-for="c in columns" :key="c.slug" class="value">
                  <span v-if="c.row" class="badge" :class="`badge-${c.row.category_code.toLowerCase()}`">{{ formatCategoryLabel(c.row.category_code) }}</span>
                  <span v-else class="muted">&mdash;</span>
                </td>
              </tr>
              <tr>
                <td class="param">Estimated Maturity</td>
                <td v-for="c in columns" :key="c.slug" class="value">
                  {{ c.row ? fmtLkr(maturityValue(amount, c.row.interest_rate, tenure)) : '—' }}
                </td>
              </tr>
              <tr>
                <td class="param">Other Tenure Options</td>
                <td v-for="c in columns" :key="c.slug" class="value small">
                  {{ c.otherTenures.length ? c.otherTenures.map((t) => fmtTenure(t)).join(', ') : '—' }}
                </td>
              </tr>
              <tr>
                <td class="param">Last Updated</td>
                <td v-for="c in columns" :key="c.slug" class="value small">
                  {{ c.row ? fmtRelativeDate(c.row.scraped_at) : '—' }}
                </td>
              </tr>
              <tr>
                <td class="param">Source</td>
                <td v-for="c in columns" :key="c.slug" class="value small">
                  <a v-if="c.row?.source_url" :href="c.row.source_url" target="_blank" rel="noopener">Official page &rarr;</a>
                  <span v-else class="muted">&mdash;</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
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
.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--accent);
  margin: 0 0 0.3rem;
}
h1 {
  margin: 0 0 0.5rem;
  font-size: 1.5rem;
  font-weight: 800;
}
.sub {
  margin: 0 0 1.2rem;
  max-width: 640px;
  color: var(--muted);
  font-size: 0.88rem;
}
.bank-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1.2rem;
}
.bank-check {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}
.empty {
  padding: 2.5rem;
  text-align: center;
  color: var(--muted);
}
.table-wrap {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow-x: auto;
  margin-bottom: 2.5rem;
}
table {
  width: 100%;
  border-collapse: collapse;
  min-width: 480px;
}
th,
td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
th {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--text);
}
.param-col {
  min-width: 160px;
}
.param {
  font-size: 0.8rem;
  color: var(--muted);
  font-weight: 600;
}
.value {
  font-size: 0.88rem;
}
.value.small {
  font-size: 0.8rem;
}
.value a {
  color: var(--accent);
  font-weight: 600;
  text-decoration: none;
}
.value a:hover {
  text-decoration: underline;
}
.muted {
  color: var(--muted);
}
tbody tr:last-child td {
  border-bottom: none;
}
</style>
