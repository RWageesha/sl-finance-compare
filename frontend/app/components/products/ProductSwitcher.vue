<script setup lang="ts">
import { PRODUCT_TYPE_ORDER, PRODUCT_TYPES } from '~/config/productTypes'

defineProps<{
  active: string
}>()

const PILLS = PRODUCT_TYPE_ORDER.map((slug) => ({ slug, label: PRODUCT_TYPES[slug].switcherLabel }))
</script>

<template>
  <nav class="my-5 flex flex-wrap gap-2" aria-label="Product type">
    <NuxtLink
      v-for="pill in PILLS"
      :key="pill.slug"
      :to="`/products/${pill.slug}`"
      class="rounded-pill px-5 py-2.5 text-sm font-semibold transition"
      :class="active === pill.slug ? 'bg-primary text-white' : 'border border-card-border bg-white text-navy hover:border-primary'"
    >
      {{ pill.label }}
    </NuxtLink>
  </nav>
</template>
