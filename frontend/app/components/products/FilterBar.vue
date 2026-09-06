<script setup lang="ts">
import type { FilterField } from '~/config/productTypes'

const props = defineProps<{
  filters: FilterField[]
  compareLabel: string
  compareHref: string
}>()
const emit = defineEmits<{ 'update:values': [Record<string, string>] }>()

const values = reactive<Record<string, string>>({})
watchEffect(() => {
  for (const f of props.filters) {
    if (!(f.key in values)) values[f.key] = f.type === 'dropdown' ? f.options?.[0] ?? '' : ''
  }
})
watch(values, (v) => emit('update:values', { ...v }), { deep: true, immediate: true })

const fieldClass =
  'w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy placeholder:text-muted md:w-auto'

// Carries the current filter selections into the Compare page's query
// string — e.g. a Tenure of "24 Months" here lands as ?tenure=24 there,
// the same param name pages/compare/[slug].vue already reads.
const liveCompareHref = computed(() => {
  if (props.compareHref === '#') return '#'
  const params = new URLSearchParams()
  if (values.amount) {
    const n = Number(values.amount.replace(/[^0-9]/g, ''))
    if (n > 0) params.set('amount', String(n))
  }
  if (values.tenure && values.tenure !== 'Any') {
    const months = parseTenureLabelToMonths(values.tenure)
    if (months) params.set('tenure', String(months))
  }
  if (values.payment && values.payment !== 'Any') params.set('payment', values.payment)
  const qs = params.toString()
  return qs ? `${props.compareHref}?${qs}` : props.compareHref
})
</script>

<template>
  <div class="flex flex-col gap-3 rounded-card border border-card-border bg-card p-4 shadow-sm md:flex-row md:flex-wrap md:items-center">
    <template v-for="f in filters" :key="f.key">
      <input
        v-if="f.type === 'input'"
        v-model="values[f.key]"
        type="text"
        :placeholder="f.placeholder || f.label"
        :class="fieldClass"
      >
      <select v-else-if="f.type === 'dropdown'" v-model="values[f.key]" :class="fieldClass">
        <option v-for="(opt, i) in f.options" :key="opt" :value="opt">{{ i === 0 ? `${f.label}: ${opt}` : opt }}</option>
      </select>
      <input
        v-else
        v-model="values[f.key]"
        type="text"
        :placeholder="f.placeholder || f.label"
        :class="fieldClass"
      >
    </template>

    <NuxtLink
      :to="liveCompareHref"
      class="w-full shrink-0 rounded-lg bg-primary px-6 py-2.5 text-center text-sm font-bold text-white transition hover:bg-primary/90 md:ml-auto md:w-auto"
    >
      {{ compareLabel }}
    </NuxtLink>
  </div>
</template>
