<script setup lang="ts">
// Vertical ad rails in the side gutters — every content page centers its
// content in a <= 1200px column (see pages/*/index.vue, AppHeader/
// AppFooter). Deliberately `absolute`, not `fixed`: these sit right
// after the hero/header and scroll away with the rest of the page past
// that point, like a normal part of the page layout, instead of staying
// pinned to the viewport as the visitor scrolls further down (which
// looked like a banner stuck on top of later sections).
//
// The top offset is measured at runtime, not a hardcoded guess — the
// homepage's hero (#hero-section, HeroSection.vue) is far taller than
// any other page's header/breadcrumb area and its height is itself
// responsive, so a fixed pixel value was either overlapping the hero on
// home or leaving a gap on every other page. Re-measured on resize since
// the hero reflows at each breakpoint.
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

// Fallback for pages with no #hero-section (everything except home) —
// just past the header bar, matching every other page's own breadcrumb
// spacing.
const FALLBACK_TOP = 96
const topPx = ref(FALLBACK_TOP)

function measureTop() {
  const hero = document.getElementById('hero-section')
  topPx.value = hero ? Math.round(hero.getBoundingClientRect().bottom + window.scrollY) : FALLBACK_TOP
}

let resizeTimer: ReturnType<typeof setTimeout> | null = null
function onResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(measureTop, 150)
}

onMounted(() => {
  measureTop()
  window.addEventListener('resize', onResize)
})
onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  if (resizeTimer) clearTimeout(resizeTimer)
})
// Route changes swap the page (and whether #hero-section exists at all)
// without a full reload, so re-measure after the new page has rendered.
watch(() => route.path, () => nextTick(measureTop))
</script>

<template>
  <div class="pointer-events-none z-30 hidden min-[1500px]:block">
    <AdSlot :slot-key="leftKeys" class="pointer-events-auto absolute left-3 block h-[600px] w-[120px]" :style="{ top: `${topPx}px` }" />
    <AdSlot :slot-key="rightKeys" class="pointer-events-auto absolute right-3 block h-[600px] w-[120px]" :style="{ top: `${topPx}px` }" />
  </div>
</template>
