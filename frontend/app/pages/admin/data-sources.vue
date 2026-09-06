<script setup lang="ts">
useHead({ title: 'Data Sources — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface Row {
  source_id: number; bank_name: string; label: string; source_type: string
  source_url: string; status: string; last_checked: string | null; last_success: string | null; last_status: string
}

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const bank = ref('')
const sourceType = ref('')
const status = ref('')
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    rows.value = (await $fetch<{ data: Row[] }>('/api/v1/admin/data-sources', {
      credentials: 'include',
      query: { bank: bank.value || undefined, type: sourceType.value || undefined, status: status.value || undefined }
    })).data ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch([bank, sourceType, status], load)
function clearFilters() { bank.value = ''; sourceType.value = ''; status.value = '' }

async function toggle(row: Row) {
  const next = row.status === 'active' ? 'disabled' : 'active'
  try {
    await $fetch(`/api/v1/admin/data-sources/${row.source_id}`, { method: 'PATCH', credentials: 'include', body: { status: next } })
    toast.value = `Source ${next === 'active' ? 'enabled' : 'disabled'}.`
    await load()
  } catch {
    toast.value = ''
  }
}

async function runManual(row: Row) {
  try {
    await $fetch(`/api/v1/admin/data-sources/${row.source_id}/run`, { method: 'POST', credentials: 'include' })
    toast.value = 'Scrape triggered (local dev only — see note in README).'
  } catch {
    toast.value = ''
  }
}

function statusPill(s: string) {
  if (s === 'success') return { cls: 'bg-emerald-50 text-emerald-700', label: '✓ Success' }
  if (s === 'partial') return { cls: 'bg-amber-50 text-amber-700', label: '! Warning' }
  if (s === 'failed') return { cls: 'bg-red-50 text-red-700', label: '✕ Error' }
  return { cls: 'bg-page text-muted', label: 'Never run' }
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Data Sources</h1>
      <span class="rounded-pill bg-navy px-3 py-1 text-[11px] font-bold text-white">PRODUCTION</span>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-card border border-card-border bg-card p-3">
      <input v-model="bank" type="text" placeholder="Bank name…" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
      <select v-model="sourceType" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Type: HTML / PDF / API</option>
        <option value="HTML">HTML</option>
        <option value="PDF">PDF</option>
        <option value="API">API</option>
      </select>
      <select v-model="status" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Status: All</option>
        <option value="active">Active</option>
        <option value="disabled">Disabled</option>
      </select>
      <button type="button" class="text-xs font-semibold text-muted hover:text-primary" @click="clearFilters">Clear Filters</button>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <NoResultsState v-else-if="rows.length === 0" @clear="clearFilters" />
    <div v-else class="overflow-x-auto rounded-card border border-card-border bg-card">
      <table class="w-full min-w-[820px] text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
            <th class="px-4 py-2.5 font-semibold">Bank</th>
            <th class="px-4 py-2.5 font-semibold">Source</th>
            <th class="px-4 py-2.5 font-semibold">Type</th>
            <th class="px-4 py-2.5 font-semibold">Checked</th>
            <th class="px-4 py-2.5 font-semibold">Success</th>
            <th class="px-4 py-2.5 font-semibold">Status</th>
            <th class="px-4 py-2.5 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.source_id" class="border-b border-card-border last:border-none">
            <td class="px-4 py-2.5 font-semibold text-navy">{{ r.bank_name }}</td>
            <td class="px-4 py-2.5">
              <p class="text-navy">{{ r.label }}</p>
              <a :href="r.source_url" target="_blank" rel="noopener" class="block max-w-[220px] truncate text-xs text-primary hover:underline">{{ r.source_url }}</a>
            </td>
            <td class="px-4 py-2.5"><span class="rounded-pill bg-page px-2 py-0.5 text-[11px] font-bold text-navy">{{ r.source_type }}</span></td>
            <td class="px-4 py-2.5 text-muted">{{ r.last_checked ? fmtRelativeDate(r.last_checked) : '—' }}</td>
            <td class="px-4 py-2.5 text-muted">{{ r.last_success ? fmtRelativeDate(r.last_success) : '—' }}</td>
            <td class="px-4 py-2.5"><span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="statusPill(r.last_status).cls">{{ statusPill(r.last_status).label }}</span></td>
            <td class="px-4 py-2.5">
              <div class="flex items-center gap-2">
                <button
                  v-if="roleAtLeast('admin')"
                  type="button"
                  class="relative h-5 w-9 rounded-full transition"
                  :class="r.status === 'active' ? 'bg-primary' : 'bg-card-border'"
                  @click="toggle(r)"
                >
                  <span class="absolute top-0.5 h-4 w-4 rounded-full bg-white transition" :class="r.status === 'active' ? 'left-4' : 'left-0.5'" />
                </button>
                <button v-if="roleAtLeast('admin')" type="button" class="text-xs font-bold text-primary hover:underline" @click="runManual(r)">Run Manual</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
