<script setup lang="ts">
// Shared by ProductResultList and CompareResultList. Real result sets can
// run to dozens of pages now (e.g. 110 fixed deposits / 4 per page = 28
// pages) — rendering one chip per page in an unwrapped row overflowed the
// page horizontally past a few hundred px of width. Compressed to first,
// last, current ± 1, with an ellipsis filling any gap, the same pattern
// pages/banks/[slug].vue already uses for its own pagination — at most 7
// chips ever show, which fits at any width, so there's no separate mobile
// fallback needed.
const props = defineProps<{
  modelValue: number
  pageCount?: number
}>()
const emit = defineEmits<{ 'update:modelValue': [number] }>()

const total = computed(() => props.pageCount ?? 3)

function go(p: number) {
  emit('update:modelValue', Math.min(Math.max(1, p), total.value))
}

const pageNumbers = computed<(number | 'ellipsis')[]>(() => {
  const current = props.modelValue
  if (total.value <= 7) return Array.from({ length: total.value }, (_, i) => i + 1)
  const pages: (number | 'ellipsis')[] = [1]
  if (current > 3) pages.push('ellipsis')
  for (let p = Math.max(2, current - 1); p <= Math.min(total.value - 1, current + 1); p++) pages.push(p)
  if (current < total.value - 2) pages.push('ellipsis')
  pages.push(total.value)
  return pages
})
</script>

<template>
  <div class="mt-6 flex flex-wrap items-center justify-center gap-1.5">
    <button
      type="button"
      class="px-2 text-sm font-semibold text-muted transition hover:text-primary disabled:cursor-not-allowed disabled:opacity-40"
      :disabled="modelValue === 1"
      @click="go(modelValue - 1)"
    >
      Previous
    </button>

    <template v-for="(p, i) in pageNumbers" :key="i">
      <span v-if="p === 'ellipsis'" class="px-1 text-sm text-muted">&hellip;</span>
      <button
        v-else
        type="button"
        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-sm font-semibold transition"
        :class="p === modelValue ? 'bg-primary text-white' : 'border border-card-border text-navy hover:border-primary'"
        @click="go(p)"
      >
        {{ p }}
      </button>
    </template>

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
