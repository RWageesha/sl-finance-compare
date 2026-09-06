<script setup lang="ts">
defineProps<{ items: string[] }>()
</script>

<template>
  <ul class="flex flex-col gap-2.5">
    <li v-for="item in items" :key="item" class="flex items-start gap-2.5 text-[13.5px] leading-snug text-navy">
      <span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-sm bg-primary" />
      <span>{{ item }}</span>
    </li>
  </ul>
</template>
