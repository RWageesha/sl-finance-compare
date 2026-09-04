<script setup lang="ts">
// Drop-in replacement for the generic <Icon name="bank"/> wherever a bank
// is shown: renders the bank's real logo (public/banks/<slug>.<ext>) when
// we have one, falling back to the generic glyph otherwise. Sized by
// whatever fixed-size container it's placed in (object-fit: contain), so
// it works the same in a 26px avatar or a 56px profile header icon.
defineProps<{
  bank?: { slug: string; icon: 'bank' | 'savings' | 'moon'; displayName: string; logoExt?: 'png' | 'jpeg' | 'svg' } | null
}>()
</script>

<template>
  <img v-if="bank?.logoExt" :src="`/banks/${bank.slug}.${bank.logoExt}`" :alt="`${bank.displayName} logo`" class="bank-logo-img">
  <Icon v-else :name="bank?.icon ?? 'bank'" />
</template>

<style scoped>
.bank-logo-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>
