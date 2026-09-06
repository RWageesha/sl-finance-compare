<script setup lang="ts">
defineProps<{
  switcherLabel: string
  heading: string
  subtitle: string
}>()
</script>

<template>
  <div>
    <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted sm:text-sm" aria-label="Breadcrumb">
      <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
      <span>&rsaquo;</span>
      <NuxtLink to="/#comparisons" class="hover:text-primary">Compare</NuxtLink>
      <span>&rsaquo;</span>
      <span class="font-semibold text-primary">{{ switcherLabel }}</span>
    </nav>
    <h1 class="text-2xl font-extrabold text-navy sm:text-[1.75rem]">{{ heading }}</h1>
    <p class="mt-1.5 max-w-2xl text-sm text-muted">{{ subtitle }}</p>
  </div>
</template>
