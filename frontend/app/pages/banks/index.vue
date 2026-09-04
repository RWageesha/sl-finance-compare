<script setup lang="ts">
import { BANK_TYPES, DIRECTORY_BANKS, computeBankStats } from '~/utils/bankDirectory'
import type { DirectoryBank, TaggedRow } from '~/utils/bankDirectory'

useHead({ title: 'Bank Directory — FindRate LK' })

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

function statsFor(bank: DirectoryBank) {
  return computeBankStats(allRows.value, bank.apiName)
}

function updatedLabel(bank: DirectoryBank): string {
  if (!bank.tracked) return 'Not yet integrated'
  if (loading.value) return 'Loading…'
  const last = statsFor(bank).lastUpdated
  return last ? `Updated ${fmtRelativeDate(last)}` : 'No data yet'
}

const searchQuery = ref('')
const typeFilter = ref('')
const productFilter = ref<'' | 'fd' | 'savings' | 'loans'>('')

const filteredBanks = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  return DIRECTORY_BANKS.filter((bank) => {
    if (q && !bank.displayName.toLowerCase().includes(q)) return false
    if (typeFilter.value && bank.type !== typeFilter.value) return false
    if (productFilter.value) {
      // A bank we don't track has zero rows in every category, so it
      // always drops out of a product filter — correct, not a bug.
      if (!bank.tracked || !bank.apiName) return false
      const has = allRows.value.some((r) => r.bank_name === bank.apiName && r.kind === productFilter.value)
      if (!has) return false
    }
    return true
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
          <span>Banks</span>
        </nav>

        <section class="hero-banner">
          <div class="hero-copy">
            <span class="pill pill-accent"><Icon name="bank" /> Sri Lanka</span>
            <h1>All Banks</h1>
            <p>Explore and search financial institutions currently integrated with FindRate LK data feeds.</p>
            <div class="hero-badges">
              <span class="pill"><Icon name="shield" /> Public disclosures</span>
              <span class="pill"><Icon name="clock" /> Updated daily</span>
            </div>
          </div>
          <div class="hero-illustration" aria-hidden="true">
            <div class="hero-illustration-icon"><Icon name="grid" /></div>
            <div class="hero-illustration-icon alt"><Icon name="refresh" /></div>
          </div>
        </section>

        <div class="filters">
          <div class="filter-input">
            <Icon name="search" />
            <input v-model="searchQuery" type="text" placeholder="Search banks...">
          </div>
          <div class="filter-select">
            <Icon name="bank" />
            <select v-model="typeFilter">
              <option value="">Bank Type: All</option>
              <option v-for="t in BANK_TYPES" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
          <div class="filter-select">
            <Icon name="coin" />
            <select v-model="productFilter">
              <option value="">Product: Any</option>
              <option value="fd">Fixed Deposits</option>
              <option value="savings">Savings</option>
              <option value="loans">Loans</option>
            </select>
          </div>
        </div>

        <p v-if="filteredBanks.length === 0" class="no-results">No banks match your filters.</p>
        <div v-else class="bank-grid">
          <div v-for="bank in filteredBanks" :key="bank.slug" class="bank-card" :class="{ untracked: !bank.tracked }">
            <div class="bank-icon"><Icon :name="bank.icon" /></div>
            <h3>{{ bank.displayName }}</h3>
            <p class="bank-type">{{ bank.type }}</p>

            <div class="stat-row">
              <span>Available Products</span>
              <strong>{{ bank.tracked ? `${statsFor(bank).productCount} products` : '—' }}</strong>
            </div>
            <div class="stat-row">
              <span>Categories tracked</span>
              <strong class="dot-stat">
                <span class="dot" :class="{ green: statsFor(bank).categoriesTracked === 3 }" />
                {{ statsFor(bank).categoriesTracked }}/3
              </strong>
            </div>

            <div class="card-footer">
              <span class="updated">{{ updatedLabel(bank) }}</span>
              <NuxtLink class="view-btn" :to="`/banks/${bank.slug}`">View Directory</NuxtLink>
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
  padding: 1.8rem;
  display: flex;
  gap: 2rem;
  align-items: center;
}
[data-theme='dark'] .hero-banner {
  background: linear-gradient(135deg, #132043, #0d1730);
}
.hero-copy {
  flex: 1 1 55%;
  min-width: 0;
}
.hero-copy h1 {
  margin: 0.6rem 0 0.4rem;
  font-size: 1.9rem;
  font-weight: 800;
}
.hero-copy > p {
  margin: 0;
  max-width: 440px;
  color: var(--muted);
  font-size: 0.92rem;
}
.hero-badges {
  display: flex;
  gap: 0.6rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}
.pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.3rem 0.7rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  background: var(--panel);
  color: var(--text);
  border: 1px solid var(--border);
}
.pill svg {
  width: 12px;
  height: 12px;
}
.pill-accent {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}
.hero-illustration {
  flex: 1 1 40%;
  min-height: 180px;
  border-radius: 10px;
  background: linear-gradient(135deg, #bcd4ff, #93b8ff);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  position: relative;
}
[data-theme='dark'] .hero-illustration {
  background: linear-gradient(135deg, #1c2f66, #13214d);
}
.hero-illustration-icon {
  width: 76px;
  height: 76px;
  border-radius: 14px;
  background: var(--panel);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
}
.hero-illustration-icon svg {
  width: 34px;
  height: 34px;
}
.hero-illustration-icon.alt {
  width: 52px;
  height: 52px;
  position: absolute;
  bottom: 18px;
  right: 24px;
  color: #00707a;
}
.hero-illustration-icon.alt svg {
  width: 24px;
  height: 24px;
}
@media (max-width: 720px) {
  .hero-banner {
    flex-direction: column;
    align-items: stretch;
  }
}

/* Filters */
.filters {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin: 1.5rem 0;
}
.filter-input,
.filter-select {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--panel);
  padding: 0 0.8rem;
}
.filter-input {
  flex: 1 1 260px;
}
.filter-select {
  flex: 0 0 auto;
  min-width: 170px;
}
.filter-input svg,
.filter-select svg {
  width: 15px;
  height: 15px;
  color: var(--muted);
  flex: none;
}
.filter-input input,
.filter-select select {
  border: none;
  outline: none;
  background: transparent;
  font: inherit;
  font-size: 0.85rem;
  color: var(--text);
  padding: 0.65rem 0;
  flex: 1;
  min-width: 0;
}

/* Bank grid */
.no-results {
  padding: 2.5rem;
  text-align: center;
  color: var(--muted);
}
.bank-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin: 0 0 2.5rem;
}
@media (max-width: 900px) {
  .bank-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (max-width: 600px) {
  .bank-grid {
    grid-template-columns: 1fr;
  }
}
.bank-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-top: 3px solid var(--accent);
  border-radius: 10px;
  padding: 1.1rem 1.2rem;
}
.bank-card.untracked {
  border-top-color: var(--border);
  opacity: 0.65;
}
.bank-icon {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  background: #e6f0ff;
  color: #1a4fb4;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 0.7rem;
}
[data-theme='dark'] .bank-icon {
  background: #17335f;
  color: #a8c6ff;
}
.bank-icon svg {
  width: 18px;
  height: 18px;
}
.bank-card h3 {
  margin: 0 0 0.1rem;
  font-size: 0.95rem;
  font-weight: 700;
}
.bank-type {
  margin: 0 0 0.9rem;
  font-size: 0.78rem;
  color: var(--muted);
}
.stat-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.82rem;
  margin-bottom: 0.4rem;
}
.stat-row span {
  color: var(--muted);
}
.stat-row strong {
  font-weight: 700;
}
.dot-stat {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--muted);
  flex: none;
}
.dot.green {
  background: #1f9d55;
}
[data-theme='dark'] .dot.green {
  background: #4ade80;
}
.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-top: 0.8rem;
  padding-top: 0.7rem;
  border-top: 1px solid var(--border);
}
.updated {
  font-size: 0.72rem;
  color: var(--muted);
  white-space: nowrap;
}
.view-btn {
  padding: 0.4rem 0.85rem;
  border-radius: 6px;
  background: #e6f0ff;
  color: #1a4fb4;
  font-size: 0.78rem;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
}
[data-theme='dark'] .view-btn {
  background: #17335f;
  color: #a8c6ff;
}
.view-btn:hover {
  filter: brightness(0.96);
}
</style>
