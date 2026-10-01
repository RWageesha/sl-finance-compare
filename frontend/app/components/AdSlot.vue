<script setup lang="ts">
// A generic ad placement. Renders nothing when the slot has no
// qualifying ad, so an empty slot never leaves a visible gap.
//
// Size is per-ad, not per-slot: every ad carries its own real IAB
// standard width/height (settings.width/height, chosen in the wizard's
// Format step — see pages/admin/site-editor/ads.vue's SIZE_OPTIONS), so
// the same physical spot (e.g. the skyscraper rail, or a page's banner
// strip) can host a 728x90 Leaderboard, a 970x250 Billboard, or anything
// else of matching orientation — the call site only controls POSITION
// (absolute/top/left, or mx-auto/block), never the size.
//
// One ad is chosen per page load (weighted-random among every ad
// currently placed on this slot, filtered to ads allowed on this
// device) — this is the Ad Manager wizard's own rotation rule: several
// ads sharing a slot rotate across page loads, weighted by `weight`,
// not in a live in-page carousel. A single ad's own multiple creatives
// (the Slider style) DO rotate live within that one ad, via
// `activeCreative` below.
import type { SiteAd } from '~/composables/useSiteContent'

// slotKey accepts several keys (e.g. a page-specific skyscraper slot
// plus the site-wide fallback) — candidates from every given slot are
// pooled together before the one weighted pick below, so a page-specific
// ad and a site-wide one can compete for the same physical rail.
//
// rail: true for fixed-position side rails. A wide ad (e.g. a 300px
// Half Page) needs more gutter space than a narrow one (120px
// Skyscraper) to clear the page's own content column without
// overlapping it — rather than hiding whenever the exact gutter doesn't
// fit, the rail scales itself DOWN to whatever gutter space actually
// exists (same idea as a horizontal banner's aspect-ratio shrink), so a
// Half Page ad still shows — smaller — on a 1536px laptop instead of
// vanishing until the visitor zooms out. It only hides outright once
// there's no meaningful gutter left at all (checked per the ad actually
// chosen, not a fixed breakpoint, since which ad wins the weighted pick
// can vary). railAlign tells it which corner to shrink away from, so it
// always retreats from the content edge, never the screen edge.
const props = withDefaults(
  defineProps<{ slotKey: string | string[]; rail?: boolean; railOffset?: number; railAlign?: 'left' | 'right' }>(),
  { rail: false, railOffset: 12, railAlign: 'left' }
)

const { fetchAds } = useSiteContent()
const ad = ref<SiteAd | null>(null)
const current = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

function deviceMatches(devices: string): boolean {
  if (devices === 'all') return true
  const isMobile = window.matchMedia('(max-width: 767px)').matches
  return devices === 'mobile' ? isMobile : !isMobile
}

function weightedPick(ads: SiteAd[]): SiteAd {
  const total = ads.reduce((sum, a) => sum + (a.weight || 1), 0)
  if (total <= 0) return ads[0]
  let r = Math.random() * total
  for (const a of ads) {
    r -= a.weight || 1
    if (r <= 0) return a
  }
  return ads[ads.length - 1]
}

// Fallback for ads saved before per-ad sizing existed (settings has no
// width/height) — keeps them rendering at a sensible default instead of
// collapsing to 0.
const DEFAULT_SIZE = { h: { w: 728, h: 90 }, v: { w: 120, h: 600 } }
const resolvedSize = computed(() => {
  const layoutKey = ad.value?.layout === 'vertical' ? 'v' : 'h'
  const w = Number(ad.value?.settings?.width) || DEFAULT_SIZE[layoutKey].w
  const h = Number(ad.value?.settings?.height) || DEFAULT_SIZE[layoutKey].h
  return { w, h }
})

// Below this many px of actual gutter, even a scaled-down rail would be
// an illegible sliver — hide instead, same as the old hard cutoff did
// for every size once the viewport got narrow enough (e.g. tablet/mobile).
const MIN_RAIL_GUTTER = 30
const railFit = ref({ show: true, scale: 1 })
function checkFit() {
  if (!props.rail) return
  const { w } = resolvedSize.value
  const contentColumn = 1200
  const gutter = (window.innerWidth - contentColumn) / 2 - props.railOffset
  railFit.value = gutter < MIN_RAIL_GUTTER ? { show: false, scale: 1 } : { show: true, scale: Math.min(1, gutter / w) }
}
let resizeTimer: ReturnType<typeof setTimeout> | null = null
function onResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(checkFit, 150)
}

onMounted(async () => {
  const keys = Array.isArray(props.slotKey) ? props.slotKey : [props.slotKey]
  const results = await Promise.all(keys.map((k) => fetchAds(k).catch(() => [])))
  const all = [...new Map(results.flat().map((a) => [a.id, a])).values()]
  const candidates = all.filter((a) => deviceMatches(a.devices) && a.creatives.length > 0)
  window.addEventListener('resize', onResize)
  if (candidates.length === 0) return
  ad.value = weightedPick(candidates)
  checkFit()
  if (ad.value.style === 'slider' && ad.value.creatives.length > 1 && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    const intervalSeconds = Number(ad.value.settings?.interval_seconds) || 5
    timer = setInterval(() => {
      current.value = (current.value + 1) % (ad.value?.creatives.length || 1)
    }, intervalSeconds * 1000)
  }
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
  window.removeEventListener('resize', onResize)
  if (resizeTimer) clearTimeout(resizeTimer)
})

const gapPx = computed(() => Number(ad.value?.settings?.gap_px) || 12)
const activeCreative = computed(() => ad.value?.creatives[current.value] ?? null)
const shouldShow = computed(() => !!ad.value && (!props.rail || railFit.value.show))
const sizeStyle = computed(() => {
  const { w, h } = resolvedSize.value
  if (!props.rail) return { width: '100%', maxWidth: `${w}px`, aspectRatio: `${w} / ${h}` }
  return {
    width: `${w}px`,
    height: `${h}px`,
    transform: railFit.value.scale < 1 ? `scale(${railFit.value.scale})` : undefined,
    transformOrigin: props.railAlign === 'right' ? 'top right' : 'top left'
  }
})
</script>

<template>
  <div v-if="shouldShow" class="ad-slot" :style="sizeStyle">
    <div class="ad-frame">
      <span class="ad-tag">Advertisement</span>

      <a
        v-if="(ad.style === 'image' || ad.style === 'gif' || ad.style === 'slider') && activeCreative"
        :href="activeCreative.target_url"
        target="_blank"
        rel="noopener sponsored"
        class="ad-fill"
        :aria-label="activeCreative.alt_text || ad.title"
      >
        <img :src="activeCreative.media_url" :alt="activeCreative.alt_text || ad.title" class="ad-img">
      </a>

      <a
        v-else-if="ad.style === 'video' && ad.creatives[0]"
        :href="ad.creatives[0].target_url"
        target="_blank"
        rel="noopener sponsored"
        class="ad-fill"
        :aria-label="ad.creatives[0].alt_text || ad.title"
      >
        <video
          :src="ad.creatives[0].media_url"
          :poster="ad.creatives[0].poster_url || undefined"
          class="ad-img"
          autoplay
          muted
          loop
          playsinline
        />
      </a>

      <div v-else-if="ad.style === 'shared'" class="ad-fill flex" :class="ad.layout === 'vertical' ? 'flex-col' : 'flex-row'" :style="{ gap: `${gapPx}px` }">
        <a
          v-for="c in ad.creatives"
          :key="c.id"
          :href="c.target_url"
          target="_blank"
          rel="noopener sponsored"
          class="ad-tile"
          :aria-label="c.alt_text || ad.title"
        >
          <img :src="c.media_url" :alt="c.alt_text || ad.title" class="ad-img">
        </a>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* .ad-slot (the root) declares NO layout CSS at all on purpose — every
   call site passes its own sizing/positioning classes (h-24, w-40,
   absolute, top-36, etc.) straight onto <AdSlot>, and Vue's scoped-style
   attribute selector (.ad-slot[data-v-*]) is more specific than a plain
   Tailwind utility class, so ANY property declared here — width, height,
   position, whatever — would silently win the cascade over what the
   caller passed in (this bit us twice: once for width/height, once for
   position). The one thing every instance genuinely needs regardless of
   caller — a relative box to anchor the "Advertisement" tag and clip the
   creative to rounded corners — lives on .ad-frame instead, a child
   Vue fully owns with no possible class collision from outside. */
.ad-frame {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 12px;
  background: var(--page, #f8fafc);
}
.ad-fill {
  display: block;
  width: 100%;
  height: 100%;
}
.ad-tile {
  flex: 1;
  min-width: 0;
  display: block;
}
.ad-img {
  width: 100%;
  height: 100%;
  /* contain, not cover — call sites now size each slot to a real IAB
     standard (e.g. 728x90 Leaderboard, 970x250 Billboard), so a
     correctly-sized upload should just fill the box exactly; contain
     keeps a slightly-off upload fully visible (letterboxed) instead of
     cropping it, which cover would do silently. */
  object-fit: contain;
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
