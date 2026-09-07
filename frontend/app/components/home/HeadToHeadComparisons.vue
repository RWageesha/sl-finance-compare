<script setup lang="ts">
const COMPARISONS = [
  { title: 'Fixed Deposit Rates', desc: 'Evaluation of HNB vs Commercial Bank 12-Month deposits.', href: '/compare/fixed-deposits', image: '/hero/fixed-deposit-card.jpg' },
  { title: 'Housing Loan Rates', desc: 'Base lending margins and administrative fees compared.', href: '/compare/housing-loans', image: '/hero/housing-loan-card.jpg' },
  { title: 'Savings Account Rates', desc: 'Savings multipliers and cash withdrawal limits contrasted.', href: '/compare/savings-accounts', image: '/hero/savings-account-card.jpg' }
]
</script>

<template>
  <section class="bg-page px-4 py-7 sm:px-6 sm:py-10">
    <div class="mx-auto max-w-[1080px]">
      <span class="inline-block rounded-pill bg-badge-bg px-3 py-1 text-xs font-bold uppercase tracking-wider text-primary">
        Trending Comparisons
      </span>
      <h2 class="mt-3 text-[1.3rem] font-bold text-navy sm:text-[1.6rem]">Popular Head-to-Head Comparisons</h2>
      <p class="mt-1 text-sm text-muted">Side-by-side analysis frequently conducted by domestic consumers.</p>

      <div class="mt-8 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div
          v-for="c in COMPARISONS"
          :key="c.title"
          class="overflow-hidden rounded-card border border-card-border bg-card shadow-sm"
        >
          <div class="h-[120px]">
            <img :src="c.image" alt="" class="h-full w-full object-cover">
          </div>
          <div class="p-[18px]">
            <h3 class="text-base font-bold text-navy">{{ c.title }}</h3>
            <NuxtLink :to="c.href" class="mt-2 inline-block text-sm font-semibold text-primary hover:underline">
              Compare Rates &rarr;
            </NuxtLink>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
