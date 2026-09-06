<script setup lang="ts">
import { PRODUCT_DIRECTORY, productHref } from '~/utils/productDirectory'
import type { ProductEntry } from '~/utils/productDirectory'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import type { TaggedRow } from '~/utils/bankDirectory'

useHead({ title: 'Financial Products — FindRate LK' })

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()
const allRows = ref<TaggedRow[]>([])
const loading = ref(true)

onMounted(async () => {
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

const TRACKED_BANK_COUNT = DIRECTORY_BANKS.filter((b) => b.tracked).length

function statsFor(item: ProductEntry) {
  const rows = allRows.value.filter((r) => r.kind === item.tab && (!item.category || r.category_code === item.category))
  return { banks: new Set(rows.map((r) => r.bank_name)).size, rates: rows.length }
}

// Same keyword router as the homepage hero search — jumps straight to the
// matching filtered rates view rather than a generic results page.
const ROUTES: { test: RegExp; href: string }[] = [
  { test: /housing|home/, href: '/compare/housing-loans' },
  { test: /personal loan/, href: '/compare/personal-loans' },
  { test: /vehicle|lease/, href: '/rates?tab=loans&category=LEASE' },
  { test: /gold|pawn/, href: '/compare/gold-loans' },
  { test: /loan/, href: '/rates?tab=loans' },
  { test: /saving/, href: '/compare/savings-accounts' },
  { test: /fixed deposit|\bfd\b|deposit/, href: '/products/fixed-deposits' }
]
const searchQuery = ref('')
function onSearch() {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) {
    navigateTo('/rates')
    return
  }
  const match = ROUTES.find((r) => r.test.test(q))
  navigateTo(match ? match.href : '/rates')
}
</script>

<template>
  <div>
    <AppHeader />

    <main>
      <div class="wrap">
        <nav class="breadcrumb" aria-label="Breadcrumb">
          <NuxtLink to="/">Home</NuxtLink>
          <span>/</span>
          <span>Products</span>
        </nav>

        <section class="hero-banner">
          <h1>Financial Products</h1>
          <p>Compare rates, terms, and criteria from verified Sri Lankan banks &amp; licensed financial institutions.</p>
          <form class="search-form" @submit.prevent="onSearch">
            <Icon name="search" />
            <input v-model="searchQuery" type="text" placeholder="Search for active products (e.g. fixed deposit, home loan)...">
            <button type="submit">Search</button>
          </form>
        </section>

        <div class="group-grid">
          <div v-for="group in PRODUCT_DIRECTORY" :key="group.label" class="group-card">
            <div class="group-header">
              <h2>{{ group.label }}</h2>
              <span class="group-count">{{ group.items.length }}</span>
            </div>
            <div class="group-items">
              <div v-for="item in group.items" :key="item.label" class="group-item" :class="{ untracked: !item.tracked }">
                <div class="item-row">
                  <span class="item-label">{{ item.label }}</span>
                  <NuxtLink v-if="item.tracked" class="compare-link" :to="productHref(item)">Compare &rarr;</NuxtLink>
                  <span v-else class="compare-link disabled">Not tracked</span>
                </div>
                <p class="item-desc">
                  <template v-if="!item.tracked">Not tracked by FindRate yet.</template>
                  <template v-else-if="loading">Loading…</template>
                  <template v-else>
                    Tracked at {{ statsFor(item).banks }} of {{ TRACKED_BANK_COUNT }} banks — {{ statsFor(item).rates }} live rate{{ statsFor(item).rates === 1 ? '' : 's' }}.
                  </template>
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <AppFooter />
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

/* Hero banner */
.hero-banner {
  background: linear-gradient(135deg, #eaf1ff, #dce7ff);
  border-radius: 14px;
  padding: 1.8rem 2rem;
}
[data-theme='dark'] .hero-banner {
  background: linear-gradient(135deg, #132043, #0d1730);
}
.hero-banner h1 {
  margin: 0 0 0.5rem;
  font-size: 1.9rem;
  font-weight: 800;
}
.hero-banner > p {
  margin: 0 0 1.2rem;
  max-width: 560px;
  color: var(--muted);
  font-size: 0.92rem;
}
.search-form {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  max-width: 560px;
  background: var(--panel);
  padding: 0.35rem 0.35rem 0.35rem 0.9rem;
  border-radius: 10px;
  border: 1px solid var(--border);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}
.search-form svg {
  width: 15px;
  height: 15px;
  color: var(--muted);
  flex: none;
}
.search-form input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  padding: 0.6rem 0;
  font: inherit;
  font-size: 0.88rem;
  color: var(--text);
  min-width: 0;
}
.search-form button {
  font: inherit;
  font-weight: 600;
  font-size: 0.85rem;
  padding: 0.6rem 1.1rem;
  border-radius: 7px;
  border: none;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
  white-space: nowrap;
}

/* Group grid */
.group-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin: 1.5rem 0 2.5rem;
  align-items: start;
}
@media (max-width: 900px) {
  .group-grid {
    grid-template-columns: 1fr;
  }
}
.group-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.2rem 1.3rem;
}
.group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.7rem;
}
.group-header h2 {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
}
.group-count {
  background: var(--bg);
  color: var(--muted);
  font-size: 0.68rem;
  font-weight: 700;
  padding: 0.15rem 0.55rem;
  border-radius: 999px;
}
.group-item {
  padding: 0.85rem 0;
  border-bottom: 1px solid var(--border);
}
.group-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}
.group-item.untracked {
  opacity: 0.65;
}
.item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.item-label {
  font-weight: 700;
  font-size: 0.88rem;
}
.compare-link {
  color: var(--accent);
  font-weight: 600;
  font-size: 0.8rem;
  text-decoration: none;
  white-space: nowrap;
}
.compare-link:hover {
  text-decoration: underline;
}
.compare-link.disabled {
  color: var(--muted);
  cursor: default;
}
.item-desc {
  margin: 0.3rem 0 0;
  font-size: 0.78rem;
  color: var(--muted);
  line-height: 1.45;
}
</style>
