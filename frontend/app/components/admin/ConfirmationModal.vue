<script setup lang="ts">
withDefaults(defineProps<{ message?: string; confirmLabel?: string }>(), {
  message: 'This action cannot be undone.',
  confirmLabel: 'Confirm'
})
const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
    <div class="w-full max-w-sm rounded-card bg-card p-6 text-center shadow-xl">
      <p class="text-lg font-bold text-navy">Are you sure?</p>
      <p class="mt-1.5 text-sm text-muted">{{ message }}</p>
      <div class="mt-5 flex justify-center gap-3">
        <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold text-navy hover:border-primary" @click="emit('cancel')">
          Cancel
        </button>
        <button type="button" class="rounded-lg bg-red-600 px-4 py-2 text-sm font-bold text-white hover:bg-red-700" @click="emit('confirm')">
          {{ confirmLabel }}
        </button>
      </div>
    </div>
  </div>
</template>
