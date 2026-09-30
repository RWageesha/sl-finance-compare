<script setup lang="ts">
// Fixed-position vertical ad rails in the side gutters — every content
// page centers its content in a <= 1200px column (see pages/*/index.vue,
// AppHeader/AppFooter). Each rail needs a 24px offset + 160px width =
// 184px of real gutter per side, which only exists once the viewport is
// at least 1200 + 2*184 = 1568px — 1600px is used as a clean, safe round
// number above that. (Tailwind's built-in 3xl, 1760px, was tried first
// but turned out wider than most real desktop browser windows — even a
// maximized 1920x1080 display often reports less than that once you
// subtract the OS taskbar/window chrome — which made the rails
// practically invisible in normal testing, defeating the point.)
// Mounted once here so every page gets it for free instead of repeating
// the wrapper per page.
</script>

<template>
  <div class="pointer-events-none fixed inset-y-0 left-0 right-0 z-30 hidden min-[1600px]:block">
    <AdSlot slot-key="skyscraper-left" class="pointer-events-auto absolute left-6 top-36 block h-[600px] w-40" />
    <AdSlot slot-key="skyscraper-right" class="pointer-events-auto absolute right-6 top-36 block h-[600px] w-40" />
  </div>
</template>
