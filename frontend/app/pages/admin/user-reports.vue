<script setup lang="ts">
useHead({ title: 'User Reports — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface Row {
  id: number; bank_name: string; product_label: string; issue_type: string
  current_value: string; correct_value: string; source_url: string; description: string
  status: string; created_at: string
}

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const status = ref('pending')
const selected = ref<Row | null>(null)
const toast = ref('')
const confirmReject = ref(false)

async function load() {
  loading.value = true
  error.value = false
  try {
    rows.value = (await $fetch<{ data: Row[] }>('/api/v1/admin/user-reports', { credentials: 'include', query: { status: status.value || undefined } })).data ?? []
    if (rows.value.length && !selected.value) selected.value = rows.value[0]
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch(status, load)

async function resolve() {
  if (!selected.value) return
  try {
    await $fetch(`/api/v1/admin/user-reports/${selected.value.id}/resolve`, { method: 'POST', credentials: 'include' })
    toast.value = 'Report resolved.'
    selected.value = null
    await load()
  } catch {
    toast.value = ''
  }
}
async function doReject() {
  if (!selected.value) return
  try {
    await $fetch(`/api/v1/admin/user-reports/${selected.value.id}/reject`, { method: 'POST', credentials: 'include' })
    toast.value = 'Report rejected.'
    confirmReject.value = false
    selected.value = null
    await load()
  } catch {
    toast.value = ''
  }
}
async function requestInfo() {
  if (!selected.value) return
  try {
    await $fetch(`/api/v1/admin/user-reports/${selected.value.id}/request-info`, { method: 'POST', credentials: 'include' })
    toast.value = 'Marked as needing more info.'
    await load()
  } catch {
    toast.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">User Reports</h1>
      <select v-model="status" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Status: All</option>
        <option value="pending">Pending</option>
        <option value="resolved">Resolved</option>
        <option value="rejected">Rejected</option>
      </select>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <EmptyState v-else-if="rows.length === 0" heading="No reports" subtext="Corrections submitted from the public site's Report Issue form will appear here." />
    <div v-else class="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_1.2fr]">
      <div class="overflow-x-auto rounded-card border border-card-border bg-card">
        <table class="w-full min-w-[420px] text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
              <th class="px-3 py-2 font-semibold">ID</th>
              <th class="px-3 py-2 font-semibold">Issue</th>
              <th class="px-3 py-2 font-semibold">Status</th>
              <th class="px-3 py-2 font-semibold"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.id" class="cursor-pointer border-b border-card-border last:border-none hover:bg-page" :class="selected?.id === r.id ? 'bg-badge-bg' : ''" @click="selected = r">
              <td class="px-3 py-2 font-semibold text-navy">REP-{{ r.id }}</td>
              <td class="px-3 py-2 text-muted">{{ r.issue_type }}</td>
              <td class="px-3 py-2">
                <span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="r.status === 'pending' ? 'bg-amber-50 text-amber-700' : r.status === 'resolved' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'">{{ r.status }}</span>
              </td>
              <td class="px-3 py-2 text-right"><span class="font-bold text-primary">View</span></td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="selected" class="rounded-card border border-card-border bg-card p-5">
        <p class="text-xs font-bold uppercase tracking-wide text-muted">Report Details: REP-{{ selected.id }}</p>
        <p class="mt-2 text-base font-bold text-navy">{{ selected.issue_type }}</p>

        <p class="mt-3 text-[11px] font-bold uppercase tracking-wide text-muted">Affected Bank &amp; Product</p>
        <p class="text-sm text-navy">{{ selected.bank_name }} — {{ selected.product_label }}</p>

        <p class="mt-3 text-[11px] font-bold uppercase tracking-wide text-muted">Submission Details</p>
        <p class="text-sm text-navy">Reported {{ fmtDate(selected.created_at) }}</p>

        <p v-if="selected.description" class="mt-3 text-[11px] font-bold uppercase tracking-wide text-muted">Reporter Description</p>
        <p v-if="selected.description" class="rounded-lg bg-page p-3 text-sm text-navy">{{ selected.description }}</p>

        <p class="mt-3 text-[11px] font-bold uppercase tracking-wide text-muted">Values</p>
        <p class="text-sm text-navy">Current: {{ selected.current_value || '—' }} &rarr; Suggested: <strong>{{ selected.correct_value }}</strong></p>

        <a :href="selected.source_url" target="_blank" rel="noopener" class="mt-1 block break-all text-xs text-primary hover:underline">{{ selected.source_url }}</a>

        <div v-if="roleAtLeast('editor') && selected.status === 'pending'" class="mt-5 flex flex-col gap-2">
          <button type="button" class="rounded-lg bg-primary py-2.5 text-sm font-bold text-white hover:bg-primary/90" @click="resolve">Resolve &amp; Update Rate</button>
          <div class="flex gap-2">
            <button type="button" class="flex-1 rounded-lg border border-red-200 py-2 text-sm font-bold text-red-600 hover:bg-red-50" @click="confirmReject = true">Reject Report</button>
            <button type="button" class="flex-1 rounded-lg border border-card-border py-2 text-sm font-semibold text-navy" @click="requestInfo">Request Info</button>
          </div>
        </div>
      </div>
    </div>

    <ConfirmationModal v-if="confirmReject" message="This report will be marked rejected." confirm-label="Reject" @confirm="doReject" @cancel="confirmReject = false" />
    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
