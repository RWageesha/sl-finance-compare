<script setup lang="ts">
useHead({ title: 'Scraping Jobs — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface Row {
  run_id: number; bank_name: string; source_label: string; source_id: number
  started_at: string; completed_at: string | null; found: number; changed: number
  status: string; error_message: string
}
interface Detail extends Row {
  source_url: string; consecutive_fails: number; expected_records: number
}

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const totalToday = ref(0)
const successRuns = ref(0)
const failedRuns = ref(0)
const expandedId = ref<number | null>(null)
const detail = ref<Detail | null>(null)
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    const res = await $fetch<{ data: Row[]; total_jobs_today: number; successful_runs: number; failed_attempts: number }>('/api/v1/admin/scraping-jobs', { credentials: 'include' })
    rows.value = res.data ?? []
    totalToday.value = res.total_jobs_today
    successRuns.value = res.successful_runs
    failedRuns.value = res.failed_attempts
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

function duration(row: Row): string {
  if (!row.completed_at) return '—'
  const ms = new Date(row.completed_at).getTime() - new Date(row.started_at).getTime()
  return `${Math.round(ms / 1000)}s`
}

async function toggleDetail(row: Row) {
  if (expandedId.value === row.run_id) {
    expandedId.value = null
    return
  }
  expandedId.value = row.run_id
  detail.value = null
  try {
    detail.value = await $fetch<Detail>(`/api/v1/admin/scraping-jobs/${row.run_id}`, { credentials: 'include' })
  } catch {
    detail.value = null
  }
}

async function retry(row: Row) {
  try {
    await $fetch(`/api/v1/admin/data-sources/${row.source_id}/run`, { method: 'POST', credentials: 'include' })
    toast.value = 'Retry triggered (local dev only).'
  } catch {
    toast.value = ''
  }
}

async function disableSource(sourceId: number) {
  try {
    await $fetch(`/api/v1/admin/data-sources/${sourceId}`, { method: 'PATCH', credentials: 'include', body: { status: 'disabled' } })
    toast.value = 'Source disabled.'
  } catch {
    toast.value = ''
  }
}

function statusMeta(s: string) {
  if (s === 'success') return { cls: 'bg-emerald-50 text-emerald-700', label: '✓ SUCCESS' }
  if (s === 'partial') return { cls: 'bg-amber-50 text-amber-700', label: '! PARTIAL' }
  return { cls: 'bg-red-50 text-red-700', label: '✕ FAILED' }
}
</script>

<template>
  <AdminShell>
    <h1 class="mb-5 text-xl font-bold text-navy">Scraping Jobs</h1>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <template v-else>
      <div class="mb-4 grid grid-cols-3 gap-3">
        <div class="rounded-card border border-card-border bg-card p-4">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Total Jobs Today</p>
          <p class="mt-1 text-2xl font-bold text-navy">{{ totalToday }}</p>
        </div>
        <div class="rounded-card border border-card-border bg-card p-4">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Successful Runs</p>
          <p class="mt-1 text-2xl font-bold text-emerald-600">{{ successRuns }}</p>
        </div>
        <div class="rounded-card border border-card-border bg-card p-4">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">Failed Attempts</p>
          <p class="mt-1 text-2xl font-bold text-red-600">{{ failedRuns }}</p>
        </div>
      </div>

      <NoResultsState v-if="rows.length === 0" @clear="load" />
      <div v-else class="overflow-x-auto rounded-card border border-card-border bg-card">
        <table class="w-full min-w-[820px] text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
              <th class="px-4 py-2.5 font-semibold">Bank</th>
              <th class="px-4 py-2.5 font-semibold">Source</th>
              <th class="px-4 py-2.5 font-semibold">Start</th>
              <th class="px-4 py-2.5 font-semibold">Duration</th>
              <th class="px-4 py-2.5 text-right font-semibold">Found</th>
              <th class="px-4 py-2.5 text-right font-semibold">Changed</th>
              <th class="px-4 py-2.5 font-semibold">Status</th>
              <th class="px-4 py-2.5 font-semibold">Action</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="r in rows" :key="r.run_id">
              <tr class="cursor-pointer border-b border-card-border last:border-none hover:bg-page" @click="toggleDetail(r)">
                <td class="px-4 py-2.5 font-semibold text-navy">{{ r.bank_name }}</td>
                <td class="px-4 py-2.5 text-muted">{{ r.source_label }}</td>
                <td class="px-4 py-2.5 text-muted">{{ fmtDate(r.started_at) }}</td>
                <td class="px-4 py-2.5 text-muted">{{ duration(r) }}</td>
                <td class="px-4 py-2.5 text-right text-navy">{{ r.found }}</td>
                <td class="px-4 py-2.5 text-right text-navy">{{ r.changed }}</td>
                <td class="px-4 py-2.5"><span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="statusMeta(r.status).cls">{{ statusMeta(r.status).label }}</span></td>
                <td class="px-4 py-2.5">
                  <button v-if="r.status === 'failed' && roleAtLeast('admin')" type="button" class="font-bold text-primary hover:underline" @click.stop="retry(r)">Retry</button>
                </td>
              </tr>
              <tr v-if="expandedId === r.run_id" class="border-b border-card-border bg-page">
                <td colspan="8" class="px-4 py-4">
                  <div v-if="!detail" class="text-sm text-muted">Loading detail…</div>
                  <div v-else class="rounded-card border border-card-border bg-card p-4">
                    <p v-if="detail.consecutive_fails > 1" class="mb-3 rounded-lg bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-700">
                      &#9888; This source has failed {{ detail.consecutive_fails }} consecutive times
                    </p>
                    <div class="grid grid-cols-2 gap-3 text-xs text-muted sm:grid-cols-4">
                      <div><span class="block font-bold uppercase text-[10px]">Source</span>{{ detail.source_url }}</div>
                      <div><span class="block font-bold uppercase text-[10px]">Type</span>{{ detail.status }}</div>
                      <div><span class="block font-bold uppercase text-[10px]">Expected</span>{{ detail.expected_records }}</div>
                      <div><span class="block font-bold uppercase text-[10px]">Current</span>{{ detail.found }}</div>
                    </div>
                    <template v-if="detail.error_message">
                      <p class="mb-1 mt-3 text-[11px] font-bold uppercase tracking-wide text-muted">Error Details</p>
                      <pre class="overflow-x-auto rounded-lg bg-navy p-3 text-xs text-white">{{ detail.error_message }}</pre>
                    </template>
                    <div v-if="roleAtLeast('admin')" class="mt-3 flex gap-4 text-xs font-bold">
                      <button type="button" class="text-primary hover:underline" @click="retry(r)">Retry Now</button>
                      <button type="button" class="text-primary hover:underline" @click="disableSource(r.source_id)">Disable Source</button>
                      <span class="text-muted" title="No task-tracking system exists in this project yet">Create Task (stub)</span>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </template>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
