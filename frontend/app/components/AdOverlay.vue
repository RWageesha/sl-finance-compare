<script setup lang="ts">
// Full-page overlay ads — a separate rendering path from AdSlot.vue since
// an overlay covers the whole viewport instead of filling one placed box.
// Mounted once in app.vue. Resolves the current route to a page key,
// fetches that page's dedicated "<page>-overlay" slot plus the sitewide
// one, waits settings.show_after_seconds before appearing, keeps the
// close button hidden for settings.close_after_seconds, and respects
// settings.frequency (every/session/day/once) via local/session storage
// keyed per ad id — exactly the rules from the Ad Manager wizard's
// Placement step for the Overlay style.
import type { SiteAd } from '~/composables/useSiteContent'

const { fetchAds } = useSiteContent()
const route = useRoute()

function resolvePageKey(path: string): string | null {
  if (path === '/') return 'home'
  if (path.startsWith('/banks')) return 'banks'
  if (path.startsWith('/products') || path.startsWith('/compare')) return 'products'
  return null
}

function overlayStorageKey(adId: number) {
  return `adOverlaySeen:${adId}`
}
function shouldShow(ad: SiteAd): boolean {
  const freq = String(ad.settings?.frequency || 'session')
  const key = overlayStorageKey(ad.id)
  if (freq === 'every') return true
  if (freq === 'once') return !localStorage.getItem(key)
  if (freq === 'day') {
    const last = Number(localStorage.getItem(key) || 0)
    return Date.now() - last > 24 * 60 * 60 * 1000
  }
  return !sessionStorage.getItem(key) // 'session'
}
function markShown(ad: SiteAd) {
  const freq = String(ad.settings?.frequency || 'session')
  const key = overlayStorageKey(ad.id)
  if (freq === 'every') return
  if (freq === 'day') localStorage.setItem(key, String(Date.now()))
  else if (freq === 'once') localStorage.setItem(key, '1')
  else sessionStorage.setItem(key, '1')
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

const active = ref<SiteAd | null>(null)
const visible = ref(false)
const closable = ref(false)
let showTimer: ReturnType<typeof setTimeout> | null = null
let closeTimer: ReturnType<typeof setTimeout> | null = null

onMounted(async () => {
  const pageKey = resolvePageKey(route.path)
  const slotKeys = [...(pageKey ? [`${pageKey}-overlay`] : []), 'sitewide-overlay']
  const results = await Promise.all(slotKeys.map((k) => fetchAds(k).catch(() => [])))
  const candidates = results.flat().filter((a) => a.creatives.length > 0 && shouldShow(a))
  if (candidates.length === 0) return

  const ad = weightedPick(candidates)
  active.value = ad
  const delaySeconds = Number(ad.settings?.show_after_seconds) || 0
  const closeAfterSeconds = Number(ad.settings?.close_after_seconds) || 0
  showTimer = setTimeout(() => {
    visible.value = true
    markShown(ad)
    closeTimer = setTimeout(() => { closable.value = true }, closeAfterSeconds * 1000)
  }, delaySeconds * 1000)
})
onUnmounted(() => {
  if (showTimer) clearTimeout(showTimer)
  if (closeTimer) clearTimeout(closeTimer)
})

function close() {
  visible.value = false
}
const creative = computed(() => active.value?.creatives[0] ?? null)
const isVideo = computed(() => /\.(mp4|webm)(\?|$)/i.test(creative.value?.media_url || ''))
</script>

<template>
  <div v-if="visible && creative" class="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 p-4">
    <div class="relative max-h-[85vh] max-w-[90vw] overflow-hidden rounded-xl bg-white shadow-2xl">
      <button
        v-if="closable"
        type="button"
        class="absolute right-2 top-2 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-black/60 text-lg text-white transition hover:bg-black/80"
        aria-label="Close advertisement"
        @click="close"
      >
        &times;
      </button>
      <a :href="creative.target_url" target="_blank" rel="noopener sponsored" class="block">
        <video v-if="isVideo" :src="creative.media_url" class="max-h-[85vh] max-w-[90vw] object-contain" autoplay muted loop playsinline />
        <img v-else :src="creative.media_url" :alt="creative.alt_text || active?.title" class="max-h-[85vh] max-w-[90vw] object-contain">
      </a>
    </div>
  </div>
</template>
