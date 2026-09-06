<script setup lang="ts">
// Route as ?product_id=&tenure_value=&tenure_label=&rate_label= rather
// than a single /[id]/ segment — the same product/tenure/rate-label line
// this whole project already uses to identify "one logical rate" (see
// fdCompare.ts / GetRateHistory), not a single product_rates row id.
useHead({ title: 'Rate History Management — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const route = useRoute()
const { roleAtLeast } = useAdminAuth()

interface HistoryRow { rate_id: number; rate: number; effective_date: string; changed_by: string; status: string }

const loading = ref(true)
const error = ref(false)
const rows = ref<HistoryRow[]>([])
const toast = ref('')
const showManualForm = ref(false)
const manualRate = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    rows.value = (await $fetch<{ data: HistoryRow[] }>('/api/v1/admin/rates/history', {
      credentials: 'include',
      query: {
        product_id: route.query.product_id,
        tenure_value: route.query.tenure_value,
        tenure_label: route.query.tenure_label,
        rate_label: route.query.rate_label
      }
    })).data ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const current = computed(() => rows.value[rows.value.length - 1])
const descRows = computed(() => [...rows.value].reverse())

async function addManualEntry() {
  if (!current.value) return
  const val = Number(manualRate.value)
  if (!Number.isFinite(val)) return
  try {
    await $fetch(`/api/v1/admin/rates/${current.value.rate_id}`, { method: 'PATCH', credentials: 'include', body: { new_rate: val } })
    toast.value = 'Manual entry added.'
    showManualForm.value = false
    manualRate.value = ''
    await load()
  } catch {
    toast.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <nav class="mb-4 text-xs text-muted">
      <NuxtLink to="/admin/rate-management" class="hover:text-primary">Rate Management</NuxtLink> &rsaquo; History
    </nav>
    <h1 class="mb-5 text-xl font-bold text-navy">Rate History - {{ route.query.bank }} {{ route.query.product }}</h1>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <template v-else>
      <div v-if="current" class="mb-5 rounded-lg bg-badge-bg px-4 py-3 text-sm text-navy">
        Current: <strong>{{ current.rate.toFixed(2) }}% p.a.</strong> — Effective: {{ fmtDate(current.effective_date) }} —
        <span class="font-semibold text-emerald-600">&#10003; {{ current.status === 'verified' ? 'Verified' : current.status }}</span>
        — {{ current.changed_by }}
      </div>

      <h2 class="mb-2 text-sm font-bold text-navy">Version History</h2>
      <div class="overflow-x-auto rounded-card border border-card-border bg-card">
        <table class="w-full min-w-[560px] text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
              <th class="px-4 py-2.5 font-semibold">Version</th>
              <th class="px-4 py-2.5 text-right font-semibold">Rate</th>
              <th class="px-4 py-2.5 font-semibold">Effective Date</th>
              <th class="px-4 py-2.5 font-semibold">Changed By</th>
              <th class="px-4 py-2.5 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(h, i) in descRows" :key="h.rate_id" class="border-b border-card-border last:border-none">
              <td class="px-4 py-2.5 text-muted">v{{ descRows.length - i }}</td>
              <td class="px-4 py-2.5 text-right font-bold text-navy">{{ h.rate.toFixed(2) }}%</td>
              <td class="px-4 py-2.5 text-muted">{{ fmtDate(h.effective_date) }}</td>
              <td class="px-4 py-2.5 text-muted">{{ h.changed_by }}</td>
              <td class="px-4 py-2.5">
                <span class="text-xs font-semibold" :class="h.status === 'verified' ? 'text-emerald-600' : h.status === 'pending' ? 'text-amber-600' : 'text-red-600'">
                  &#10003; {{ h.status }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <button v-if="roleAtLeast('admin') && !showManualForm" type="button" class="mt-3 text-sm font-bold text-primary hover:underline" @click="showManualForm = true">
        + Add Manual Entry
      </button>
      <div v-if="showManualForm" class="mt-3 flex items-center gap-2 rounded-card border border-card-border bg-card p-4">
        <input v-model="manualRate" type="number" step="0.01" placeholder="New rate %" class="rounded-lg border border-card-border px-3 py-2 text-sm">
        <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="addManualEntry">Save</button>
        <button type="button" class="text-sm font-semibold text-muted" @click="showManualForm = false">Cancel</button>
      </div>
    </template>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
