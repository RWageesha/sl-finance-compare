<script setup lang="ts">
// Real logos already exist in public/banks/ (from the bank directory work
// earlier in the project) — used directly rather than placeholders. Product
// counts are computed from the live API (see onMounted below), never
// hardcoded — a bank's real count changes every time its scraper runs.
import { computeBankStats, DIRECTORY_BANKS } from '~/utils/bankDirectory'
import type { TaggedRow } from '~/utils/bankDirectory'

const BANKS = [
  // BOC has no icon-cropped -small variant, so it keeps the full logo.
  { name: 'Bank of Ceylon', logoSrc: '/banks/boc.jpeg', slug: 'boc' },
  { name: 'Commercial Bank', logoSrc: '/banks/combank-small.png', slug: 'combank' },
  { name: 'Sampath Bank', logoSrc: '/banks/sampath-small.png', slug: 'sampath' },
  { name: 'HNB', logoSrc: '/banks/hnb-small.png', slug: 'hnb' },
  { name: 'National Savings Bank', logoSrc: '/banks/nsb-small.png', slug: 'nsb' },
  { name: "People's Bank", logoSrc: '/banks/peoples-small.png', slug: 'peoples' }
]

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()
const allRows = ref<TaggedRow[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const [fd, savings, loans] = await Promise.all([
      fetchFixedDeposits().catch(() => []),
      fetchSavings().catch(() => []),
      fetchLoans().catch(() => [])
    ])
    allRows.value = [
      ...fd.map((r) => ({ ...r, kind: 'fd' as const })),
      ...savings.map((r) => ({ ...r, kind: 'savings' as const })),
      ...loans.map((r) => ({ ...r, kind: 'loans' as const }))
    ]
  } finally {
    loading.value = false
  }
})

function productCountFor(slug: string): number {
  const apiName = DIRECTORY_BANKS.find((b) => b.slug === slug)?.apiName
  return computeBankStats(allRows.value, apiName).productCount
}
</script>

<template>
  <section class="bg-card px-4 py-7 sm:px-6 sm:py-10">
    <div class="mx-auto max-w-[1080px]">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <span class="inline-block rounded-pill bg-badge-bg px-3 py-1 text-xs font-bold uppercase tracking-wider text-primary">
            Banks
          </span>
          <h2 class="mt-3 text-[1.3rem] font-bold text-navy sm:text-[1.6rem]">Most Popular Banks</h2>
          <p class="mt-1 text-sm text-muted">The most-tracked bank directories on FindRate LK.</p>
        </div>
        <NuxtLink to="/banks" class="text-sm font-semibold text-primary hover:underline">View All Banks &rarr;</NuxtLink>
      </div>

      <div class="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-3.5 md:grid-cols-6">
        <NuxtLink
          v-for="bank in BANKS"
          :key="bank.slug"
          :to="`/banks/${bank.slug}`"
          class="flex flex-col items-center rounded-card border border-card-border bg-card px-3 py-4 text-center transition hover:-translate-y-0.5 hover:border-primary hover:shadow-md sm:px-4 sm:py-5"
        >
          <span class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-lg border border-card-border bg-white p-1.5">
            <img :src="bank.logoSrc" :alt="`${bank.name} logo`" class="h-full w-full object-contain">
          </span>
          <span class="mt-3 text-[13px] font-bold leading-snug text-navy">{{ bank.name }}</span>
          <span class="mt-1 text-xs text-muted">{{ loading ? '…' : `${productCountFor(bank.slug)} products` }}</span>
        </NuxtLink>
      </div>
    </div>
  </section>
</template>
