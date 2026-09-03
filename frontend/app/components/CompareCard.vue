<script setup lang="ts">
defineProps<{
  label: string
  desc: string
  href: string
  chart: 'bar-chart' | 'line-chart' | 'bar-chart-alt'
  dark?: boolean
}>()
</script>

<template>
  <div class="compare-card">
    <div class="thumb" :class="{ dark }">
      <Icon :name="chart" />
    </div>
    <h3>{{ label }}</h3>
    <p>{{ desc }}</p>
    <NuxtLink :to="href">Compare Rates &rarr;</NuxtLink>
  </div>
</template>

<style scoped>
.compare-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.2rem;
}
.compare-card .thumb {
  height: 90px;
  border-radius: 8px;
  margin-bottom: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--bg), var(--border));
  color: var(--accent);
}
.compare-card .thumb.dark {
  background: linear-gradient(135deg, var(--navy-dark), var(--navy));
  color: #7fa8ff;
}
.compare-card .thumb svg {
  width: 64px;
  height: 64px;
  opacity: 0.85;
}
.compare-card h3 {
  margin: 0 0 0.35rem;
  font-size: 1rem;
  font-weight: 700;
}
.compare-card p {
  margin: 0 0 0.8rem;
  font-size: 0.83rem;
  color: var(--muted);
  line-height: 1.45;
}
.compare-card a {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
}
.compare-card a:hover {
  text-decoration: underline;
}
</style>
