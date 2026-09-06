<script setup lang="ts">
// A compact vertical-bar preview (not the full interactive line chart on
// the /history page — that one already exists and this links to it).
// Dependency-free CSS bars, since a handful of values doesn't need a
// charting library.
const props = defineProps<{
  points: { date: string; rate: number }[]
  historyHref: string
  recentChanges: { date: string; oldRate: number; newRate: number }[]
}>()

const rates = computed(() => props.points.map((p) => p.rate))
const minRate = computed(() => Math.min(...rates.value))
const maxRate = computed(() => Math.max(...rates.value))
const range = computed(() => Math.max(maxRate.value - minRate.value, 0.25))

function barHeightPct(rate: number): number {
  // Give the shortest bar a visible floor instead of collapsing to 0.
  return 15 + ((rate - minRate.value) / range.value) * 85
}

function fmtShort(iso: string): string {
  return new Date(iso).toLocaleDateString('en-LK', { month: 'short', year: '2-digit' })
}

// Up to 4 evenly-spaced x-axis labels across the point range.
const axisLabels = computed(() => {
  const pts = props.points
  if (pts.length === 0) return []
  const count = Math.min(4, pts.length)
  const step = (pts.length - 1) / Math.max(count - 1, 1)
  return Array.from({ length: count }, (_, i) => fmtShort(pts[Math.round(i * step)].date))
})
</script>

<template>
  <section class="rounded-[14px] border border-card-border bg-card p-5 shadow-sm">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-base font-bold text-navy">Rate History</h2>
      <NuxtLink :to="historyHref" class="text-[12.5px] font-bold text-primary hover:underline">View Full Timeline &rarr;</NuxtLink>
    </div>

    <template v-if="points.length > 1">
      <div class="flex h-20 items-end gap-1.5 border-b border-card-border pb-1">
        <div v-for="(p, i) in points" :key="i" class="flex-1 rounded-t bg-primary" :style="{ height: `${barHeightPct(p.rate)}%` }" />
      </div>
      <div class="mt-1 flex justify-between text-[10px] text-muted">
        <span v-for="(label, i) in axisLabels" :key="i">{{ label }}</span>
      </div>
    </template>
    <p v-else class="rounded-lg border border-dashed border-card-border p-3 text-xs leading-relaxed text-muted">
      Only one scrape recorded so far — a trend appears once at least two data points exist.
    </p>

    <template v-if="recentChanges.length">
      <p class="mb-1.5 mt-4 text-[10px] font-bold uppercase tracking-wide text-muted">Recent Rate Changes</p>
      <table class="w-full text-[11.5px]">
        <thead>
          <tr class="text-left text-[10px] uppercase tracking-wide text-muted">
            <th class="pb-1.5 font-semibold">Effective Date</th>
            <th class="pb-1.5 font-semibold">Old Rate</th>
            <th class="pb-1.5 text-right font-semibold">New Rate</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in recentChanges" :key="c.date" class="border-t border-card-border">
            <td class="py-1.5 text-muted">{{ fmtDate(c.date) }}</td>
            <td class="py-1.5 text-navy">{{ c.oldRate.toFixed(2) }}%</td>
            <td class="py-1.5 text-right font-bold text-primary">{{ c.newRate.toFixed(2) }}%</td>
          </tr>
        </tbody>
      </table>
    </template>
  </section>
</template>
