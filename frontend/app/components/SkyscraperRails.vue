<script setup lang="ts">
// Fixed-position vertical ad rails in the side gutters — every content
// page centers its content in a <= 1200px column (see pages/*/index.vue,
// AppHeader/AppFooter), so at 3xl (1760px, see tailwind.config.ts) and up
// there's real, unused space on both sides. Mounted once here so every
// page gets it for free instead of repeating the wrapper per page.
// Hidden below 3xl, where there's no real gutter left to use.
</script>

<template>
  <div class="pointer-events-none fixed inset-y-0 left-0 right-0 z-30 hidden 3xl:block">
    <AdSlot slot-key="skyscraper-left" class="pointer-events-auto absolute left-6 top-36 block h-[600px] w-40" />
    <AdSlot slot-key="skyscraper-right" class="pointer-events-auto absolute right-6 top-36 block h-[600px] w-40" />
  </div>
</template>
