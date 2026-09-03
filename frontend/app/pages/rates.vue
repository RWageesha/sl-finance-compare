<script setup lang="ts">
import type { ProductRate } from '~/composables/useRatesApi'
import type { ColumnDef } from '~/components/RatesTable.vue'

interface TabDef {
  id: string
  label: string
  emptyMessage: string
  filterKey: keyof ProductRate
  filterLabel: string
  defaultSortKey: keyof ProductRate
  columns: ColumnDef[]
  load: () => Promise<ProductRate[]>
}

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()

const TABS: TabDef[] = [
  {
    id: 'fd',
    label: 'Fixed Deposits',
    emptyMessage: 'No fixed deposit rates in the database yet.',
    filterKey: 'category_code',
    filterLabel: 'Rate type',
    defaultSortKey: 'interest_rate',
    load: fetchFixedDeposits,
    columns: [
      { key: 'bank_name', label: 'Bank', type: 'text' },
      { key: 'tenure_value', label: 'Tenure', type: 'tenure' },
      { key: 'category_code', label: 'Type', type: 'badge' },
      { key: 'interest_rate', label: 'Rate', type: 'rate' },
      { key: 'scraped_at', label: 'Last updated', type: 'date' }
    ]
  },
  {
    id: 'savings',
    label: 'Savings',
    emptyMessage: 'No savings rates in the database yet.',
    filterKey: 'category_code',
    filterLabel: 'Category',
    defaultSortKey: 'interest_rate',
    load: fetchSavings,
    columns: [
      { key: 'bank_name', label: 'Bank', type: 'text' },
      { key: 'category_code', label: 'Category', type: 'badge' },
      { key: 'product_name', label: 'Account', type: 'text' },
      { key: 'tenure_label', label: 'Balance tier', type: 'text' },
      { key: 'interest_rate', label: 'Rate', type: 'rate' },
      { key: 'scraped_at', label: 'Last updated', type: 'date' }
    ]
  },
  {
    id: 'loans',
    label: 'Loans',
    emptyMessage: 'No loan rates in the database yet.',
    filterKey: 'category_code',
    filterLabel: 'Category',
    defaultSortKey: 'interest_rate',
    load: fetchLoans,
    columns: [
      { key: 'bank_name', label: 'Bank', type: 'text' },
      { key: 'category_code', label: 'Category', type: 'badge' },
      { key: 'product_name', label: 'Product', type: 'text' },
      { key: 'tenure_label', label: 'Tenure', type: 'text' },
      { key: 'rate_label', label: 'Rate label', type: 'text' },
      { key: 'interest_rate', label: 'Rate', type: 'rate' },
      { key: 'scraped_at', label: 'Last updated', type: 'date' }
    ]
  }
]

const route = useRoute()
const router = useRouter()

const requestedTab = typeof route.query.tab === 'string' ? route.query.tab : null
const initialCategory = typeof route.query.category === 'string' ? route.query.category : undefined
const initialBank = typeof route.query.bank === 'string' ? route.query.bank : undefined

const activeTabId = ref(TABS.find((t) => t.id === requestedTab)?.id ?? TABS[0].id)
const activeTab = computed(() => TABS.find((t) => t.id === activeTabId.value)!)

const rowsByTab = reactive<Record<string, ProductRate[]>>({})
const loadingByTab = reactive<Record<string, boolean>>({})
const errorByTab = reactive<Record<string, string | null>>({})

async function loadTab(tab: TabDef) {
  if (rowsByTab[tab.id] || loadingByTab[tab.id]) return
  loadingByTab[tab.id] = true
  errorByTab[tab.id] = null
  try {
    rowsByTab[tab.id] = await tab.load()
  } catch (err: any) {
    errorByTab[tab.id] = err?.message || 'Failed to load data'
    rowsByTab[tab.id] = []
  } finally {
    loadingByTab[tab.id] = false
  }
}

function selectTab(tab: TabDef) {
  activeTabId.value = tab.id
  router.replace({ query: { tab: tab.id } })
  loadTab(tab)
}

onMounted(() => {
  loadTab(activeTab.value)
})

useHead({ title: 'Compare Rates — OpenFinance LK' })
</script>

<template>
  <div>
    <header class="page-header">
      <div class="wrap">
        <div class="header-top">
          <NuxtLink class="back" to="/">&larr; Back to home</NuxtLink>
          <ThemeToggle />
        </div>
        <h1>OpenFinance LK</h1>
        <p>Fixed Deposit, Savings, and Loan rates across Sri Lankan banks</p>
      </div>
    </header>

    <main class="wrap">
      <div class="tabs">
        <button
          v-for="tab in TABS"
          :key="tab.id"
          type="button"
          class="tab-btn"
          :class="{ active: tab.id === activeTabId }"
          @click="selectTab(tab)"
        >
          {{ tab.label }}
        </button>
      </div>

      <RatesTable
        :tab-id="activeTabId"
        :rows="rowsByTab[activeTabId] || []"
        :columns="activeTab.columns"
        :filter-key="activeTab.filterKey"
        :filter-label="activeTab.filterLabel"
        :default-sort-key="activeTab.defaultSortKey"
        :loading="!!loadingByTab[activeTabId]"
        :empty-message="activeTab.emptyMessage"
        :initial-category="activeTabId === requestedTab ? initialCategory : undefined"
        :initial-bank="initialBank"
      />
    </main>

    <footer class="page-footer wrap">
      Rates are as published by each bank at last scrape time and may not reflect current promotional offers.
    </footer>
  </div>
</template>

<style scoped>
.wrap {
  max-width: 960px;
  margin: 0 auto;
  padding: 1.5rem;
}
.page-header {
  background: var(--navy-dark);
  color: #f2f2f4;
}
.page-header .wrap {
  padding: 1.5rem 1.5rem 1.25rem;
}
.back {
  display: inline-block;
  color: #b6b7bf;
  font-size: 0.85rem;
  text-decoration: none;
  margin-bottom: 0.5rem;
}
.back:hover {
  color: #f2f2f4;
}
.header-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}
.page-header h1 {
  margin: 0 0 0.25rem;
  font-size: 1.4rem;
  font-weight: 600;
}
.page-header p {
  margin: 0;
  color: #b6b7bf;
  font-size: 0.9rem;
}
.tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}
.tab-btn {
  font: inherit;
  font-weight: 600;
  font-size: 0.85rem;
  padding: 0.5rem 0.9rem;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--muted);
  cursor: pointer;
}
.tab-btn:hover {
  color: var(--text);
}
.tab-btn.active {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
.page-footer {
  font-size: 0.78rem;
  color: var(--muted);
  margin-top: 1rem;
  margin-bottom: 2rem;
}
</style>
