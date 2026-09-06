<script setup lang="ts">
// Dependency-free SVG line chart for a single product's rate history.
// Deliberately not pulling in a charting library for what's a handful of
// points on a simple line — an inline SVG keeps the bundle small and the
// rendering fully predictable (no dark-mode theming quirks from a
// third-party canvas renderer either).
const props = withDefaults(
  defineProps<{
    points: { date: string; rate: number }[]
    /** Taller rendering + a filled area gradient beneath the line, for
     * the full Timeline page's larger chart (the compact one used on
     * Product Detail stays a plain line). */
    large?: boolean
  }>(),
  { large: false }
)

const WIDTH = 640
const HEIGHT = computed(() => (props.large ? 360 : 220))
const PAD_X = 44
const PAD_TOP = 24
// Deliberately taller than PAD_TOP: the axis line sits above this
// margin, with the min-rate label just above it and the date labels
// below it, inside this reserved band — kept apart on purpose so the
// two label rows can never visually collide.
const PAD_BOTTOM = 36
const gradientId = `rtc-fill-${Math.random().toString(36).slice(2)}`

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
  const h = HEIGHT.value
  const t = (rate - minRate.value) / range.value
  return h - PAD_BOTTOM - t * (h - PAD_TOP - PAD_BOTTOM)
}

const axisY = computed(() => HEIGHT.value - PAD_BOTTOM)

const linePoints = computed(() => props.points.map((p, i) => `${x(i)},${y(p.rate)}`).join(' '))
// Line points plus two baseline corners, closing the shape for the fill.
const areaPoints = computed(() => {
  if (props.points.length === 0) return ''
  return `${x(0)},${axisY.value} ${linePoints.value} ${x(props.points.length - 1)},${axisY.value}`
})

const firstLabel = computed(() => (props.points[0] ? fmtDate(props.points[0].date) : ''))
const lastLabel = computed(() => {
  const last = props.points[props.points.length - 1]
  return last ? fmtDate(last.date) : ''
})

function dotTitle(p: { date: string; rate: number }): string {
  return `${p.rate.toFixed(2)}% on ${fmtDate(p.date)}`
}
</script>

<template>
  <div class="chart">
    <svg :viewBox="`0 0 ${WIDTH} ${HEIGHT}`" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Rate history trend chart">
      <defs v-if="large">
        <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.22" />
          <stop offset="100%" stop-color="var(--accent)" stop-opacity="0" />
        </linearGradient>
      </defs>

      <line
        v-for="frac in large ? [0.25, 0.5, 0.75] : []"
        :key="frac"
        :x1="PAD_X"
        :y1="PAD_TOP + frac * (axisY - PAD_TOP)"
        :x2="WIDTH - PAD_X"
        :y2="PAD_TOP + frac * (axisY - PAD_TOP)"
        class="gridline"
      />
      <line :x1="PAD_X" :y1="axisY" :x2="WIDTH - PAD_X" :y2="axisY" class="axis" />
      <text :x="4" :y="PAD_TOP - 8" class="axis-label">{{ maxRate.toFixed(2) }}%</text>
      <text :x="4" :y="axisY - 6" class="axis-label">{{ minRate.toFixed(2) }}%</text>

      <polygon v-if="large" :points="areaPoints" :fill="`url(#${gradientId})`" stroke="none" />
      <polyline :points="linePoints" class="line" fill="none" />
      <circle v-for="(p, i) in points" :key="i" :cx="x(i)" :cy="y(p.rate)" :r="large ? 4 : 3.5" class="dot">
        <title>{{ dotTitle(p) }}</title>
      </circle>

      <text :x="PAD_X" :y="HEIGHT - 6" class="date-label" text-anchor="start">{{ firstLabel }}</text>
      <text :x="WIDTH - PAD_X" :y="HEIGHT - 6" class="date-label" text-anchor="end">{{ lastLabel }}</text>
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
.gridline {
  stroke: var(--border);
  stroke-width: 1;
  stroke-dasharray: 3 3;
  opacity: 0.6;
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
  cursor: pointer;
}
</style>
