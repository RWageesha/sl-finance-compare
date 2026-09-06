<script setup lang="ts">
withDefaults(defineProps<{ heading?: string; subtext?: string; actionLabel?: string }>(), {
  heading: 'No data yet',
  subtext: 'Add your first item to get started.'
})
const emit = defineEmits<{ action: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center rounded-card border border-card-border bg-card p-10 text-center">
    <span class="flex h-10 w-10 items-center justify-center rounded-full bg-page text-xl text-muted">
      <slot name="icon">&#8709;</slot>
    </span>
    <p class="mt-3 text-base font-bold text-navy">{{ heading }}</p>
    <p class="mt-1 text-sm text-muted">{{ subtext }}</p>
    <button v-if="actionLabel" type="button" class="mt-4 rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="emit('action')">
      {{ actionLabel }}
    </button>
  </div>
</template>
