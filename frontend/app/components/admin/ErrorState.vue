<script setup lang="ts">
withDefaults(defineProps<{ message?: string }>(), { message: 'Something went wrong. Please try again.' })
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center rounded-card border border-red-200 bg-red-50 p-10 text-center">
    <span class="text-2xl font-bold text-red-500">!</span>
    <p class="mt-2 text-base font-bold text-navy">Something went wrong</p>
    <p class="mt-1 text-sm text-muted">{{ message }}</p>
    <button type="button" class="mt-4 rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="emit('retry')">
      Retry
    </button>
  </div>
</template>
