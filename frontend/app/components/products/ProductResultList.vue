<script setup lang="ts">
import type { ColumnDef, ProductRow } from '~/config/productTypes'
import { bankColor, bankInitial } from '~/utils/bankColors'

const props = defineProps<{
  columns: ColumnDef[]
  rows: ProductRow[]
}>()

// --- Mobile card layout: derived generically from the same column set ---
// the desktop table uses, so a new product type never needs its own card
// template — see the config file for how each type's columns differ.
const bankCol = computed(() => props.columns.find((c) => c.type === 'bank'))
const productCol = computed(() => props.columns.find((c) => c.key === 'product'))
const verifiedCol = computed(() => props.columns.find((c) => c.type === 'verification'))
const actionCol = computed(() => props.columns.find((c) => c.type === 'action'))
// The card's bold top-right headline: the rate/rate-range column if this
// product type has one (FD, loans, cards), else the first remaining text
// column (e.g. Annual Fee for debit cards, which carry no rate at all).
const headlineCol = computed(() => {
  const rate = props.columns.find((c) => c.type === 'rate' || c.type === 'rateRange')
  if (rate) return rate
  return props.columns.find((c) => c.type === 'text' && c.key !== productCol.value?.key)
})
const restCols = computed(() =>
  props.columns.filter(
    (c) => ![bankCol.value?.key, productCol.value?.key, verifiedCol.value?.key, actionCol.value?.key, headlineCol.value?.key].includes(c.key)
  )
)
const restPairs = computed(() => {
  const pairs: ColumnDef[][] = []
  const cols = restCols.value
  for (let i = 0; i < cols.length; i += 2) pairs.push(cols.slice(i, i + 2))
  return pairs
})

const PAGE_SIZE = 4
const currentPage = ref(1)
const pageCount = computed(() => Math.max(1, Math.ceil(props.rows.length / PAGE_SIZE)))
const pagedRows = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return props.rows.slice(start, start + PAGE_SIZE)
})
watch(
  () => props.rows,
  () => { currentPage.value = 1 }
)
</script>

<template>
  <div>
    <p v-if="rows.length === 0" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
      No products match your filters.
    </p>

    <!-- Desktop / tablet table -->
    <div v-else class="hidden overflow-x-auto rounded-card border border-card-border bg-card shadow-sm md:block">
      <table class="w-full min-w-[720px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] font-bold uppercase tracking-wide text-muted">
            <th v-for="col in columns" :key="col.key" class="px-4 py-3" :class="col.align === 'right' ? 'text-right' : 'text-left'">
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in pagedRows" :key="row.id" class="border-b border-card-border last:border-none">
            <td v-for="col in columns" :key="col.key" class="px-4 py-3.5" :class="col.align === 'right' ? 'text-right' : 'text-left'">
              <span v-if="col.type === 'bank'" class="flex items-center gap-2.5">
                <span
                  class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
                  :style="{ backgroundColor: bankColor(row.bank) }"
                >{{ bankInitial(row.bank) }}</span>
                <span class="font-semibold text-navy">{{ row.bank }}</span>
              </span>

              <span v-else-if="col.type === 'rate'" class="text-[15px] font-bold text-primary">{{ row[col.key] }}</span>

              <span v-else-if="col.type === 'rateRange'">
                <span v-if="row.tiered" class="inline-flex items-center gap-1.5">
                  <span class="text-[15px] font-bold text-primary">{{ row[col.key] }}</span>
                  <span class="rounded-pill bg-badge-bg px-2 py-0.5 text-[10px] font-bold text-primary">Tiered</span>
                </span>
                <span v-else class="text-[15px] font-bold text-navy">{{ row[col.key] }}</span>
              </span>

              <span v-else-if="col.type === 'verification'" class="inline-flex items-center gap-1.5 text-muted">
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />{{ row[col.key] }}
              </span>

              <NuxtLink v-else-if="col.type === 'action'" :to="`/products/sample/${row.id}`" class="font-bold text-primary hover:underline">
                Details &rarr;
              </NuxtLink>

              <span v-else class="text-muted">{{ row[col.key] }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile cards -->
    <div class="flex flex-col gap-3 md:hidden">
      <div v-for="row in pagedRows" :key="row.id" class="rounded-xl border border-card-border bg-card p-3.5 shadow-sm">
        <div class="flex items-center justify-between gap-3">
          <span class="flex min-w-0 items-center gap-2.5">
            <span
              v-if="bankCol"
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white"
              :style="{ backgroundColor: bankColor(row.bank) }"
            >{{ bankInitial(row.bank) }}</span>
            <span class="truncate font-semibold text-navy">{{ row.bank }}</span>
          </span>
          <span v-if="headlineCol" class="shrink-0 text-right">
            <span
              class="text-[15px] font-bold"
              :class="headlineCol.type === 'rateRange' && !row.tiered ? 'text-navy' : 'text-primary'"
            >{{ row[headlineCol.key] }}</span>
            <span v-if="headlineCol.type === 'rateRange' && row.tiered" class="ml-1 rounded-pill bg-badge-bg px-1.5 py-0.5 text-[9px] font-bold text-primary">
              Tiered
            </span>
          </span>
        </div>

        <p v-if="productCol" class="mt-1.5 text-[13px] text-muted">{{ row[productCol.key] }}</p>

        <div v-for="(pair, pi) in restPairs" :key="pi" class="mt-1.5 flex items-center justify-between text-[12.5px] text-muted">
          <span v-for="col in pair" :key="col.key">{{ col.label }}: {{ row[col.key] }}</span>
        </div>

        <div class="mt-2.5 flex items-center justify-between border-t border-card-border pt-2.5">
          <span v-if="verifiedCol" class="inline-flex items-center gap-1.5 text-xs text-muted">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />{{ row[verifiedCol.key] }}
          </span>
          <span v-else />
          <NuxtLink v-if="actionCol" :to="`/products/sample/${row.id}`" class="text-[13px] font-bold text-primary">Details &rarr;</NuxtLink>
        </div>
      </div>
    </div>

    <Pagination v-if="pageCount > 1" v-model="currentPage" :page-count="pageCount" />
  </div>
</template>
