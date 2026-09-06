<script setup lang="ts">
const emit = defineEmits<{ clear: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center rounded-card border border-card-border bg-card p-10 text-center">
    <span class="text-2xl text-muted">&#128269;</span>
    <p class="mt-2 text-base font-bold text-navy">No results found</p>
    <p class="mt-1 text-sm text-muted">Try adjusting your search or filters.</p>
    <button type="button" class="mt-4 rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="emit('clear')">
      Clear Filters
    </button>
  </div>
</template>
