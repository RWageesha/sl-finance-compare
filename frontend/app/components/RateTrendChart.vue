<script setup lang="ts">
// Dependency-free SVG line chart for a single product's rate history.
// Deliberately not pulling in a charting library for what's a handful of
// points on a simple line — an inline SVG keeps the bundle small and the
// rendering fully predictable (no dark-mode theming quirks from a
// third-party canvas renderer either).
const props = defineProps<{
  points: { date: string; rate: number }[]
}>()

const WIDTH = 640
const HEIGHT = 220
const PAD_X = 44
const PAD_Y = 24

const rates = computed(() => props.points.map((p) => p.rate))
const minRate = computed(() => Math.min(...rates.value))
const maxRate = computed(() => Math.max(...rates.value))
// Give a flat series (all points equal) some visual headroom instead of a
// zero-height range collapsing the line onto one edge.
const range = computed(() => Math.max(maxRate.value - minRate.value, 0.25))

function x(i: number): number {
  const n = props.points.length
  if (n <= 1) return WIDTH / 2
  return PAD_X + (i / (n - 1)) * (WIDTH - PAD_X * 2)
}

function y(rate: number): number {
  const t = (rate - minRate.value) / range.value
  return HEIGHT - PAD_Y - t * (HEIGHT - PAD_Y * 2)
}

const linePoints = computed(() => props.points.map((p, i) => `${x(i)},${y(p.rate)}`).join(' '))

const firstLabel = computed(() => (props.points[0] ? fmtDate(props.points[0].date) : ''))
const lastLabel = computed(() => {
  const last = props.points[props.points.length - 1]
  return last ? fmtDate(last.date) : ''
})
</script>

<template>
  <div class="chart">
    <svg :viewBox="`0 0 ${WIDTH} ${HEIGHT}`" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Rate history trend chart">
      <line :x1="PAD_X" :y1="HEIGHT - PAD_Y" :x2="WIDTH - PAD_X" :y2="HEIGHT - PAD_Y" class="axis" />
      <text :x="PAD_X" :y="PAD_Y - 8" class="axis-label">{{ maxRate.toFixed(2) }}%</text>
      <text :x="PAD_X" :y="HEIGHT - PAD_Y + 16" class="axis-label">{{ minRate.toFixed(2) }}%</text>

      <polyline :points="linePoints" class="line" fill="none" />
      <circle v-for="(p, i) in points" :key="i" :cx="x(i)" :cy="y(p.rate)" r="3.5" class="dot" />

      <text :x="PAD_X" :y="HEIGHT - 4" class="date-label" text-anchor="start">{{ firstLabel }}</text>
      <text :x="WIDTH - PAD_X" :y="HEIGHT - 4" class="date-label" text-anchor="end">{{ lastLabel }}</text>
    </svg>
  </div>
</template>

<style scoped>
.chart {
  width: 100%;
}
svg {
  width: 100%;
  height: auto;
  display: block;
}
.axis {
  stroke: var(--border);
  stroke-width: 1;
}
.axis-label {
  fill: var(--muted);
  font-size: 11px;
}
.date-label {
  fill: var(--muted);
  font-size: 11px;
}
.line {
  stroke: var(--accent);
  stroke-width: 2.5;
  stroke-linejoin: round;
  stroke-linecap: round;
}
.dot {
  fill: var(--panel);
  stroke: var(--accent);
  stroke-width: 2;
}
</style>
