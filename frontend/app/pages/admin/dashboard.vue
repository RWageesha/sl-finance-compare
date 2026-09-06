<script setup lang="ts">
useHead({ title: 'Admin Dashboard — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

interface Stats {
  banks: number; products: number; active_rates: number
  pending_review: number; outdated_data: number; failed_sources: number
}
interface ScrapingRow { bank_name: string; last_run: string | null; status: string; records_found: number }
interface Alert { level: string; message: string; at: string }
interface RateChange { bank_name: string; product_name: string; old_rate: number | null; new_rate: number; rate_id: number; detected_at: string; status: string }

const loading = ref(true)
const error = ref(false)
const stats = ref<Stats | null>(null)
const scraping = ref<ScrapingRow[]>([])
const alerts = ref<Alert[]>([])
const recentChanges = ref<RateChange[]>([])
const toast = ref('')
const runningAll = ref(false)

async function load() {
  loading.value = true
  error.value = false
  try {
    const res = await $fetch<{ stats: Stats; scraping: ScrapingRow[]; alerts: Alert[]; recent_changes: RateChange[] }>('/api/v1/admin/dashboard', { credentials: 'include' })
    stats.value = res.stats
    scraping.value = res.scraping
    alerts.value = res.alerts
    recentChanges.value = res.recent_changes
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const STAT_CARDS = computed(() => stats.value ? [
  { label: 'Banks', value: stats.value.banks, icon: 'bank', color: 'bg-indigo-500' },
  { label: 'Products', value: stats.value.products, icon: 'tag', color: 'bg-orange-500' },
  { label: 'Active Rates', value: stats.value.active_rates, icon: 'percent', color: 'bg-emerald-500' },
  { label: 'Pending Review', value: stats.value.pending_review, icon: 'clock', color: 'bg-amber-500' },
  { label: 'Outdated Data', value: stats.value.outdated_data, icon: 'database', color: 'bg-purple-500' },
  { label: 'Failed Sources', value: stats.value.failed_sources, icon: 'close', color: 'bg-red-500' }
] : [])

async function runAllScrapers() {
  runningAll.value = true
  try {
    await $fetch('/api/v1/admin/scrapers/run-all', { method: 'POST', credentials: 'include' })
    toast.value = 'Scrapers triggered.'
  } catch {
    toast.value = ''
  } finally {
    runningAll.value = false
  }
}

function alertClass(level: string) {
  if (level === 'critical') return 'border-red-200 bg-red-50 text-red-700'
  if (level === 'warning') return 'border-amber-200 bg-amber-50 text-amber-700'
  return 'border-blue-200 bg-blue-50 text-blue-700'
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Admin Dashboard</h1>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <template v-else>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <div v-for="c in STAT_CARDS" :key="c.label" class="rounded-card border border-card-border bg-card p-4">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">{{ c.label }}</p>
          <p class="mt-1 text-2xl font-bold text-navy">{{ c.value }}</p>
          <span class="mt-2 flex h-8 w-8 items-center justify-center rounded-full text-white" :class="c.color">
            <Icon :name="c.icon" class="h-4 w-4" />
          </span>
        </div>
      </div>

      <div class="mt-5 grid grid-cols-1 gap-4 lg:grid-cols-[1.6fr_1fr]">
        <section class="rounded-card border border-card-border bg-card p-5">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-base font-bold text-navy">Scraping Status</h2>
            <button type="button" :disabled="runningAll" class="rounded-lg bg-primary px-3.5 py-1.5 text-xs font-bold text-white hover:bg-primary/90 disabled:opacity-60" @click="runAllScrapers">
              {{ runningAll ? 'Starting…' : 'Run All Scrapers' }}
            </button>
          </div>
          <NoResultsState v-if="scraping.length === 0" @clear="load" />
          <table v-else class="w-full text-sm">
            <thead>
              <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
                <th class="pb-2 font-semibold">Bank</th>
                <th class="pb-2 font-semibold">Last Run</th>
                <th class="pb-2 font-semibold">Status</th>
                <th class="pb-2 text-right font-semibold">Records</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in scraping" :key="s.bank_name" class="border-b border-card-border last:border-none">
                <td class="py-2 font-semibold text-navy">{{ s.bank_name }}</td>
                <td class="py-2 text-muted">{{ s.last_run ? fmtRelativeDate(s.last_run) : '—' }}</td>
                <td class="py-2">
                  <span
                    class="rounded-pill px-2 py-0.5 text-[11px] font-bold"
                    :class="s.status === 'success' ? 'bg-emerald-50 text-emerald-700' : s.status === 'partial' ? 'bg-amber-50 text-amber-700' : 'bg-red-50 text-red-700'"
                  >{{ s.status === 'success' ? '✓ OK' : s.status === 'partial' ? '! WARN' : '✕ FAILED' }}</span>
                </td>
                <td class="py-2 text-right text-navy">{{ s.records_found }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section class="rounded-card border border-card-border bg-card p-5">
          <h2 class="mb-3 text-base font-bold text-navy">System Alerts</h2>
          <div v-if="alerts.length === 0" class="text-sm text-muted">No alerts — everything looks healthy.</div>
          <div v-else class="flex flex-col gap-2">
            <div v-for="(a, i) in alerts" :key="i" class="rounded-lg border px-3 py-2 text-xs" :class="alertClass(a.level)">
              <span class="font-bold uppercase">[{{ a.level }}]</span> {{ a.message }}
            </div>
          </div>
        </section>
      </div>

      <section class="mt-4 rounded-card border border-card-border bg-card p-5">
        <h2 class="mb-3 text-base font-bold text-navy">Recent Rate Changes</h2>
        <NoResultsState v-if="recentChanges.length === 0" @clear="load" />
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[560px] text-sm">
            <thead>
              <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
                <th class="pb-2 font-semibold">Bank</th>
                <th class="pb-2 font-semibold">Product</th>
                <th class="pb-2 text-right font-semibold">Old</th>
                <th class="pb-2 text-right font-semibold">New</th>
                <th class="pb-2 font-semibold">Detected</th>
                <th class="pb-2 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in recentChanges" :key="c.rate_id" class="border-b border-card-border last:border-none">
                <td class="py-2 font-semibold text-navy">{{ c.bank_name }}</td>
                <td class="py-2 text-muted">{{ c.product_name }}</td>
                <td class="py-2 text-right text-muted">{{ c.old_rate?.toFixed(2) ?? '—' }}%</td>
                <td class="py-2 text-right font-bold text-navy">{{ c.new_rate.toFixed(2) }}%</td>
                <td class="py-2 text-muted">{{ fmtRelativeDate(c.detected_at) }}</td>
                <td class="py-2">
                  <span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="c.status === 'pending' ? 'bg-amber-50 text-amber-700' : 'bg-emerald-50 text-emerald-700'">
                    {{ c.status === 'pending' ? 'Pending' : 'Approved' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
