<script setup lang="ts">
defineProps<{ items: string[] }>()
</script>

<template>
  <section class="rounded-[14px] border border-card-border bg-card p-5 shadow-sm">
    <h2 class="mb-4 text-base font-bold text-navy">Fees &amp; Conditions</h2>
    <BulletList :items="items" />
  </section>
</template>
