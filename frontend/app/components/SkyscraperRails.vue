<script setup lang="ts">
// Vertical ad rails in the side gutters — every content page centers its
// content in a <= 1200px column (see pages/*/index.vue, AppHeader/
// AppFooter). Deliberately `absolute`, not `fixed`: these sit right
// after the hero/header (top-36) and scroll away with the rest of the
// page past that point, like a normal part of the page layout, instead
// of staying pinned to the viewport as the visitor scrolls further down
// (which looked like a banner stuck on top of later sections).
//
// Each rail needs an offset + width of real gutter per side. Using a
// 120px rail (the IAB "Skyscraper" standard size, narrower than the
// 160px "Wide Skyscraper" tried first) at a 12px offset needs 132px of
// gutter, which exists once the viewport is at least 1200 + 2*132 =
// 1464px — 1500px is used as a clean, safe round number above that.
// (1600px/160px-wide was tried first but excluded common scaled desktop
// widths — e.g. a 1920px display at Windows' own 125% scaling reports a
// 1536px CSS viewport, narrower than that.)
//
// Mounted once here so every page gets it for free instead of repeating
// the wrapper per page. Each rail resolves the current page's own
// skyscraper slot (migrations/009_page_skyscrapers.sql) plus the
// site-wide one, so an admin can target a specific page or fall back to
// "every page" — same choice every other ad placement already has.
const route = useRoute()
const pageKey = computed(() => resolvePageKey(route.path))
const leftKeys = computed(() => [...(pageKey.value ? [`${pageKey.value}-skyscraper-left`] : []), 'skyscraper-left'])
const rightKeys = computed(() => [...(pageKey.value ? [`${pageKey.value}-skyscraper-right`] : []), 'skyscraper-right'])
</script>

<template>
  <div class="pointer-events-none z-30 hidden min-[1500px]:block">
    <AdSlot :slot-key="leftKeys" class="pointer-events-auto absolute left-3 top-36 block h-[600px] w-[120px]" />
    <AdSlot :slot-key="rightKeys" class="pointer-events-auto absolute right-3 top-36 block h-[600px] w-[120px]" />
  </div>
</template>
