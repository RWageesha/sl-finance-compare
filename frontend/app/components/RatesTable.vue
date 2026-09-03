<script setup lang="ts">
import type { ProductRate } from '~/composables/useRatesApi'
import { productDetailHref } from '~/utils/fdCompare'

export interface ColumnDef {
  key: keyof ProductRate
  label: string
  type: 'text' | 'tenure' | 'rate' | 'date' | 'badge'
}

const props = defineProps<{
  /** Identifies which tab this data belongs to (e.g. "fd"/"savings"/
   * "loans") — used only to detect a tab switch and reset filter/sort
   * state. filterKey is the same ("category_code") for every tab, so it
   * can't be used for that; rows/columns are new array/object identities
   * on every parent re-render even for the same tab, so they can't either. */
  tabId: string
  rows: ProductRate[]
  columns: ColumnDef[]
  filterKey: keyof ProductRate
  filterLabel: string
  defaultSortKey: keyof ProductRate
  loading: boolean
  emptyMessage: string
  /** Pre-selects the secondary filter dropdown on first load, e.g. from a
   * ?category= deep link. Only applied once the value actually appears
   * among the loaded rows. */
  initialCategory?: string
  /** Same idea as initialCategory but for the Bank dropdown, e.g. from the
   * Bank Directory page's "View Directory" link. Bank identity is the same
   * across every tab (unlike category codes, which are tab-specific), so
   * this applies on whichever tab the rows arrive for. */
  initialBank?: string
}>()

const bankFilter = ref('')
const typeFilter = ref('')
const sortKey = ref<keyof ProductRate>(props.defaultSortKey)
const sortDir = ref<1 | -1>(-1)

const banks = computed(() => uniqueSorted(props.rows.map((r) => r.bank_name)))
const types = computed(() => uniqueSorted(props.rows.map((r) => String(r[props.filterKey] ?? ''))))

function uniqueSorted(values: string[]): string[] {
  return [...new Set(values.filter(Boolean))].sort()
}

// Single watcher covering both "switched to a different tab" (reset
// filters/sort) and "rows for the initially-requested tab just arrived"
// (apply the ?category= deep link, once). Combined into one watcher
// rather than two separate ones so there's no ordering dependency between
// a reset and an apply that both react to the same underlying change.
let lastTabId: string | null = null
let appliedInitialCategory = false
let appliedInitialBank = false

watch(
  [() => props.tabId, () => props.rows],
  ([tabId, rows]) => {
    if (tabId !== lastTabId) {
      bankFilter.value = ''
      typeFilter.value = ''
      sortKey.value = props.defaultSortKey
      sortDir.value = -1
      lastTabId = tabId
      appliedInitialCategory = false
      appliedInitialBank = false
    }

    if (!appliedInitialCategory && props.initialCategory && rows.length > 0) {
      if (types.value.includes(props.initialCategory)) {
        typeFilter.value = props.initialCategory
      }
      appliedInitialCategory = true
    }

    if (!appliedInitialBank && props.initialBank && rows.length > 0) {
      if (banks.value.includes(props.initialBank)) {
        bankFilter.value = props.initialBank
      }
      appliedInitialBank = true
    }
  },
  { immediate: true }
)

const filteredRows = computed(() => {
  let rows = props.rows.filter((r) => {
    return (
      (!bankFilter.value || r.bank_name === bankFilter.value) &&
      (!typeFilter.value || String(r[props.filterKey] ?? '') === typeFilter.value)
    )
  })

  rows = [...rows].sort((a, b) => {
    let av: any = a[sortKey.value]
    let bv: any = b[sortKey.value]
    if (typeof av === 'string') av = av.toLowerCase()
    if (typeof bv === 'string') bv = bv.toLowerCase()
    if (av === undefined || av === null) av = ''
    if (bv === undefined || bv === null) bv = ''
    if (av < bv) return -1 * sortDir.value
    if (av > bv) return 1 * sortDir.value
    return 0
  })

  return rows
})

function toggleSort(key: keyof ProductRate) {
  if (sortKey.value === key) {
    sortDir.value = (sortDir.value * -1) as 1 | -1
  } else {
    sortKey.value = key
    sortDir.value = key === 'interest_rate' ? -1 : 1
  }
}

function cellText(col: ColumnDef, row: ProductRate): string {
  const value = row[col.key]
  if (col.type === 'tenure') return fmtTenure(Number(value ?? 0))
  if (col.type === 'rate') return `${Number(value ?? 0).toFixed(2)}%`
  if (col.type === 'date') return fmtDate(String(value ?? ''))
  return value == null ? '' : String(value)
}
</script>

<template>
  <div>
    <div class="toolbar">
      <label>
        Bank
        <select v-model="bankFilter">
          <option value="">All banks</option>
          <option v-for="b in banks" :key="b" :value="b">{{ b }}</option>
        </select>
      </label>
      <label>
        <span>{{ filterLabel }}</span>
        <select v-model="typeFilter">
          <option value="">All</option>
          <option v-for="t in types" :key="t" :value="t">{{ formatCategoryLabel(t) }}</option>
        </select>
      </label>
      <span class="status">
        <template v-if="loading">Loading…</template>
        <template v-else>{{ filteredRows.length }} rate{{ filteredRows.length === 1 ? '' : 's' }} loaded</template>
      </span>
    </div>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th
              v-for="col in columns"
              :key="col.key"
              :class="{ sorted: sortKey === col.key }"
              @click="toggleSort(col.key)"
            >
              {{ col.label }}
              <span v-if="sortKey === col.key">{{ sortDir === 1 ? '▲' : '▼' }}</span>
            </th>
            <th class="details-col"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td :colspan="columns.length + 1" class="empty">Loading data…</td>
          </tr>
          <tr v-else-if="rows.length === 0">
            <td :colspan="columns.length + 1" class="empty">{{ emptyMessage }}</td>
          </tr>
          <tr v-else-if="filteredRows.length === 0">
            <td :colspan="columns.length + 1" class="empty">No rates match the current filters.</td>
          </tr>
          <tr v-for="row in filteredRows" :key="row.id">
            <td v-for="col in columns" :key="col.key" :class="{ rate: col.type === 'rate', tenure: col.type === 'tenure' }">
              <span v-if="col.type === 'badge'" class="badge" :class="`badge-${String(row[col.key]).toLowerCase()}`">
                {{ formatCategoryLabel(String(row[col.key])) }}
              </span>
              <template v-else>{{ cellText(col, row) }}</template>
            </td>
            <td class="details-col">
              <NuxtLink v-if="productDetailHref(row)" class="details-link" :to="productDetailHref(row)!">View Details</NuxtLink>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 1rem;
}
.toolbar label {
  font-size: 0.85rem;
  color: var(--muted);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
select {
  font: inherit;
  padding: 0.4rem 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--text);
}
.status {
  margin-left: auto;
  font-size: 0.8rem;
  color: var(--muted);
}
.table-wrap {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.92rem;
  min-width: 640px;
}
th,
td {
  padding: 0.6rem 0.8rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
th {
  cursor: pointer;
  user-select: none;
  color: var(--muted);
  font-weight: 600;
  font-size: 0.78rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
th:hover {
  color: var(--text);
}
th.sorted {
  color: var(--text);
}
tbody tr:nth-child(even) {
  background: var(--row-alt, rgba(127, 127, 127, 0.04));
}
td.rate {
  font-weight: 600;
  text-align: right;
}
td.tenure {
  text-align: right;
}
.empty {
  padding: 2rem;
  text-align: center;
  color: var(--muted);
  white-space: normal;
}
.details-col {
  text-align: right;
}
.details-link {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
  white-space: nowrap;
}
.details-link:hover {
  text-decoration: underline;
}
</style>
