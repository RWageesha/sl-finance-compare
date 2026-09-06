<script setup lang="ts">
defineProps<{
  bankName: string
  bankSlug?: string
  productName: string
  lastUpdated: string
  rate: string
  paymentFrequency?: string
}>()
</script>

<template>
  <section class="flex flex-col gap-5 rounded-[16px] bg-navy p-5 sm:flex-row sm:items-start sm:justify-between sm:p-7">
    <div class="min-w-0">
      <div class="mb-3 flex items-center gap-2.5">
        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/20">
          <Icon name="bank" class="h-4 w-4 text-[#7fa8ff]" />
        </span>
        <NuxtLink v-if="bankSlug" :to="`/banks/${bankSlug}`" class="text-[13px] font-bold text-[#7fa8ff] hover:underline">{{ bankName }}</NuxtLink>
        <span v-else class="text-[13px] font-bold text-[#7fa8ff]">{{ bankName }}</span>
      </div>
      <h1 class="text-[22px] font-bold text-white sm:text-[24px]">{{ productName }}</h1>
      <p class="mt-1.5 text-xs text-[#94a3b8]">Last updated: {{ lastUpdated }}</p>
    </div>
    <div class="shrink-0 text-right">
      <p class="text-[10px] font-bold uppercase tracking-wider text-[#94a3b8]">Annual Interest Rate</p>
      <p class="text-[36px] font-bold leading-none text-[#7fa8ff] sm:text-[40px]">
        {{ rate }}<span class="ml-1 text-base font-normal text-[#cbd5e1]">p.a.</span>
      </p>
      <p v-if="paymentFrequency" class="mt-1.5 text-xs text-[#94a3b8]">{{ paymentFrequency }}</p>
    </div>
  </section>
</template>
