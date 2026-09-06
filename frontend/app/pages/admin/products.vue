<script setup lang="ts">
// Read-only for now: editing the taxonomy's `code` values here would risk
// breaking the scraper's GetOrCreateCategory lookups, which key off those
// exact codes — a structural edit form for this is a separate, more
// careful piece of work than what fits this pass. "Banks" still drills
// into which banks/products actually use each category, from real data.
useHead({ title: 'Product Management — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

interface Row { id: number; code: string; name: string; group_name: string; bank_count: number; field_count: number }
interface BankRow { bank_name: string; product_name: string }

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const groupFilter = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    rows.value = (await $fetch<{ data: Row[] }>('/api/v1/admin/products', { credentials: 'include' })).data ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const groups = computed(() => Array.from(new Set(rows.value.map((r) => r.group_name))))
const filtered = computed(() => (groupFilter.value ? rows.value.filter((r) => r.group_name === groupFilter.value) : rows.value))

const viewingBanks = ref<Row | null>(null)
const bankRows = ref<BankRow[]>([])
async function viewBanks(row: Row) {
  viewingBanks.value = row
  bankRows.value = (await $fetch<{ data: BankRow[] }>(`/api/v1/admin/products/${row.id}/banks`, { credentials: 'include' })).data ?? []
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Product Management</h1>
    </div>

    <div class="mb-4 flex flex-wrap gap-2 rounded-card border border-card-border bg-card p-3">
      <button
        v-for="g in ['', ...groups]"
        :key="g || 'all'"
        type="button"
        class="rounded-pill px-3 py-1.5 text-xs font-bold"
        :class="groupFilter === g ? 'bg-primary text-white' : 'bg-page text-navy'"
        @click="groupFilter = g"
      >{{ g || 'All' }}</button>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <NoResultsState v-else-if="filtered.length === 0" @clear="groupFilter = ''" />
    <div v-else class="overflow-x-auto rounded-card border border-card-border bg-card">
      <table class="w-full min-w-[640px] text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
            <th class="px-4 py-2.5 font-semibold">Product</th>
            <th class="px-4 py-2.5 font-semibold">Category</th>
            <th class="px-4 py-2.5 text-right font-semibold">Banks</th>
            <th class="px-4 py-2.5 text-right font-semibold">Fields</th>
            <th class="px-4 py-2.5 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in filtered" :key="r.id" class="border-b border-card-border last:border-none">
            <td class="px-4 py-2.5 font-semibold text-navy">{{ r.name }}</td>
            <td class="px-4 py-2.5 text-muted">{{ r.group_name }}</td>
            <td class="px-4 py-2.5 text-right text-navy">{{ r.bank_count }}</td>
            <td class="px-4 py-2.5 text-right text-navy">{{ r.field_count }}</td>
            <td class="px-4 py-2.5"><button type="button" class="font-bold text-primary hover:underline" @click="viewBanks(r)">Banks</button></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="viewingBanks" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-md rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Banks offering {{ viewingBanks.name }}</h3>
        <div class="mt-3 max-h-80 overflow-y-auto">
          <div v-for="(b, i) in bankRows" :key="i" class="flex justify-between border-b border-card-border py-2 text-sm last:border-none">
            <span class="font-semibold text-navy">{{ b.bank_name }}</span>
            <span class="text-muted">{{ b.product_name }}</span>
          </div>
          <p v-if="bankRows.length === 0" class="py-4 text-center text-sm text-muted">No banks currently offer this product.</p>
        </div>
        <button type="button" class="mt-4 w-full rounded-lg border border-card-border py-2 text-sm font-semibold" @click="viewingBanks = null">Close</button>
      </div>
    </div>
  </AdminShell>
</template>
