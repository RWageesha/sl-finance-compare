<script setup lang="ts">
const PRODUCTS = [
  { icon: 'bank' as const, title: 'Fixed Deposits', desc: 'Compare yields across all licensed banks.', href: '/products/fixed-deposits' },
  { icon: 'home' as const, title: 'Housing Loans', desc: 'Find the lowest mortgage interest rates.', href: '/products/housing-loans' },
  { icon: 'savings' as const, title: 'Savings Accounts', desc: 'Daily-compounded yields and benefits.', href: '/products/savings-accounts' },
  { icon: 'card' as const, title: 'Personal Loans', desc: 'Unsecured funding rates comparison.', href: '/products/personal-loans' },
  { icon: 'coin' as const, title: 'Gold Loans', desc: 'Pawning rates based on gold carats.', href: '/products/gold-loans' },
  { icon: 'card' as const, title: 'Credit Cards', desc: 'Annual fees, cash-back, and rewards.', href: '/products/credit-cards' }
]

const cardClass =
  'group flex items-center gap-3.5 rounded-card border border-card-border bg-card p-[18px] shadow-sm transition hover:-translate-y-0.5 hover:border-primary hover:shadow-md'
</script>

<template>
  <section class="bg-page px-4 py-7 sm:px-6 sm:py-10">
    <div class="mx-auto max-w-[1080px]">
      <span class="inline-block rounded-pill bg-badge-bg px-3 py-1 text-xs font-bold uppercase tracking-wider text-primary">
        Products
      </span>
      <h2 class="mt-3 text-[1.3rem] font-bold text-navy sm:text-[1.6rem]">Popular Financial Products</h2>
      <p class="mt-1 text-sm text-muted">Compare current rates across licensed Sri Lankan banks.</p>

      <div class="mt-8 grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3">
        <template v-for="p in PRODUCTS" :key="p.title">
          <NuxtLink v-if="p.href" :to="p.href" :class="cardClass">
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-badge-bg text-primary sm:h-11 sm:w-11">
              <Icon :name="p.icon" class="h-4 w-4 sm:h-5 sm:w-5" />
            </div>
            <div class="min-w-0">
              <h3 class="truncate text-[13px] font-bold text-navy sm:text-[15px]">{{ p.title }}</h3>
              <p class="mt-0.5 truncate text-[11px] text-muted sm:text-xs">{{ p.desc }}</p>
            </div>
          </NuxtLink>
          <div v-else :class="[cardClass, 'cursor-default opacity-70 hover:translate-y-0 hover:border-card-border hover:shadow-sm']">
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-badge-bg text-primary sm:h-11 sm:w-11">
              <Icon :name="p.icon" class="h-4 w-4 sm:h-5 sm:w-5" />
            </div>
            <div class="min-w-0">
              <h3 class="truncate text-[13px] font-bold text-navy sm:text-[15px]">{{ p.title }}</h3>
              <p class="mt-0.5 truncate text-[11px] text-muted sm:text-xs">{{ p.desc }}</p>
              <span class="mt-1 inline-block text-[10px] font-bold uppercase tracking-wide text-muted">Coming soon</span>
            </div>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>
