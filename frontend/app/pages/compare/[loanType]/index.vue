<script setup lang="ts">
// Housing and Personal Loan comparison share this one page (structurally
// identical: amount+tenure inputs driving a real EMI projection, a list of
// real rows for that category). Gold Loans and Savings have no comparable
// amount-based calculation (no LTV or balance data), so they're separate,
// simpler pages. "fixed-deposits" is a static sibling route and always
// takes precedence over this dynamic segment, so there's no collision.
import { productDetailHref, emiPayment, fmtLkr, bankForApiName } from '~/utils/fdCompare'
import type { ProductRate } from '~/composables/useRatesApi'

const LOAN_TYPE_META = {
  'housing-loans': {
    category: 'HOUSING_LOAN',
    tabId: 'housing-loans' as const,
    title: 'Housing Loan Comparison',
    desc: 'Compare structured residential housing finance products from banks FindRate LK tracks in Sri Lanka.',
    defaultTenure: 240
  },
  'personal-loans': {
    category: 'PERSONAL_LOAN',
    tabId: 'personal-loans' as const,
    title: 'Personal Loan Comparison',
    desc: 'Compare personal loan rates from banks FindRate LK tracks in Sri Lanka.',
    defaultTenure: 60
  }
} as const

const route = useRoute()
const loanType = String(route.params.loanType) as keyof typeof LOAN_TYPE_META
if (!(loanType in LOAN_TYPE_META)) {
  throw createError({ statusCode: 404, statusMessage: 'Not found', fatal: true })
}
const meta = LOAN_TYPE_META[loanType]

useHead({ title: `${meta.title} — FindRate LK` })

const { fetchLoans } = useRatesApi()
const rows = ref<ProductRate[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    rows.value = (await fetchLoans().catch(() => [])).filter((r) => r.category_code === meta.category)
  } finally {
    loading.value = false
  }
})

const amount = ref(1000000)
const tenure = ref(meta.defaultTenure)
const TENURE_OPTIONS = [12, 24, 36, 60, 120, 180, 240, 300, 360]
const sortBy = ref<'rate_asc' | 'rate_desc'>('rate_asc')

const sorted = computed(() =>
  [...rows.value].sort((a, b) => (sortBy.value === 'rate_asc' ? a.interest_rate - b.interest_rate : b.interest_rate - a.interest_rate))
)

function productLabel(r: ProductRate): string {
  const parts = [r.tenure_label, r.rate_label].filter(Boolean)
  return parts.length ? parts.join(' — ') : formatCategoryLabel(r.category_code)
}
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <p class="eyebrow">Financial Products</p>
        <h1>{{ meta.title }}</h1>
        <p class="sub">{{ meta.desc }}</p>

        <CompareTabs :active="meta.tabId" />

        <div class="filter-bar">
          <div class="field">
            <label for="amount">Loan Amount (LKR)</label>
            <input id="amount" v-model.number="amount" type="number" min="0" step="10000">
          </div>
          <div class="field">
            <label for="tenure">Tenure</label>
            <select id="tenure" v-model.number="tenure">
              <option v-for="t in TENURE_OPTIONS" :key="t" :value="t">{{ fmtTenure(t) }}</option>
            </select>
          </div>
        </div>

        <div class="results-header">
          <span class="count">
            <template v-if="loading">Loading…</template>
            <template v-else>{{ sorted.length }} product{{ sorted.length === 1 ? '' : 's' }} found</template>
          </span>
          <label class="sort-label">
            Sort by:
            <select v-model="sortBy">
              <option value="rate_asc">Lowest Interest Rate</option>
              <option value="rate_desc">Highest Interest Rate</option>
            </select>
          </label>
        </div>

        <p v-if="!loading && sorted.length === 0" class="empty">No {{ meta.title.toLowerCase() }} products currently tracked.</p>

        <div v-else class="card-list">
          <div v-for="r in sorted" :key="r.id" class="loan-card">
            <div class="loan-icon"><BankLogo :bank="bankForApiName(r.bank_name)" /></div>
            <div class="loan-main">
              <h3>{{ r.bank_name }}</h3>
              <p class="loan-sub">{{ productLabel(r) }}</p>
            </div>
            <div class="loan-stat">
              <span class="stat-label">Interest Rate</span>
              <strong class="stat-value">{{ r.interest_rate.toFixed(2) }}% p.a.</strong>
            </div>
            <div class="loan-stat">
              <span class="stat-label">Est. Monthly Payment</span>
              <strong class="stat-value">{{ fmtLkr(emiPayment(amount, r.interest_rate, tenure)) }}</strong>
            </div>
            <div class="loan-meta">
              <span class="updated">Updated {{ fmtRelativeDate(r.scraped_at) }}</span>
              <NuxtLink v-if="productDetailHref(r)" class="detail-link" :to="productDetailHref(r)!">View Details</NuxtLink>
            </div>
          </div>
        </div>

        <p class="calc-note">Est. Monthly Payment is a projection using each bank's disclosed rate at the amount and tenure above — not a bank-confirmed quote. Actual eligibility, fees, and terms vary; check the linked source before deciding.</p>
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
  min-width: 160px;
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
  background: #fff;
  border: 1px solid var(--border);
  color: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 5px;
  overflow: hidden;
}
.loan-icon svg {
  width: 18px;
  height: 18px;
}
.loan-main {
  flex: 1 1 200px;
  min-width: 0;
}
.loan-main h3 {
  margin: 0 0 0.1rem;
  font-size: 0.92rem;
  font-weight: 700;
}
.loan-sub {
  margin: 0;
  font-size: 0.78rem;
  color: var(--muted);
  max-width: 340px;
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
