<script setup lang="ts">
defineProps<{
  sourceLabel?: string
  sourceUrl?: string
  lastVerified: string
}>()
</script>

<template>
  <section class="rounded-[14px] border border-card-border bg-card p-5 shadow-sm">
    <h2 class="mb-3 text-base font-bold text-navy">Source &amp; Verification</h2>
    <p class="text-[10px] font-semibold uppercase tracking-wide text-muted">Data Source</p>
    <p v-if="sourceLabel" class="mb-0.5 text-[13px] font-bold text-navy">{{ sourceLabel }}</p>
    <a v-if="sourceUrl" :href="sourceUrl" target="_blank" rel="noopener" class="break-all text-xs font-semibold text-primary hover:underline">{{ sourceUrl }}</a>
    <p class="mt-3 text-[10px] font-semibold uppercase tracking-wide text-muted">Last Verification Date</p>
    <p class="text-[13px] font-bold text-navy">{{ lastVerified }}</p>
  </section>
</template>
