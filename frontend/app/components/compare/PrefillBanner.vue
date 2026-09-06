<script setup lang="ts">
defineProps<{
  message: string
  baseHref: string
}>()
</script>

<template>
  <div class="mt-4 flex items-center gap-3 rounded-lg bg-badge-bg px-4 py-3 text-sm">
    <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
    <p class="min-w-0 flex-1 text-navy">
      {{ message }}
      <NuxtLink :to="baseHref" class="ml-1 font-semibold text-primary underline">Clear filters</NuxtLink>
    </p>
    <NuxtLink :to="baseHref" aria-label="Dismiss" class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-muted hover:bg-white hover:text-navy">
      <Icon name="close" class="h-3.5 w-3.5" />
    </NuxtLink>
  </div>
</template>
