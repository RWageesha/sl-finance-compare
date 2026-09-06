<script setup lang="ts">
useHead({ title: 'Bank Management — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface Row {
  id: number; name: string; code: string; bank_type: string; status: string
  website: string; product_count: number; source_count: number; health_pct: number | null; updated_at: string | null
}

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const search = ref('')
const typeFilter = ref('')
const statusFilter = ref('')
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    rows.value = (await $fetch<{ data: Row[] }>('/api/v1/admin/banks', { credentials: 'include' })).data ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const filtered = computed(() => rows.value.filter((r) => {
  if (search.value && !r.name.toLowerCase().includes(search.value.toLowerCase())) return false
  if (typeFilter.value && r.bank_type !== typeFilter.value) return false
  if (statusFilter.value && r.status !== statusFilter.value) return false
  return true
}))

const editing = ref<Row | null>(null)
const form = reactive({ name: '', bank_type: '', website: '', status: 'active' })
const adding = ref(false)
const addForm = reactive({ name: '', code: '', bank_type: 'Commercial Bank', website: '' })

function openEdit(row: Row) {
  editing.value = row
  form.name = row.name; form.bank_type = row.bank_type; form.website = row.website; form.status = row.status
}
async function saveEdit() {
  if (!editing.value) return
  try {
    await $fetch(`/api/v1/admin/banks/${editing.value.id}`, { method: 'PATCH', credentials: 'include', body: form })
    toast.value = 'Bank updated.'
    editing.value = null
    await load()
  } catch {
    toast.value = ''
  }
}
async function saveAdd() {
  try {
    await $fetch('/api/v1/admin/banks', { method: 'POST', credentials: 'include', body: addForm })
    toast.value = 'Bank added.'
    adding.value = false
    addForm.name = ''; addForm.code = ''; addForm.website = ''
    await load()
  } catch {
    toast.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Bank Management</h1>
      <button v-if="roleAtLeast('admin')" type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="adding = true">+ Add Bank</button>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-card border border-card-border bg-card p-3">
      <input v-model="search" type="text" placeholder="Search banks…" class="min-w-[200px] flex-1 rounded-lg border border-card-border px-3 py-1.5 text-sm">
      <select v-model="typeFilter" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Type: All</option>
        <option value="Commercial Bank">Commercial Bank</option>
        <option value="Savings Bank">Savings Bank</option>
        <option value="Licensed Commercial Bank">Licensed Commercial Bank</option>
      </select>
      <select v-model="statusFilter" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Status: All</option>
        <option value="active">Active</option>
        <option value="inactive">Inactive</option>
      </select>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <NoResultsState v-else-if="filtered.length === 0" @clear="search = ''; typeFilter = ''; statusFilter = ''" />
    <div v-else class="overflow-x-auto rounded-card border border-card-border bg-card">
      <table class="w-full min-w-[760px] text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
            <th class="px-4 py-2.5 font-semibold">Bank Name</th>
            <th class="px-4 py-2.5 font-semibold">Type</th>
            <th class="px-4 py-2.5 font-semibold">Status</th>
            <th class="px-4 py-2.5 text-right font-semibold">Products</th>
            <th class="px-4 py-2.5 text-right font-semibold">Sources</th>
            <th class="px-4 py-2.5 text-right font-semibold">Health</th>
            <th class="px-4 py-2.5 font-semibold">Updated</th>
            <th class="px-4 py-2.5 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in filtered" :key="r.id" class="border-b border-card-border last:border-none">
            <td class="px-4 py-2.5 font-semibold text-navy">{{ r.name }}</td>
            <td class="px-4 py-2.5 text-muted">{{ r.bank_type }}</td>
            <td class="px-4 py-2.5">
              <span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="r.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-page text-muted'">{{ r.status }}</span>
            </td>
            <td class="px-4 py-2.5 text-right text-navy">{{ r.product_count }}</td>
            <td class="px-4 py-2.5 text-right text-navy">{{ r.source_count }}</td>
            <td class="px-4 py-2.5 text-right font-bold" :class="r.health_pct === null ? 'text-muted' : r.health_pct >= 90 ? 'text-emerald-600' : 'text-amber-600'">
              {{ r.health_pct !== null ? r.health_pct + '%' : '—' }}
            </td>
            <td class="px-4 py-2.5 text-muted">{{ r.updated_at ? fmtRelativeDate(r.updated_at) : '—' }}</td>
            <td class="px-4 py-2.5"><button v-if="roleAtLeast('admin')" type="button" class="font-bold text-primary hover:underline" @click="openEdit(r)">Manage</button></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-sm rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Manage {{ editing.name }}</h3>
        <div class="mt-4 flex flex-col gap-3">
          <div><label class="mb-1 block text-xs font-semibold text-navy">Name</label><input v-model="form.name" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
          <div><label class="mb-1 block text-xs font-semibold text-navy">Type</label>
            <select v-model="form.bank_type" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
              <option>Commercial Bank</option><option>Savings Bank</option><option>Licensed Commercial Bank</option>
            </select>
          </div>
          <div><label class="mb-1 block text-xs font-semibold text-navy">Website</label><input v-model="form.website" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
          <div><label class="mb-1 block text-xs font-semibold text-navy">Status</label>
            <select v-model="form.status" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"><option value="active">Active</option><option value="inactive">Inactive</option></select>
          </div>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold" @click="editing = null">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="saveEdit">Save</button>
        </div>
      </div>
    </div>

    <div v-if="adding" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-sm rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Add Bank</h3>
        <div class="mt-4 flex flex-col gap-3">
          <div><label class="mb-1 block text-xs font-semibold text-navy">Name</label><input v-model="addForm.name" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
          <div><label class="mb-1 block text-xs font-semibold text-navy">Code</label><input v-model="addForm.code" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm" placeholder="e.g. SEYLAN"></div>
          <div><label class="mb-1 block text-xs font-semibold text-navy">Website</label><input v-model="addForm.website" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold" @click="adding = false">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="saveAdd">Add</button>
        </div>
      </div>
    </div>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
