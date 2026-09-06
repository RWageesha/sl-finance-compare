<script setup lang="ts">
// Shared by ProductResultList and CompareResultList. Cosmetic for now —
// see each caller's note on why (hardcoded placeholder datasets have no
// real page 2 behind them yet).
const props = defineProps<{
  modelValue: number
  pageCount?: number
}>()
const emit = defineEmits<{ 'update:modelValue': [number] }>()

const total = computed(() => props.pageCount ?? 3)
const chips = computed(() => Array.from({ length: total.value }, (_, i) => i + 1))

function go(p: number) {
  emit('update:modelValue', Math.min(Math.max(1, p), total.value))
}
</script>

<template>
  <div class="mt-6 flex items-center justify-center gap-1.5">
    <button
      type="button"
      class="px-2 text-sm font-semibold text-muted transition hover:text-primary disabled:cursor-not-allowed disabled:opacity-40"
      :disabled="modelValue === 1"
      @click="go(modelValue - 1)"
    >
      Previous
    </button>

    <!-- Desktop/tablet: numbered chips -->
    <div class="hidden items-center gap-1.5 sm:flex">
      <button
        v-for="p in chips"
        :key="p"
        type="button"
        class="flex h-8 w-8 items-center justify-center rounded-lg text-sm font-semibold transition"
        :class="p === modelValue ? 'bg-primary text-white' : 'border border-card-border text-navy hover:border-primary'"
        @click="go(p)"
      >
        {{ p }}
      </button>
    </div>

    <!-- Mobile: numbered chips don't fit cleanly — just say which page -->
    <span class="text-sm font-semibold text-navy sm:hidden">Page {{ modelValue }} of {{ total }}</span>

    <button
      type="button"
      class="px-2 text-sm font-semibold text-primary transition hover:underline disabled:cursor-not-allowed disabled:text-muted disabled:no-underline disabled:opacity-40"
      :disabled="modelValue === total"
      @click="go(modelValue + 1)"
    >
      Next
    </button>
  </div>
</template>
