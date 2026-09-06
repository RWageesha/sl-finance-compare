<script setup lang="ts">
// Dependency-free SVG multi-series line chart — same rationale as
// RateTrendChart.vue (a handful of points per bank doesn't need a
// charting library), extended to draw one line per tracked bank.
const props = defineProps<{
  series: { label: string; color: string; points: { date: string; rate: number }[] }[]
}>()

const WIDTH = 900
const HEIGHT = 280
const PAD_X = 48
const PAD_TOP = 24
// Deliberately taller than PAD_TOP so the min-rate label (just above
// the axis line) and the month labels (below it) have their own
// separate bands and never overlap.
const PAD_BOTTOM = 34
const axisY = HEIGHT - PAD_BOTTOM

const allPoints = computed(() => props.series.flatMap((s) => s.points))
const allRates = computed(() => allPoints.value.map((p) => p.rate))
const minRate = computed(() => (allRates.value.length ? Math.min(...allRates.value) : 0))
const maxRate = computed(() => (allRates.value.length ? Math.max(...allRates.value) : 1))
const range = computed(() => Math.max(maxRate.value - minRate.value, 0.25))

const maxLen = computed(() => Math.max(1, ...props.series.map((s) => s.points.length)))

function x(i: number): number {
  if (maxLen.value <= 1) return WIDTH / 2
  return PAD_X + (i / (maxLen.value - 1)) * (WIDTH - PAD_X * 2)
}
function y(rate: number): number {
  const t = (rate - minRate.value) / range.value
  return axisY - t * (axisY - PAD_TOP)
}
function linePoints(points: { date: string; rate: number }[]): string {
  return points.map((p, i) => `${x(i)},${y(p.rate)}`).join(' ')
}

// x-axis labels sampled from whichever series has the most points.
const axisSource = computed(() => props.series.find((s) => s.points.length === maxLen.value)?.points ?? [])
const axisLabels = computed(() => {
  const pts = axisSource.value
  if (pts.length === 0) return []
  const count = Math.min(6, pts.length)
  const step = (pts.length - 1) / Math.max(count - 1, 1)
  return Array.from({ length: count }, (_, i) => {
    const idx = Math.round(i * step)
    return { x: x(idx), label: fmtShort(pts[idx].date) }
  })
})
function fmtShort(iso: string): string {
  return new Date(iso).toLocaleDateString('en-LK', { month: 'short' })
}
</script>

<template>
  <div class="w-full">
    <svg :viewBox="`0 0 ${WIDTH} ${HEIGHT}`" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Rate trends by bank" class="block w-full">
      <line
        v-for="frac in [0.25, 0.5, 0.75]"
        :key="frac"
        :x1="PAD_X"
        :y1="PAD_TOP + frac * (axisY - PAD_TOP)"
        :x2="WIDTH - PAD_X"
        :y2="PAD_TOP + frac * (axisY - PAD_TOP)"
        stroke="var(--card-border, #E2E8F0)"
        stroke-dasharray="3 3"
      />
      <line :x1="PAD_X" :y1="axisY" :x2="WIDTH - PAD_X" :y2="axisY" stroke="var(--card-border, #E2E8F0)" />
      <text :x="4" :y="PAD_TOP - 8" font-size="11" fill="#64748B">{{ maxRate.toFixed(2) }}%</text>
      <text :x="4" :y="axisY - 6" font-size="11" fill="#64748B">{{ minRate.toFixed(2) }}%</text>

      <template v-for="s in series" :key="s.label">
        <polyline :points="linePoints(s.points)" fill="none" :stroke="s.color" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round" />
        <circle v-for="(p, i) in s.points" :key="i" :cx="x(i)" :cy="y(p.rate)" r="3" :fill="s.color">
          <title>{{ s.label }}: {{ p.rate.toFixed(2) }}% on {{ fmtDate(p.date) }}</title>
        </circle>
      </template>

      <text v-for="a in axisLabels" :key="a.label + a.x" :x="a.x" :y="HEIGHT - 6" font-size="10" fill="#64748B" text-anchor="middle">{{ a.label }}</text>
    </svg>
  </div>
</template>
