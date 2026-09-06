<script setup lang="ts">
useHead({ title: 'Audit Logs — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

interface Row { id: number; admin_username: string; action: string; record_ref: string; old_value: string; new_value: string; created_at: string }

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const total = ref(0)
const admins = ref<string[]>([])
const adminFilter = ref('')
const actionFilter = ref('')
const page = ref(1)
const PER_PAGE = 20
const confirmClear = ref(false)
const viewingJson = ref<Row | null>(null)
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    const res = await $fetch<{ data: Row[]; total: number; admins: string[] }>('/api/v1/admin/audit-logs', {
      credentials: 'include',
      query: { admin: adminFilter.value || undefined, action: actionFilter.value || undefined, page: page.value }
    })
    rows.value = res.data ?? []
    total.value = res.total
    admins.value = res.admins ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch([adminFilter, actionFilter], () => { page.value = 1; load() })
watch(page, load)

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / PER_PAGE)))
const actions = computed(() => Array.from(new Set(rows.value.map((r) => r.action))))

async function clearLogs() {
  try {
    await $fetch('/api/v1/admin/audit-logs', { method: 'DELETE', credentials: 'include' })
    toast.value = 'Audit logs cleared.'
    confirmClear.value = false
    await load()
  } catch {
    toast.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Audit Logs</h1>
      <button type="button" class="rounded-lg border border-red-200 px-3.5 py-1.5 text-xs font-bold text-red-600 hover:bg-red-50" @click="confirmClear = true">Clear Logs</button>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-card border border-card-border bg-card p-3">
      <select v-model="adminFilter" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Admin: All</option>
        <option v-for="a in admins" :key="a" :value="a">{{ a }}</option>
      </select>
      <select v-model="actionFilter" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Action: All</option>
        <option v-for="a in actions" :key="a" :value="a">{{ a }}</option>
      </select>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <NoResultsState v-else-if="rows.length === 0" @clear="adminFilter = ''; actionFilter = ''" />
    <template v-else>
      <div class="overflow-x-auto rounded-card border border-card-border bg-card">
        <table class="w-full min-w-[760px] text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
              <th class="px-4 py-2.5 font-semibold">Timestamp</th>
              <th class="px-4 py-2.5 font-semibold">Admin</th>
              <th class="px-4 py-2.5 font-semibold">Action</th>
              <th class="px-4 py-2.5 font-semibold">Record</th>
              <th class="px-4 py-2.5 font-semibold">Old Value</th>
              <th class="px-4 py-2.5 font-semibold">New Value</th>
              <th class="px-4 py-2.5 font-semibold"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.id" class="border-b border-card-border last:border-none">
              <td class="px-4 py-2.5 text-muted">{{ fmtDate(r.created_at) }}</td>
              <td class="px-4 py-2.5 font-semibold text-navy">{{ r.admin_username }}</td>
              <td class="px-4 py-2.5"><span class="rounded-pill bg-page px-2 py-0.5 text-[11px] font-bold text-navy">{{ r.action }}</span></td>
              <td class="px-4 py-2.5 text-muted">{{ r.record_ref }}</td>
              <td class="px-4 py-2.5 text-muted">{{ r.old_value || '—' }}</td>
              <td class="px-4 py-2.5 font-semibold text-navy">{{ r.new_value || '—' }}</td>
              <td class="px-4 py-2.5"><button type="button" class="text-xs font-bold text-primary hover:underline" @click="viewingJson = r">View JSON</button></td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="pageCount > 1" class="mt-4 flex items-center justify-between text-sm text-muted">
        <span>Page {{ page }} of {{ pageCount }} ({{ total }} entries)</span>
        <div class="flex gap-2">
          <button type="button" class="rounded-lg border border-card-border px-3 py-1.5 disabled:opacity-40" :disabled="page === 1" @click="page--">Previous</button>
          <button type="button" class="rounded-lg border border-card-border px-3 py-1.5 disabled:opacity-40" :disabled="page === pageCount" @click="page++">Next</button>
        </div>
      </div>
    </template>

    <div v-if="viewingJson" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="w-full max-w-lg rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">Raw Log Entry</h3>
        <pre class="mt-3 max-h-80 overflow-auto rounded-lg bg-navy p-3 text-xs text-white">{{ JSON.stringify(viewingJson, null, 2) }}</pre>
        <button type="button" class="mt-4 w-full rounded-lg border border-card-border py-2 text-sm font-semibold" @click="viewingJson = null">Close</button>
      </div>
    </div>

    <ConfirmationModal v-if="confirmClear" message="Every audit log entry will be permanently deleted." confirm-label="Clear Logs" @confirm="clearLogs" @cancel="confirmClear = false" />
    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
