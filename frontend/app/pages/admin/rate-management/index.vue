<script setup lang="ts">
useHead({ title: 'Rate Management — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface Row {
  rate_id: number; product_id: number; bank_name: string; product_name: string
  tenure_value: number | null; tenure_label: string; rate_label: string; current_rate: number
  effective_from: string; status: string; last_checked: string
}

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const total = ref(0)
const search = ref('')
const bank = ref('')
const category = ref('')
const status = ref('')
const page = ref(1)
const PER_PAGE = 10
const toast = ref('')

const editingRow = ref<Row | null>(null)
const newRate = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    const res = await $fetch<{ data: Row[]; total: number }>('/api/v1/admin/rates', {
      credentials: 'include',
      query: { search: search.value || undefined, bank: bank.value || undefined, category: category.value || undefined, status: status.value || undefined, page: page.value }
    })
    rows.value = res.data ?? []
    total.value = res.total
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch([search, bank, category, status], () => { page.value = 1; load() })
watch(page, load)

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / PER_PAGE)))

function clearFilters() {
  search.value = ''; bank.value = ''; category.value = ''; status.value = ''
}

function statusPill(s: string) {
  if (s === 'verified') return 'bg-emerald-50 text-emerald-700'
  if (s === 'pending') return 'bg-amber-50 text-amber-700'
  return 'bg-red-50 text-red-700'
}

function openEdit(row: Row) {
  editingRow.value = row
  newRate.value = String(row.current_rate)
}
async function submitEdit() {
  if (!editingRow.value) return
  const val = Number(newRate.value)
  if (!Number.isFinite(val)) return
  try {
    await $fetch(`/api/v1/admin/rates/${editingRow.value.rate_id}`, { method: 'PATCH', credentials: 'include', body: { new_rate: val } })
    toast.value = 'Rate updated.'
    editingRow.value = null
    await load()
  } catch {
    toast.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Rate Management</h1>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-card border border-card-border bg-card p-3">
      <input v-model="search" type="text" placeholder="Search banks or products…" class="min-w-[200px] flex-1 rounded-lg border border-card-border px-3 py-1.5 text-sm text-navy">
      <select v-model="category" class="rounded-lg border border-card-border px-3 py-1.5 text-sm text-navy">
        <option value="">Product: All Types</option>
        <option value="FIXED_DEPOSIT">Fixed Deposits</option>
        <option value="SAVINGS">Savings</option>
        <option value="LOAN">Loans</option>
      </select>
      <select v-model="status" class="rounded-lg border border-card-border px-3 py-1.5 text-sm text-navy">
        <option value="">Status: Active</option>
        <option value="verified">Verified</option>
        <option value="pending">Pending</option>
        <option value="rejected">Rejected</option>
      </select>
      <button type="button" class="text-xs font-semibold text-muted hover:text-primary" @click="clearFilters">Clear</button>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <NoResultsState v-else-if="rows.length === 0" @clear="clearFilters" />
    <template v-else>
      <div class="overflow-x-auto rounded-card border border-card-border bg-card">
        <table class="w-full min-w-[720px] text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
              <th class="px-4 py-2.5 font-semibold">Bank</th>
              <th class="px-4 py-2.5 font-semibold">Product</th>
              <th class="px-4 py-2.5 text-right font-semibold">Current Rate</th>
              <th class="px-4 py-2.5 font-semibold">Effective From</th>
              <th class="px-4 py-2.5 font-semibold">Status</th>
              <th class="px-4 py-2.5 font-semibold">Last Checked</th>
              <th class="px-4 py-2.5 font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.rate_id" class="border-b border-card-border last:border-none">
              <td class="px-4 py-2.5 font-semibold text-navy">{{ r.bank_name }}</td>
              <td class="px-4 py-2.5 text-muted">{{ r.product_name }} <span v-if="r.tenure_label">({{ r.tenure_label }})</span></td>
              <td class="px-4 py-2.5 text-right font-bold text-navy">{{ r.current_rate.toFixed(2) }}%</td>
              <td class="px-4 py-2.5 text-muted">{{ fmtDate(r.effective_from) }}</td>
              <td class="px-4 py-2.5"><span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="statusPill(r.status)">{{ r.status }}</span></td>
              <td class="px-4 py-2.5 text-muted">{{ fmtRelativeDate(r.last_checked) }}</td>
              <td class="px-4 py-2.5">
                <button v-if="roleAtLeast('admin')" type="button" class="mr-3 font-bold text-primary hover:underline" @click="openEdit(r)">Edit</button>
                <NuxtLink
                  :to="{ path: '/admin/rate-management/history', query: { product_id: r.product_id, tenure_value: r.tenure_value ?? undefined, tenure_label: r.tenure_label, rate_label: r.rate_label, bank: r.bank_name, product: r.product_name } }"
                  class="font-bold text-primary hover:underline"
                >History</NuxtLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="pageCount > 1" class="mt-4 flex items-center justify-between text-sm text-muted">
        <span>Showing page {{ page }} of {{ pageCount }} ({{ total }} results)</span>
        <div class="flex gap-2">
          <button type="button" class="rounded-lg border border-card-border px-3 py-1.5 disabled:opacity-40" :disabled="page === 1" @click="page--">Previous</button>
          <button type="button" class="rounded-lg border border-card-border px-3 py-1.5 disabled:opacity-40" :disabled="page === pageCount" @click="page++">Next</button>
        </div>
      </div>
    </template>

    <div v-if="editingRow" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-sm rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Edit Rate</h3>
        <p class="mt-1 text-xs text-muted">{{ editingRow.bank_name }} — {{ editingRow.product_name }}</p>
        <label class="mb-1 mt-4 block text-sm font-semibold text-navy">New Rate (%)</label>
        <input v-model="newRate" type="number" step="0.01" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm text-navy">
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold text-navy" @click="editingRow = null">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="submitEdit">Save</button>
        </div>
      </div>
    </div>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
