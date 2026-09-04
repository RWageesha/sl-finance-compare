<script setup lang="ts">
import { findDirectoryBank, computeBankStats, computeProductGroups } from '~/utils/bankDirectory'
import type { TaggedRow } from '~/utils/bankDirectory'

const route = useRoute()
const slug = String(route.params.slug)
const bank = findDirectoryBank(slug)

if (!bank) {
  throw createError({ statusCode: 404, statusMessage: 'Bank not found', fatal: true })
}

useHead({ title: `${bank.displayName} — FindRate LK` })

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()

const allRows = ref<TaggedRow[]>([])
const loading = ref(true)

onMounted(async () => {
  if (!bank.tracked) {
    loading.value = false
    return
  }
  try {
    const [fd, savings, loans] = await Promise.all([
      fetchFixedDeposits().catch(() => []),
      fetchSavings().catch(() => []),
      fetchLoans().catch(() => [])
    ])
    allRows.value = [
      ...fd.map((r) => ({ ...r, kind: 'fd' as const })),
      ...savings.map((r) => ({ ...r, kind: 'savings' as const })),
      ...loans.map((r) => ({ ...r, kind: 'loans' as const }))
    ]
  } finally {
    loading.value = false
  }
})

const stats = computed(() => computeBankStats(allRows.value, bank.apiName))
const productGroups = computed(() => computeProductGroups(allRows.value, bank.apiName))

const updatedLabel = computed(() => {
  if (!bank.tracked) return 'Not yet integrated'
  if (loading.value) return 'Loading…'
  return stats.value.lastUpdated ? fmtRelativeDate(stats.value.lastUpdated) : 'No data yet'
})

function fmtPct(n: number): string {
  return `${n.toFixed(2)}%`
}
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <nav class="breadcrumb" aria-label="Breadcrumb">
          <NuxtLink to="/">Home</NuxtLink>
          <span>/</span>
          <NuxtLink to="/banks">Banks</NuxtLink>
          <span>/</span>
          <span>{{ bank.slug.toUpperCase() }}</span>
        </nav>

        <section class="profile-header">
          <div class="profile-main">
            <div class="title-row">
              <h1>{{ bank.displayName }}</h1>
              <span class="type-pill">{{ bank.type }}</span>
            </div>
            <p class="profile-desc">
              <template v-if="bank.tracked">
                Rates below are scraped directly from {{ bank.displayName }}'s own published rates page and refreshed automatically — every figure links back to its source so you can verify it yourself.
              </template>
              <template v-else>
                FindRate doesn't have a live data feed for {{ bank.displayName }} yet. It's listed here to show the full shape of the market — {{ bank.type.toLowerCase() }} coverage may be added later.
              </template>
            </p>
            <a v-if="bank.sourceUrl" class="official-link" :href="bank.sourceUrl" target="_blank" rel="noopener">
              Visit official rates page &rarr;
            </a>
          </div>
          <div v-if="bank.tracked" class="source-box">
            <p class="source-eyebrow">Data Source</p>
            <p class="source-main">{{ stats.sourceCount }} verified source{{ stats.sourceCount === 1 ? '' : 's' }}</p>
            <p class="source-sub">Every rate links back to its exact page</p>
          </div>
        </section>

        <div class="content-grid">
          <div class="products-col">
            <h2 class="section-title">Offered Products &amp; Interest Rates</h2>
            <p class="section-desc">
              <template v-if="bank.tracked">Grouped from current live disclosures — ranges reflect every sub-product we track in that category.</template>
              <template v-else>No products tracked for this bank yet.</template>
            </p>
            <div class="product-grid">
              <div v-for="g in productGroups" :key="g.label" class="product-card" :class="{ empty: g.count === 0 }">
                <p class="product-label">{{ g.label }}</p>
                <p v-if="g.count > 0" class="product-rate">
                  {{ g.minRate === g.maxRate ? fmtPct(g.minRate!) : `${fmtPct(g.minRate!)} – ${fmtPct(g.maxRate!)}` }}
                  <span class="unit">p.a.</span>
                </p>
                <p v-else class="product-rate muted">Not currently listed</p>
                <div class="product-footer">
                  <span class="muted">{{ g.count }} product{{ g.count === 1 ? '' : 's' }}</span>
                  <NuxtLink v-if="g.count > 0" class="view-link" :to="g.href">View Details &rsaquo;</NuxtLink>
                </div>
              </div>
            </div>
          </div>

          <aside class="sidebar-col">
            <div class="side-card">
              <p class="side-eyebrow">Coverage</p>
              <p class="side-big" :class="{ dim: stats.categoriesTracked === 0 }">{{ stats.categoriesTracked }}/3</p>
              <p class="side-sub">categories tracked{{ stats.productCount ? ` · ${stats.productCount} products` : '' }}</p>
              <div class="side-rows">
                <div class="side-row">
                  <span>Last updated</span>
                  <strong>{{ updatedLabel }}</strong>
                </div>
                <div class="side-row">
                  <span>Verified sources</span>
                  <strong>{{ bank.tracked ? stats.sourceCount : '—' }}</strong>
                </div>
              </div>
            </div>

            <div class="side-card">
              <p class="side-eyebrow">Rate History</p>
              <p class="side-note">
                Every scrape is stored rather than overwritten, so historical rate-change tracking is technically possible — that view just isn't built yet.
              </p>
            </div>
          </aside>
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

/* Breadcrumb */
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

/* Profile header */
.profile-header {
  display: flex;
  gap: 1.5rem;
  align-items: flex-start;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.4rem 1.6rem;
}
.profile-main {
  flex: 1;
  min-width: 0;
}
.title-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}
.title-row h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 800;
}
.type-pill {
  display: inline-block;
  padding: 0.25rem 0.65rem;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 700;
  background: #e6f0ff;
  color: #1a4fb4;
}
[data-theme='dark'] .type-pill {
  background: #17335f;
  color: #a8c6ff;
}
.profile-desc {
  margin: 0.6rem 0 0.8rem;
  max-width: 620px;
  color: var(--muted);
  font-size: 0.88rem;
  line-height: 1.55;
}
.official-link {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
}
.official-link:hover {
  text-decoration: underline;
}
.source-box {
  flex: none;
  width: 220px;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem;
  background: var(--bg);
}
.source-eyebrow {
  margin: 0 0 0.3rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}
.source-main {
  margin: 0 0 0.2rem;
  font-size: 0.95rem;
  font-weight: 700;
}
.source-sub {
  margin: 0;
  font-size: 0.75rem;
  color: var(--muted);
}
@media (max-width: 720px) {
  .profile-header {
    flex-direction: column;
  }
  .source-box {
    width: 100%;
  }
}

/* Content grid */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 300px;
  gap: 1.5rem;
  margin: 1.6rem 0 2.5rem;
  align-items: start;
}
@media (max-width: 900px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}
.section-title {
  margin: 0 0 0.3rem;
  font-size: 1.1rem;
  font-weight: 700;
}
.section-desc {
  margin: 0 0 1rem;
  color: var(--muted);
  font-size: 0.85rem;
}

/* Product grid */
.product-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}
@media (max-width: 600px) {
  .product-grid {
    grid-template-columns: 1fr;
  }
}
.product-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem 1.1rem;
}
.product-card.empty {
  opacity: 0.65;
}
.product-label {
  margin: 0 0 0.4rem;
  font-size: 0.82rem;
  color: var(--muted);
}
.product-rate {
  margin: 0 0 0.6rem;
  font-size: 1.25rem;
  font-weight: 800;
}
.product-rate .unit {
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--muted);
  margin-left: 0.25rem;
}
.product-rate.muted {
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--muted);
}
.product-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 0.78rem;
  padding-top: 0.6rem;
  border-top: 1px solid var(--border);
}
.muted {
  color: var(--muted);
}
.view-link {
  color: var(--accent);
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
}
.view-link:hover {
  text-decoration: underline;
}

/* Sidebar */
.sidebar-col {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.side-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.1rem 1.2rem;
}
.side-eyebrow {
  margin: 0 0 0.4rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}
.side-big {
  margin: 0 0 0.2rem;
  font-size: 2rem;
  font-weight: 800;
  color: var(--accent);
}
.side-big.dim {
  color: var(--muted);
}
.side-sub {
  margin: 0 0 0.9rem;
  font-size: 0.78rem;
  color: var(--muted);
}
.side-rows {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.7rem;
  border-top: 1px solid var(--border);
}
.side-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.82rem;
}
.side-row span {
  color: var(--muted);
}
.side-row strong {
  font-weight: 700;
}
.side-note {
  margin: 0;
  font-size: 0.82rem;
  color: var(--muted);
  line-height: 1.55;
}
</style>
