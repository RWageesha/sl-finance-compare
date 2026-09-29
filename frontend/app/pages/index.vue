<script setup lang="ts">
// HeroSection is auto-imported by Nuxt (no local import path to type
// InstanceType<> against), so the ref stays loosely typed — it only needs
// the exposed focus() method, called from AppHeader's search button.
const heroRef = ref<{ focus: () => void } | null>(null)

const { fetchSections } = useSiteContent()
const sections = ref<Record<string, { is_visible: boolean; layout_variant: string; background_image_url: string | null }>>({})
onMounted(async () => {
  const rows = await fetchSections('home').catch(() => [])
  sections.value = Object.fromEntries(rows.map((r) => [r.section_key, r]))
})
// Defaults to visible so the homepage looks normal before the fetch
// resolves (and if it fails) — hiding is something an admin opts into,
// never the fallback state.
function isVisible(key: string) {
  return sections.value[key]?.is_visible ?? true
}
</script>

<template>
  <div>
    <AppHeader @focus-search="heroRef?.focus()" />
    <HeroSection
      ref="heroRef"
      :background-image="sections.hero?.background_image_url ?? null"
      :variant="sections.hero?.layout_variant ?? 'default'"
    />
    <div class="mx-auto max-w-[1200px] px-4 sm:px-6">
      <AdSlot slot-key="home-after-hero" class="my-4 block h-24 sm:h-28" />
    </div>
    <PopularProducts v-if="isVisible('popular_products')" />
    <PopularBanks v-if="isVisible('popular_banks')" />
    <div class="mx-auto max-w-[1200px] px-4 sm:px-6">
      <AdSlot slot-key="home-mid" class="my-4 block h-24 sm:h-28" />
    </div>
    <QuickCalculators v-if="isVisible('quick_calculators')" />
    <HeadToHeadComparisons v-if="isVisible('head_to_head')" />
    <RecentRates v-if="isVisible('recent_rates')" />
    <div class="mx-auto max-w-[1200px] px-4 sm:px-6">
      <AdSlot slot-key="home-pre-footer" class="my-4 block h-24 sm:h-28" />
    </div>
    <TransparencyBanner v-if="isVisible('transparency_banner')" />
    <AppFooter />
  </div>
</template>
