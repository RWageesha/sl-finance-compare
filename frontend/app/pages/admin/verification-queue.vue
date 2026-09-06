<script setup lang="ts">
useHead({ title: 'Verification Queue — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface Row {
  rate_id: number; bank_name: string; product_name: string; category_code: string
  prev_rate: number | null; new_rate: number; source_url: string; confidence: string; detected_at: string
}

const loading = ref(true)
const error = ref(false)
const rows = ref<Row[]>([])
const category = ref('')
const bank = ref('')
const toast = ref('')
const confirmTarget = ref<{ id: number; action: 'approve' | 'reject' } | null>(null)

async function load() {
  loading.value = true
  error.value = false
  try {
    rows.value = (await $fetch<{ data: Row[] }>('/api/v1/admin/verification-queue', {
      credentials: 'include',
      query: { category: category.value || undefined, bank: bank.value || undefined }
    })).data ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch([category, bank], load)

function clearFilters() {
  category.value = ''
  bank.value = ''
}

async function approve(id: number) {
  try {
    await $fetch(`/api/v1/admin/rates/${id}/approve`, { method: 'POST', credentials: 'include' })
    toast.value = 'Rate approved.'
    await load()
  } catch {
    toast.value = ''
  }
}

function confirmReject(id: number) {
  confirmTarget.value = { id, action: 'reject' }
}
async function doReject() {
  if (!confirmTarget.value) return
  try {
    await $fetch(`/api/v1/admin/rates/${confirmTarget.value.id}/reject`, { method: 'POST', credentials: 'include' })
    toast.value = 'Rate rejected.'
    await load()
  } catch {
    toast.value = ''
  } finally {
    confirmTarget.value = null
  }
}
</script>

<template>
  <AdminShell>
    <h1 class="mb-5 text-xl font-bold text-navy">Verification Queue</h1>

    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-card border border-card-border bg-card p-3">
      <select class="rounded-lg border border-card-border px-3 py-1.5 text-sm text-navy" disabled>
        <option>Status: Pending Review</option>
      </select>
      <select v-model="category" class="rounded-lg border border-card-border px-3 py-1.5 text-sm text-navy">
        <option value="">Product Type: All</option>
        <option value="FIXED_DEPOSIT">Fixed Deposits</option>
        <option value="SAVINGS">Savings</option>
        <option value="LOAN">Loans</option>
      </select>
      <input v-model="bank" type="text" placeholder="Filter by bank name…" class="rounded-lg border border-card-border px-3 py-1.5 text-sm text-navy">
      <button type="button" class="text-xs font-semibold text-muted hover:text-primary" @click="clearFilters">Clear All</button>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <EmptyState v-else-if="rows.length === 0" heading="Nothing pending review" subtext="New scraped rates will appear here for approval before they're marked verified." />
    <div v-else class="flex flex-col gap-3">
      <div v-for="row in rows" :key="row.rate_id" class="rounded-card border border-card-border bg-card p-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-xs text-muted">{{ row.bank_name }}</p>
            <p class="font-bold text-navy">{{ row.product_name }}</p>
          </div>
          <span class="text-xs text-muted">Detected {{ fmtRelativeDate(row.detected_at) }}</span>
        </div>
        <div class="mt-3 flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-3 text-sm">
            <span class="text-muted">{{ row.prev_rate !== null ? row.prev_rate.toFixed(2) + '%' : 'New' }}</span>
            <span class="text-muted">&rarr;</span>
            <strong class="text-primary">{{ row.new_rate.toFixed(2) }}%</strong>
            <a v-if="row.source_url" :href="row.source_url" target="_blank" rel="noopener" class="text-xs font-semibold text-primary hover:underline">View Source</a>
            <span class="text-xs text-muted">Confidence: {{ row.confidence }}</span>
          </div>
          <div v-if="roleAtLeast('editor')" class="flex gap-2">
            <button type="button" class="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-bold text-red-600 hover:bg-red-50" @click="confirmReject(row.rate_id)">Reject</button>
            <button type="button" class="rounded-lg bg-primary px-3 py-1.5 text-xs font-bold text-white hover:bg-primary/90" @click="approve(row.rate_id)">Approve</button>
          </div>
        </div>
      </div>
    </div>

    <ConfirmationModal v-if="confirmTarget" message="This rate will be marked rejected in the audit trail." confirm-label="Reject" @confirm="doReject" @cancel="confirmTarget = null" />
    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
