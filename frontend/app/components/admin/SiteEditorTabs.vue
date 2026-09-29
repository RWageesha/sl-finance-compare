<script setup lang="ts">
// Shared sub-nav for the four Site Editor pages (AdminShell's "Site
// Editor" nav item links here) — one flat array, same pattern as
// AdminShell's own NAV.
defineProps<{ active: 'ads' | 'sections' | 'bank-logos' | 'pages' }>()

const TABS = [
  { key: 'ads', label: 'Ads', to: '/admin/site-editor/ads' },
  { key: 'sections', label: 'Homepage Sections', to: '/admin/site-editor/sections' },
  { key: 'bank-logos', label: 'Bank Logos', to: '/admin/site-editor/bank-logos' },
  { key: 'pages', label: 'Page Text', to: '/admin/site-editor/pages' }
] as const
</script>

<template>
  <div class="mb-5 flex flex-wrap gap-1.5 border-b border-card-border">
    <NuxtLink
      v-for="t in TABS"
      :key="t.key"
      :to="t.to"
      class="rounded-t-lg px-3.5 py-2 text-sm font-semibold transition"
      :class="active === t.key ? 'border-b-2 border-primary text-primary' : 'text-muted hover:text-navy'"
    >
      {{ t.label }}
    </NuxtLink>
  </div>
</template>
