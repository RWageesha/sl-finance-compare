<script setup lang="ts">
withDefaults(defineProps<{ message?: string; showCancel?: boolean }>(), { message: 'Please wait while we fetch the data.', showCancel: false })
const emit = defineEmits<{ cancel: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center rounded-card border border-card-border bg-card p-10 text-center">
    <svg class="h-8 w-8 animate-spin text-primary" viewBox="0 0 24 24" fill="none">
      <path d="M21 12a9 9 0 1 1-9-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" />
    </svg>
    <p class="mt-3 text-base font-bold text-navy">Loading...</p>
    <p class="mt-1 text-sm text-muted">{{ message }}</p>
    <button v-if="showCancel" type="button" class="mt-4 rounded-lg border border-card-border px-4 py-2 text-sm font-semibold text-navy hover:border-primary" @click="emit('cancel')">
      Cancel
    </button>
  </div>
</template>
