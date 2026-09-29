<script setup lang="ts">
// Drop-in replacement for the generic <Icon name="bank"/> wherever a bank
// is shown: renders the bank's real logo (public/banks/<slug>.<ext>) when
// we have one, falling back to the generic glyph otherwise. Sized by
// whatever fixed-size container it's placed in (object-fit: contain), so
// it works the same in a 26px avatar or a 56px profile header icon.
const props = defineProps<{
  bank?: { slug: string; icon: 'bank' | 'savings' | 'moon'; displayName: string; logoExt?: 'png' | 'jpeg' | 'svg'; logoSmallExt?: 'png' | 'jpeg' | 'svg' } | null
  /** Use the icon-cropped -small variant (Popular Banks cards, the bank
   * directory grid) when the bank has one; silently falls back to the
   * full logo for banks that don't. */
  small?: boolean
}>()

// Admin-uploaded overrides (Site Editor -> Bank Logos) win over the
// static default file when set — fetched once and shared across every
// BankLogo on the page via useBankLogoOverrides' module-level cache.
const { overrides, ensureLoaded } = useBankLogoOverrides()
onMounted(ensureLoaded)

const src = computed(() => {
  if (!props.bank) return null
  const override = overrides.value[props.bank.slug]
  if (props.small) {
    if (override?.logo_small_url) return override.logo_small_url
    if (override?.logo_url) return override.logo_url
    if (props.bank.logoSmallExt) return `/banks/${props.bank.slug}-small.${props.bank.logoSmallExt}`
    if (props.bank.logoExt) return `/banks/${props.bank.slug}.${props.bank.logoExt}`
    return null
  }
  if (override?.logo_url) return override.logo_url
  if (props.bank.logoExt) return `/banks/${props.bank.slug}.${props.bank.logoExt}`
  return null
})
</script>

<template>
  <img v-if="src" :src="src" :alt="`${bank?.displayName} logo`" class="bank-logo-img">
  <Icon v-else :name="bank?.icon ?? 'bank'" />
</template>

<style scoped>
.bank-logo-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>
