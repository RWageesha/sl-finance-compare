<script setup lang="ts">
import { isFdRow, parseFdSlug, findFdRow, fdSlug } from '~/utils/fdCompare'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import type { ProductRate } from '~/composables/useRatesApi'

const route = useRoute()
const slug = String(route.params.slug)
const parsed = parseFdSlug(slug)
if (!parsed) {
  throw createError({ statusCode: 404, statusMessage: 'Product not found', fatal: true })
}

const { fetchFixedDeposits } = useRatesApi()
const fdRows = ref<ProductRate[]>([])
const loading = ref(true)

const row = ref<ProductRate | null>(null)

onMounted(async () => {
  try {
    fdRows.value = (await fetchFixedDeposits().catch(() => [])).filter(isFdRow)
    row.value = findFdRow(fdRows.value, parsed!) ?? null
  } finally {
    loading.value = false
  }
})

const bank = computed(() => DIRECTORY_BANKS.find((b) => b.slug === parsed!.bankSlug))

useHead({
  title: computed(() =>
    row.value ? `${bank.value?.displayName ?? row.value.bank_name} — ${fmtTenure(row.value.tenure_value ?? 0)} Fixed Deposit — OpenFinance LK` : 'Fixed Deposit — OpenFinance LK'
  )
})

// Other tenures of the same product (same bank + category) — real, drawn
// from the same fetch, not a separate "similar products" algorithm.
const similar = computed(() => {
  if (!row.value) return []
  return fdRows.value
    .filter((r) => r.bank_name === row.value!.bank_name && r.category_code === row.value!.category_code && r.id !== row.value!.id)
    .sort((a, b) => (a.tenure_value ?? 0) - (b.tenure_value ?? 0))
})

function similarHref(r: ProductRate): string {
  return `/products/fixed-deposits/${fdSlug(parsed!.bankSlug, r.category_code, r.tenure_value ?? 0)}`
}

const reportUrl = computed(() => {
  if (!row.value) return 'https://github.com/RWageesha/sl-finance-compare/issues/new'
  const title = encodeURIComponent(`Incorrect rate: ${row.value.bank_name} ${fmtTenure(row.value.tenure_value ?? 0)} ${formatCategoryLabel(row.value.category_code)}`)
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
          <NuxtLink to="/compare/fixed-deposits">Fixed Deposits</NuxtLink>
          <span>/</span>
          <span v-if="bank">{{ bank.displayName }} {{ row ? fmtTenure(row.tenure_value ?? 0) : '' }} Fixed Deposit</span>
        </nav>

        <p v-if="loading" class="empty">Loading…</p>
        <p v-else-if="!row" class="empty">
          This product isn't in our current dataset — it may have been part of a scrape that's since rotated out, or the link is out of date.
          <NuxtLink to="/compare/fixed-deposits">Browse Fixed Deposits &rarr;</NuxtLink>
        </p>

        <template v-else>
          <section class="header-card">
            <div class="header-left">
              <div class="bank-line">
                <div class="bank-icon"><Icon name="bank" /></div>
                <NuxtLink v-if="bank" :to="`/banks/${bank.slug}`" class="bank-name">{{ bank.displayName }}</NuxtLink>
                <span v-else class="bank-name">{{ row.bank_name }}</span>
              </div>
              <h1>{{ fmtTenure(row.tenure_value ?? 0) }} Fixed Deposit</h1>
              <p class="updated-line">Last updated: {{ fmtDate(row.scraped_at) }}</p>
            </div>
            <div class="header-right">
              <p class="rate-label">Annual Interest Rate</p>
              <p class="rate-value">{{ row.interest_rate.toFixed(2) }}% <span class="unit">p.a.</span></p>
            </div>
          </section>

          <div class="content-grid">
            <div class="main-col">
              <section class="panel">
                <h2>Key Information</h2>
                <div class="info-grid">
                  <div class="info-item">
                    <span class="info-label">Tenure</span>
                    <strong>{{ fmtTenure(row.tenure_value ?? 0) }}</strong>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Category</span>
                    <span class="badge" :class="`badge-${row.category_code.toLowerCase()}`">{{ formatCategoryLabel(row.category_code) }}</span>
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
                  OpenFinance LK doesn't yet capture minimum/maximum deposit, interest payment frequency, early-withdrawal terms, or tax treatment for this product.
                  <a v-if="row.source_url" :href="row.source_url" target="_blank" rel="noopener">See {{ bank?.displayName ?? row.bank_name }}'s official page</a> for full terms before making a decision.
                </p>
              </section>

              <section v-if="similar.length" class="panel">
                <h2>Compare Similar Fixed Deposits</h2>
                <p class="panel-sub">Other tenures of the same {{ formatCategoryLabel(row.category_code) }} product at {{ row.bank_name }}.</p>
                <div class="similar-list">
                  <NuxtLink v-for="r in similar" :key="r.id" class="similar-item" :to="similarHref(r)">
                    <span>{{ fmtTenure(r.tenure_value ?? 0) }}</span>
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
                <p class="history-note">A trend chart will build up here as future scrapes are recorded — every scrape is stored, not overwritten.</p>
                <NuxtLink class="history-link" :to="`/products/fixed-deposits/${slug}/history`">View Full Timeline &rarr;</NuxtLink>
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
  margin: 0 0 0.3rem;
  font-size: 1.4rem;
  font-weight: 800;
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
}
.similar-item:hover {
  border-color: var(--accent);
}
.similar-item strong {
  color: var(--accent);
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
