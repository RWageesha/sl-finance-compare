<script setup lang="ts">
// Shared detail page for Savings and Loan products (Fixed Deposits have
// their own dedicated page at pages/products/fixed-deposits/[slug]/ since
// tenure genuinely identifies one logical FD product — see fdCompare.ts
// for why that scheme doesn't work here). Static routes take precedence
// over this dynamic [group] segment in Nuxt's router, so /products/
// fixed-deposits/... always resolves to that page, never this one.
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import type { ProductRate } from '~/composables/useRatesApi'

const GROUP_META = {
  savings: {
    noun: 'Savings Account',
    directoryLabel: 'Savings Accounts',
    directoryHref: '/rates?tab=savings',
    disclosure: 'minimum balance, account fees, or other terms'
  },
  loans: {
    noun: 'Loan',
    directoryLabel: 'Loans',
    directoryHref: '/rates?tab=loans',
    disclosure: 'processing fees, collateral or eligibility requirements, or other terms'
  }
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

useHead({
  title: computed(() =>
    row.value ? `${row.value.bank_name} — ${formatCategoryLabel(row.value.category_code)} — OpenFinance LK` : `${meta.noun} — OpenFinance LK`
  )
})

// Real product label: whatever the bank's own tenure/rate label says
// (e.g. "Below Rs. 500,000/-", "Fixed Rate"), falling back to the
// category name when neither is present.
function productLabel(r: ProductRate): string {
  return r.tenure_label || r.rate_label || formatCategoryLabel(r.category_code)
}

const similar = computed(() => {
  if (!row.value) return []
  return rows.value.filter((r) => r.bank_name === row.value!.bank_name && r.category_code === row.value!.category_code && r.id !== row.value!.id)
})

const reportUrl = computed(() => {
  if (!row.value) return 'https://github.com/RWageesha/sl-finance-compare/issues/new'
  const title = encodeURIComponent(`Incorrect rate: ${row.value.bank_name} ${formatCategoryLabel(row.value.category_code)} (id ${row.value.id})`)
  return `https://github.com/RWageesha/sl-finance-compare/issues/new?title=${title}`
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
          <span v-if="row">{{ row.bank_name }} {{ formatCategoryLabel(row.category_code) }}</span>
        </nav>

        <p v-if="loading" class="empty">Loading…</p>
        <p v-else-if="!row" class="empty">
          This product isn't in our current dataset — it may have been part of a scrape that's since rotated out, or the link is out of date.
          <NuxtLink :to="meta.directoryHref">Browse {{ meta.directoryLabel }} &rarr;</NuxtLink>
        </p>

        <template v-else>
          <section class="header-card">
            <div class="header-left">
              <div class="bank-line">
                <div class="bank-icon"><Icon name="bank" /></div>
                <NuxtLink v-if="bank" :to="`/banks/${bank.slug}`" class="bank-name">{{ bank.displayName }}</NuxtLink>
                <span v-else class="bank-name">{{ row.bank_name }}</span>
              </div>
              <h1>{{ formatCategoryLabel(row.category_code) }}</h1>
              <p v-if="row.tenure_label || row.rate_label" class="product-sub">{{ productLabel(row) }}</p>
              <p class="updated-line">Last updated: {{ fmtDate(row.scraped_at) }}</p>
            </div>
            <div class="header-right">
              <p class="rate-label">Interest Rate</p>
              <p class="rate-value">{{ row.interest_rate.toFixed(2) }}% <span class="unit">p.a.</span></p>
            </div>
          </section>

          <div class="content-grid">
            <div class="main-col">
              <section class="panel">
                <h2>Key Information</h2>
                <div class="info-grid">
                  <div class="info-item">
                    <span class="info-label">Category</span>
                    <span class="badge" :class="`badge-${row.category_code.toLowerCase()}`">{{ formatCategoryLabel(row.category_code) }}</span>
                  </div>
                  <div v-if="row.tenure_label" class="info-item">
                    <span class="info-label">{{ group === 'savings' ? 'Balance Tier' : 'Tenure' }}</span>
                    <strong>{{ row.tenure_label }}</strong>
                  </div>
                  <div v-if="row.rate_label" class="info-item">
                    <span class="info-label">Rate Label</span>
                    <strong>{{ row.rate_label }}</strong>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Bank</span>
                    <strong>{{ row.bank_name }}</strong>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Last Verified Scrape</span>
                    <strong>{{ fmtRelativeDate(row.scraped_at) }}</strong>
                  </div>
                </div>
                <p class="disclosure-note">
                  OpenFinance LK doesn't yet capture {{ meta.disclosure }} for this product.
                  <a v-if="row.source_url" :href="row.source_url" target="_blank" rel="noopener">See {{ bank?.displayName ?? row.bank_name }}'s official page</a> for full terms before making a decision.
                </p>
              </section>

              <section v-if="similar.length" class="panel">
                <h2>Compare Similar {{ meta.directoryLabel }}</h2>
                <p class="panel-sub">Other {{ formatCategoryLabel(row.category_code) }} products at {{ row.bank_name }}.</p>
                <div class="similar-list">
                  <NuxtLink v-for="r in similar" :key="r.id" class="similar-item" :to="`/products/${group}/${r.id}`">
                    <span>{{ productLabel(r) }}</span>
                    <strong>{{ r.interest_rate.toFixed(2) }}% p.a.</strong>
                  </NuxtLink>
                </div>
              </section>
            </div>

            <aside class="side-col">
              <div class="panel">
                <h2>Rate History</h2>
                <p class="panel-sub">1 verified data point on file so far.</p>
                <div class="history-row">
                  <span>{{ fmtDate(row.scraped_at) }}</span>
                  <strong>{{ row.interest_rate.toFixed(2) }}%</strong>
                </div>
                <p class="history-note">Scrapes run twice daily (around 06:00 and 18:00 Sri Lanka time). A trend will build up here once at least two are recorded — every scrape is stored, not overwritten.</p>
                <NuxtLink class="history-link" :to="`/products/${group}/${id}/history`">View Full Timeline &rarr;</NuxtLink>
              </div>

              <div class="panel">
                <h2>Source &amp; Verification</h2>
                <p class="side-label">Data Source</p>
                <a v-if="row.source_url" class="source-link" :href="row.source_url" target="_blank" rel="noopener">{{ row.source_url }}</a>
                <p class="side-label" style="margin-top: 0.8rem">Last Scraped</p>
                <p class="side-value">{{ fmtDate(row.scraped_at) }}</p>
              </div>
            </aside>
          </div>

          <div class="report-row">
            <p>Noticed an inaccuracy in the rate above?</p>
            <a class="report-btn" :href="reportUrl" target="_blank" rel="noopener">Report Incorrect Information</a>
          </div>
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
}
.bank-line {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}
.bank-icon {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: #e6f0ff;
  color: #1a4fb4;
  display: flex;
  align-items: center;
  justify-content: center;
}
[data-theme='dark'] .bank-icon {
  background: #17335f;
  color: #a8c6ff;
}
.bank-icon svg {
  width: 14px;
  height: 14px;
}
.bank-name {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--accent);
  text-decoration: none;
}
.header-left h1 {
  margin: 0 0 0.2rem;
  font-size: 1.4rem;
  font-weight: 800;
}
.product-sub {
  margin: 0 0 0.3rem;
  font-size: 0.85rem;
  color: var(--muted);
}
.updated-line {
  margin: 0;
  font-size: 0.78rem;
  color: var(--muted);
}
.header-right {
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
  font-size: 2rem;
  font-weight: 800;
  color: var(--accent);
}
.rate-value .unit {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--muted);
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: 1.5rem;
  margin: 1.5rem 0 2rem;
  align-items: start;
}
@media (max-width: 900px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}
.main-col,
.side-col {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.2rem 1.3rem;
}
.panel h2 {
  margin: 0 0 0.7rem;
  font-size: 1rem;
  font-weight: 700;
}
.panel-sub {
  margin: -0.3rem 0 0.8rem;
  font-size: 0.8rem;
  color: var(--muted);
}
.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.9rem;
  margin-bottom: 1rem;
}
.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.info-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
}
.disclosure-note {
  margin: 0;
  font-size: 0.8rem;
  color: var(--muted);
  line-height: 1.5;
  padding-top: 0.8rem;
  border-top: 1px solid var(--border);
}
.disclosure-note a {
  color: var(--accent);
  font-weight: 600;
  text-decoration: none;
}
.disclosure-note a:hover {
  text-decoration: underline;
}

.similar-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.similar-item {
  display: flex;
  justify-content: space-between;
  padding: 0.6rem 0.8rem;
  border: 1px solid var(--border);
  border-radius: 7px;
  text-decoration: none;
  color: var(--text);
  font-size: 0.85rem;
  gap: 1rem;
}
.similar-item:hover {
  border-color: var(--accent);
}
.similar-item span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.similar-item strong {
  color: var(--accent);
  white-space: nowrap;
}

.history-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  font-size: 0.85rem;
}
.history-note {
  margin: 0.6rem 0 0;
  font-size: 0.76rem;
  color: var(--muted);
  line-height: 1.5;
}
.history-link {
  display: inline-block;
  margin-top: 0.7rem;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
}
.history-link:hover {
  text-decoration: underline;
}

.side-label {
  margin: 0 0 0.2rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}
.side-value {
  margin: 0;
  font-size: 0.85rem;
  font-weight: 600;
}
.source-link {
  display: block;
  font-size: 0.8rem;
  color: var(--accent);
  text-decoration: none;
  word-break: break-all;
}
.source-link:hover {
  text-decoration: underline;
}

.report-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.8rem;
  padding: 1.2rem 0 2.5rem;
  border-top: 1px solid var(--border);
}
.report-row p {
  margin: 0;
  font-size: 0.85rem;
  color: var(--muted);
}
.report-btn {
  font-size: 0.85rem;
  font-weight: 700;
  padding: 0.6rem 1.1rem;
  border-radius: 7px;
  border: 1px solid var(--border);
  color: var(--text);
  text-decoration: none;
  white-space: nowrap;
}
.report-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}
</style>
