<script setup lang="ts">
defineProps<{ items: { label: string; value: string }[] }>()
</script>

<template>
  <section class="rounded-[14px] border border-card-border bg-card p-5 shadow-sm">
    <h2 class="mb-4 text-base font-bold text-navy">Key Information</h2>
    <div class="grid grid-cols-2 gap-4">
      <div v-for="item in items" :key="item.label" class="flex flex-col gap-1">
        <span class="text-[10.5px] font-semibold uppercase tracking-wide text-muted">{{ item.label }}</span>
        <strong class="text-sm font-bold text-navy">{{ item.value }}</strong>
      </div>
    </div>
  </section>
</template>
