<script setup lang="ts">
import { isFdRow, bankSlugForApiName, fdSlug, maturityValue, fmtLkr, bankForApiName } from '~/utils/fdCompare'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Fixed Deposit Comparison — FindRate LK' })

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
const sortBy = ref<'rate_desc' | 'rate_asc'>('rate_desc')

const tenureOptions = computed(() => {
  const values = new Set(fdRows.value.map((r) => r.tenure_value).filter((v): v is number => v != null))
  return [...values].sort((a, b) => a - b)
})

function applyFilters() {
  navigateTo({ path: '/compare/fixed-deposits/results', query: { amount: String(amount.value), tenure: String(tenure.value) } })
}

const matches = computed(() => {
  const rows = fdRows.value.filter((r) => r.tenure_value === tenure.value)
  return [...rows].sort((a, b) => (sortBy.value === 'rate_desc' ? b.interest_rate - a.interest_rate : a.interest_rate - b.interest_rate))
})

function detailHref(r: ProductRate): string | null {
  const bankSlug = bankSlugForApiName(r.bank_name)
  if (!bankSlug) return null
  return `/products/fixed-deposits/${fdSlug(bankSlug, r.category_code, r.tenure_value ?? 0)}`
}
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <h1>Fixed Deposit Comparison</h1>
        <p class="sub">Compare fixed deposit rates from tracked commercial banks in Sri Lanka. Adjust your deposit amount and tenure to view options.</p>

        <div class="filter-bar">
          <div class="field">
            <label for="amount">Deposit Amount (LKR)</label>
            <input id="amount" v-model.number="amount" type="number" min="0" step="10000">
          </div>
          <div class="field">
            <label for="tenure">Tenure</label>
            <select id="tenure" v-model.number="tenure">
              <option v-for="t in tenureOptions" :key="t" :value="t">{{ fmtTenure(t) }}</option>
            </select>
          </div>
          <button type="button" class="apply-btn" @click="applyFilters">Compare</button>
        </div>

        <div class="results-header">
          <span class="count">
            <template v-if="loading">Loading…</template>
            <template v-else>{{ matches.length }} product{{ matches.length === 1 ? '' : 's' }} found</template>
          </span>
          <div class="results-actions">
            <label class="sort-label">
              Sort by:
              <select v-model="sortBy">
                <option value="rate_desc">Highest Rate</option>
                <option value="rate_asc">Lowest Rate</option>
              </select>
            </label>
            <NuxtLink class="table-link" :to="`/compare/fixed-deposits/table?amount=${amount}&tenure=${tenure}`">View as Table &rarr;</NuxtLink>
          </div>
        </div>

        <p v-if="!loading && matches.length === 0" class="empty">No fixed deposits found at this tenure. Try a different one.</p>

        <div v-else class="card-list">
          <div v-for="r in matches" :key="r.id" class="fd-card">
            <div class="fd-icon"><BankLogo :bank="bankForApiName(r.bank_name)" /></div>
            <div class="fd-main">
              <h3>{{ r.bank_name }}</h3>
              <p class="fd-sub">{{ fmtTenure(r.tenure_value ?? 0) }} Fixed Deposit &middot; {{ formatCategoryLabel(r.category_code) }}</p>
            </div>
            <div class="fd-stat">
              <span class="stat-label">Interest Rate</span>
              <strong class="stat-value">{{ r.interest_rate.toFixed(2) }}% p.a.</strong>
            </div>
            <div class="fd-stat">
              <span class="stat-label">Maturity Value</span>
              <strong class="stat-value">{{ fmtLkr(maturityValue(amount, r.interest_rate, r.tenure_value ?? 0)) }}</strong>
            </div>
            <div class="fd-meta">
              <span class="updated">Updated {{ fmtRelativeDate(r.scraped_at) }}</span>
              <NuxtLink v-if="detailHref(r)" class="detail-link" :to="detailHref(r)!">View Details</NuxtLink>
            </div>
          </div>
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
  padding: 1.5rem 0 1rem;
}
h1 {
  margin: 0 0 0.4rem;
  font-size: 1.6rem;
  font-weight: 800;
}
.sub {
  margin: 0 0 1.3rem;
  max-width: 640px;
  color: var(--muted);
  font-size: 0.88rem;
}

.filter-bar {
  display: flex;
  align-items: flex-end;
  gap: 1rem;
  flex-wrap: wrap;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem 1.2rem;
  margin-bottom: 1.3rem;
}
.field {
  flex: 1;
  min-width: 140px;
}
.field label {
  display: block;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--muted);
  margin-bottom: 0.3rem;
}
.field input,
.field select {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--bg);
  font: inherit;
  font-size: 0.88rem;
  padding: 0.5rem 0.65rem;
  color: var(--text);
}
.apply-btn {
  font: inherit;
  font-weight: 700;
  font-size: 0.85rem;
  padding: 0.55rem 1.2rem;
  border-radius: 7px;
  border: none;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
  white-space: nowrap;
}

.results-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.8rem;
  margin-bottom: 1rem;
}
.count {
  font-size: 0.85rem;
  font-weight: 600;
}
.results-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.sort-label {
  font-size: 0.8rem;
  color: var(--muted);
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.sort-label select {
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--panel);
  color: var(--text);
  font: inherit;
  font-size: 0.8rem;
  padding: 0.3rem 0.5rem;
}
.table-link {
  color: var(--accent);
  font-weight: 600;
  font-size: 0.82rem;
  text-decoration: none;
}
.table-link:hover {
  text-decoration: underline;
}

.empty {
  padding: 2.5rem;
  text-align: center;
  color: var(--muted);
}
.card-list {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
  margin-bottom: 2.5rem;
}
.fd-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem 1.2rem;
  flex-wrap: wrap;
}
.fd-icon {
  width: 38px;
  height: 38px;
  flex: none;
  border-radius: 8px;
  background: #fff;
  border: 1px solid var(--border);
  color: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 5px;
  overflow: hidden;
}
.fd-icon svg {
  width: 18px;
  height: 18px;
}
.fd-main {
  flex: 1 1 200px;
  min-width: 0;
}
.fd-main h3 {
  margin: 0 0 0.1rem;
  font-size: 0.92rem;
  font-weight: 700;
}
.fd-sub {
  margin: 0;
  font-size: 0.78rem;
  color: var(--muted);
}
.fd-stat {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 110px;
}
.stat-label {
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
}
.stat-value {
  font-size: 0.95rem;
  font-weight: 800;
}
.fd-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.4rem;
  margin-left: auto;
}
.updated {
  font-size: 0.72rem;
  color: var(--muted);
  white-space: nowrap;
}
.detail-link {
  font-size: 0.8rem;
  font-weight: 700;
  padding: 0.4rem 0.85rem;
  border-radius: 6px;
  background: var(--accent);
  color: #fff;
  text-decoration: none;
  white-space: nowrap;
}
</style>
