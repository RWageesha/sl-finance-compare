<script setup lang="ts">
defineProps<{
  eyebrow: string
  title: string
  subtitle: string
}>()
</script>

<template>
  <section class="rounded-card bg-badge-bg p-5 sm:p-7">
    <span class="inline-block rounded-pill bg-primary px-3 py-1 text-[11px] font-bold uppercase tracking-wider text-white">
      {{ eyebrow }}
    </span>
    <h1 class="mt-3 text-2xl font-extrabold text-navy sm:text-[1.9rem]">{{ title }}</h1>
    <p class="mt-1.5 max-w-2xl text-sm text-muted">{{ subtitle }}</p>
  </section>
</template>
