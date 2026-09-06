<script setup lang="ts">
defineProps<{
  categoryLabel: string
  compareLabel: string
  compareHref: string
  reportUrl: string
}>()
</script>

<template>
  <section class="mt-6">
    <span class="text-[11px] font-bold uppercase tracking-wider text-muted">Related Products</span>
    <h2 class="mt-1 text-[22px] font-bold text-navy">Compare Similar {{ categoryLabel }}</h2>
    <NuxtLink :to="compareHref" class="mt-4 inline-block rounded-lg bg-primary px-6 py-3 text-sm font-bold text-white transition hover:bg-primary/90">
      {{ compareLabel }}
    </NuxtLink>

    <div class="mt-6 flex flex-wrap items-center justify-between gap-3 border-t border-card-border pt-4">
      <p class="text-sm text-muted">Noticed an inaccuracy in the rates listed above?</p>
      <NuxtLink :to="reportUrl" class="rounded-lg border border-card-border bg-white px-4 py-2 text-[13px] font-bold text-navy transition hover:border-primary hover:text-primary">
        Report Incorrect Information
      </NuxtLink>
    </div>
  </section>
</template>
