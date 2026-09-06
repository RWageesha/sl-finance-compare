<script setup lang="ts">
// HeroSection is auto-imported by Nuxt (no local import path to type
// InstanceType<> against), so the ref stays loosely typed — it only needs
// the exposed focus() method, called from AppHeader's search button.
const heroRef = ref<{ focus: () => void } | null>(null)
</script>

<template>
  <div>
    <AppHeader @focus-search="heroRef?.focus()" />
    <HeroSection ref="heroRef" />
    <PopularProducts />
    <PopularBanks />
    <QuickCalculators />
    <HeadToHeadComparisons />
    <RecentRates />
    <TransparencyBanner />
    <AppFooter />
  </div>
</template>
