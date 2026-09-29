<script setup lang="ts">
// A generic ad placement — fills whatever box its call site sizes it to
// (skyscraper rail, homepage banner, footer strip, etc. all use the same
// component with different wrapper CSS). Renders nothing when the slot
// has no active ads, so an empty slot never leaves a visible gap. When a
// slot has more than one active ad, rotates between them on a timer —
// this is the "several ads share one space" / "sliding advertisement"
// behavior from the feature request, with no carousel library needed.
import type { SiteAd } from '~/composables/useSiteContent'

const props = defineProps<{ slotKey: string }>()

const { fetchAds } = useSiteContent()
const ads = ref<SiteAd[]>([])
const current = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

const ROTATE_MS = 6000

onMounted(async () => {
  ads.value = await fetchAds(props.slotKey).catch(() => [])
  if (ads.value.length > 1 && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    timer = setInterval(() => {
      current.value = (current.value + 1) % ads.value.length
    }, ROTATE_MS)
  }
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const activeAd = computed(() => ads.value[current.value] ?? null)
</script>

<template>
  <a
    v-if="activeAd"
    :href="activeAd.target_url"
    target="_blank"
    rel="noopener sponsored"
    class="ad-slot"
    :aria-label="activeAd.title"
  >
    <span class="ad-tag">Advertisement</span>
    <img :src="activeAd.image_url" :alt="activeAd.title" class="ad-img">
  </a>
</template>

<style scoped>
/* No width/height here on purpose — every call site passes its own
   sizing classes (h-24, w-40, etc.) straight onto <AdSlot>, and a
   fixed width/height in scoped CSS would win the cascade over those
   (Vue's data-v-* attribute selector makes scoped rules more specific
   than a plain Tailwind utility class) and silently override them. */
.ad-slot {
  position: relative;
  overflow: hidden;
  border-radius: 12px;
  background: var(--page, #f8fafc);
}
.ad-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.ad-tag {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 1;
  padding: 1px 7px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.65);
  color: #fff;
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
