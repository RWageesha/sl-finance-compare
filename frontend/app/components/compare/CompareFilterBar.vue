<script setup lang="ts">
import type { FilterField } from '~/config/productTypes'

const props = defineProps<{
  filters: FilterField[]
  initialValues: Record<string, string>
  buttonLabel: string
  sortBy: string
}>()
const emit = defineEmits<{ apply: [Record<string, string>]; 'update:sortBy': [string] }>()

// Draft values only commit to the parent (and so only recompute the
// results) when the button is clicked — matches "Update Results" being
// a deliberate action, not live-as-you-type.
const draft = reactive<Record<string, string>>({ ...props.initialValues })
watch(
  () => props.initialValues,
  (v) => Object.assign(draft, v),
  { deep: true }
)

const fieldClass = 'w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy md:w-auto'
</script>

<template>
  <div class="rounded-card border border-card-border bg-card p-4 shadow-sm sm:p-5">
    <div class="flex flex-col gap-3 md:flex-row md:flex-wrap md:items-end">
      <template v-for="f in filters" :key="f.key">
        <div class="flex flex-col gap-1">
          <label class="text-xs font-semibold text-muted">{{ f.label }}</label>
          <input v-if="f.type === 'input'" v-model="draft[f.key]" type="text" :placeholder="f.placeholder" :class="fieldClass">
          <select v-else v-model="draft[f.key]" :class="fieldClass">
            <option v-for="opt in f.options" :key="opt" :value="opt">{{ opt }}</option>
          </select>
        </div>
      </template>

      <button
        type="button"
        class="w-full shrink-0 rounded-lg bg-primary px-6 py-2.5 text-sm font-bold text-white transition hover:bg-primary/90 md:ml-auto md:w-auto"
        @click="emit('apply', { ...draft })"
      >
        {{ buttonLabel }}
      </button>
    </div>

    <div class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-card-border pt-4">
      <div class="flex items-center gap-2 text-xs text-muted">
        <span class="font-semibold">Filter Banks:</span>
        <span class="rounded-pill bg-badge-bg px-3 py-1 font-semibold text-primary">All Licensed Banks</span>
      </div>
      <label class="flex items-center gap-2 text-xs text-muted">
        <span class="font-semibold">Sort by:</span>
        <select
          :value="sortBy"
          class="rounded-lg border border-card-border bg-white px-2.5 py-1.5 text-xs text-navy"
          @change="emit('update:sortBy', ($event.target as HTMLSelectElement).value)"
        >
          <option>Highest Rate</option>
        </select>
      </label>
    </div>
  </div>
</template>
