<script setup lang="ts">
import { productDetailHref } from '~/utils/fdCompare'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Savings Account Comparison — OpenFinance LK' })

const { fetchSavings } = useRatesApi()
const rows = ref<ProductRate[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    rows.value = await fetchSavings().catch(() => [])
  } finally {
    loading.value = false
  }
})

const searchQuery = ref('')
const sortBy = ref<'rate_desc' | 'rate_asc'>('rate_desc')

const filtered = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  const matched = q ? rows.value.filter((r) => r.bank_name.toLowerCase().includes(q)) : rows.value
  return [...matched].sort((a, b) => (sortBy.value === 'rate_desc' ? b.interest_rate - a.interest_rate : a.interest_rate - b.interest_rate))
})

// Balance tier / eligibility text is real — scraped straight from the
// bank's own rate table (e.g. "Below Rs. 500,000/-", "Above 60 Years") —
// unlike a fabricated "minimum balance" or "account fee" figure, which we
// don't have anywhere in the schema.
function productLabel(r: ProductRate): string {
  return r.tenure_label || r.rate_label || formatCategoryLabel(r.category_code)
}
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <p class="eyebrow">Financial Products</p>
        <h1>Savings Account Comparison</h1>
        <p class="sub">Compare savings account interest rates across banks OpenFinance LK tracks in Sri Lanka.</p>

        <CompareTabs active="savings" />

        <div class="filter-bar">
          <div class="field-input">
            <Icon name="search" />
            <input v-model="searchQuery" type="text" placeholder="Search banks...">
          </div>
        </div>

        <div class="results-header">
          <span class="count">
            <template v-if="loading">Loading…</template>
            <template v-else>{{ filtered.length }} account{{ filtered.length === 1 ? '' : 's' }} found</template>
          </span>
          <label class="sort-label">
            Sort by:
            <select v-model="sortBy">
              <option value="rate_desc">Highest Interest Rate</option>
              <option value="rate_asc">Lowest Interest Rate</option>
            </select>
          </label>
        </div>

        <p v-if="!loading && filtered.length === 0" class="empty">No savings accounts match.</p>

        <div v-else class="card-list">
          <div v-for="r in filtered" :key="r.id" class="loan-card">
            <div class="loan-icon"><Icon name="savings" /></div>
            <div class="loan-main">
              <h3>{{ r.bank_name }}</h3>
              <p class="loan-sub">
                <span class="badge" :class="`badge-${r.category_code.toLowerCase()}`">{{ formatCategoryLabel(r.category_code) }}</span>
                <span v-if="r.tenure_label || r.rate_label"> &middot; {{ productLabel(r) }}</span>
              </p>
            </div>
            <div class="loan-stat">
              <span class="stat-label">Interest Rate</span>
              <strong class="stat-value">{{ r.interest_rate.toFixed(2) }}% p.a.</strong>
            </div>
            <div class="loan-meta">
              <span class="updated">Updated {{ fmtRelativeDate(r.scraped_at) }}</span>
              <NuxtLink v-if="productDetailHref(r)" class="detail-link" :to="productDetailHref(r)!">View Details</NuxtLink>
            </div>
          </div>
        </div>

        <p class="calc-note">OpenFinance LK doesn't yet capture minimum balance requirements or account fees for savings accounts — see each bank's linked source for that.</p>
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
.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--accent);
  margin: 0 0 0.3rem;
}
h1 {
  margin: 0 0 0.4rem;
  font-size: 1.6rem;
  font-weight: 800;
}
.sub {
  margin: 0 0 1.2rem;
  max-width: 680px;
  color: var(--muted);
  font-size: 0.88rem;
}

.filter-bar {
  margin-bottom: 1.3rem;
}
.field-input {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--panel);
  padding: 0 0.8rem;
  max-width: 340px;
}
.field-input svg {
  width: 15px;
  height: 15px;
  color: var(--muted);
  flex: none;
}
.field-input input {
  border: none;
  outline: none;
  background: transparent;
  font: inherit;
  font-size: 0.85rem;
  color: var(--text);
  padding: 0.6rem 0;
  flex: 1;
  min-width: 0;
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

.empty {
  padding: 2.5rem;
  text-align: center;
  color: var(--muted);
}
.card-list {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
  margin-bottom: 1rem;
}
.loan-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem 1.2rem;
  flex-wrap: wrap;
}
.loan-icon {
  width: 38px;
  height: 38px;
  flex: none;
  border-radius: 8px;
  background: #e0f7fa;
  color: #00707a;
  display: flex;
  align-items: center;
  justify-content: center;
}
[data-theme='dark'] .loan-icon {
  background: #0d3a3e;
  color: #7fe0eb;
}
.loan-icon svg {
  width: 18px;
  height: 18px;
}
.loan-main {
  flex: 1 1 220px;
  min-width: 0;
}
.loan-main h3 {
  margin: 0 0 0.3rem;
  font-size: 0.92rem;
  font-weight: 700;
}
.loan-sub {
  margin: 0;
  font-size: 0.78rem;
  color: var(--muted);
  display: flex;
  align-items: center;
  gap: 0.3rem;
  flex-wrap: wrap;
}
.loan-stat {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 130px;
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
.loan-meta {
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
.calc-note {
  font-size: 0.76rem;
  color: var(--muted);
  line-height: 1.5;
  margin: 0 0 2rem;
}
</style>
