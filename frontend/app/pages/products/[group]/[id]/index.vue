<script setup lang="ts">
// Shared detail page for Savings and Loan products (Fixed Deposits have
// their own dedicated page at pages/products/fixed-deposits/[slug]/ since
// tenure genuinely identifies one logical FD product — see fdCompare.ts
// for why that scheme doesn't work here). Static routes take precedence
// over this dynamic [group] segment in Nuxt's router, so /products/
// fixed-deposits/... always resolves to that page, never this one.
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import { findDetailExtras } from '~/config/productDetailExtras'
import type { ProductRate } from '~/composables/useRatesApi'

const GROUP_META = {
  savings: {
    noun: 'Savings Account',
    directoryLabel: 'Savings Accounts',
    directoryHref: '/rates?tab=savings',
    disclosure: 'minimum balance, account fees, or other terms'
  },
  loans: {
    noun: 'Loan',
    directoryLabel: 'Loans',
    directoryHref: '/rates?tab=loans',
    disclosure: 'processing fees, collateral or eligibility requirements, or other terms'
  }
} as const

const route = useRoute()
const group = String(route.params.group) as keyof typeof GROUP_META
const id = Number(route.params.id)

if (!(group in GROUP_META) || !Number.isFinite(id)) {
  throw createError({ statusCode: 404, statusMessage: 'Product not found', fatal: true })
}
const meta = GROUP_META[group]

const { fetchSavings, fetchLoans, fetchRateHistory } = useRatesApi()
const rows = ref<ProductRate[]>([])
const loading = ref(true)
const row = ref<ProductRate | null>(null)
const history = ref<ProductRate[]>([])

onMounted(async () => {
  try {
    rows.value = await (group === 'savings' ? fetchSavings() : fetchLoans()).catch(() => [])
    row.value = rows.value.find((r) => r.id === id) ?? null
    if (row.value) {
      history.value = await fetchRateHistory({
        product_id: row.value.product_id,
        tenure_value: row.value.tenure_value,
        tenure_label: row.value.tenure_label,
        rate_label: row.value.rate_label
      }).catch(() => [])
    }
  } finally {
    loading.value = false
  }
})

const bank = computed(() => (row.value ? DIRECTORY_BANKS.find((b) => b.apiName === row.value!.bank_name) : undefined))
const extras = computed(() => findDetailExtras(`${group}-${id}`))

useHead({
  title: computed(() =>
    row.value ? `${row.value.bank_name} — ${formatCategoryLabel(row.value.category_code)} — FindRate LK` : `${meta.noun} — FindRate LK`
  )
})

// Real product label: whatever the bank's own tenure/rate label says
// (e.g. "Below Rs. 500,000/-", "Fixed Rate"), falling back to the
// category name when neither is present.
function productLabel(r: ProductRate): string {
  return r.tenure_label || r.rate_label || formatCategoryLabel(r.category_code)
}

const similar = computed(() => {
  if (!row.value) return []
  return rows.value.filter((r) => r.bank_name === row.value!.bank_name && r.category_code === row.value!.category_code && r.id !== row.value!.id)
})

// Which new /compare/[slug] page this product belongs to — loans need
// their own category (housing vs personal) since both share this one
// detail page; gold loans and anything else has no new Compare page yet,
// so those fall back to the old /rates directory link instead.
const compareHref = computed(() => {
  if (!row.value) return meta.directoryHref
  let slug: string | null = null
  if (group === 'savings') slug = 'savings-accounts'
  else if (row.value.category_code === 'HOUSING_LOAN') slug = 'housing-loans'
  else if (row.value.category_code === 'PERSONAL_LOAN') slug = 'personal-loans'
  if (!slug) return meta.directoryHref
  const params = new URLSearchParams({
    fromBank: bank.value?.slug ?? row.value.bank_name,
    fromProduct: productLabel(row.value),
    rate: row.value.interest_rate.toFixed(2)
  })
  if (row.value.tenure_value) params.set('tenure', String(row.value.tenure_value))
  if (row.value.min_amount) params.set('amount', String(row.value.min_amount))
  return `/compare/${slug}?${params.toString()}`
})
const compareLabel = computed(() => (group === 'savings' ? 'Compare Savings' : 'Compare Loan'))

const reportUrl = computed(() => {
  if (!row.value) return '/report-issue'
  const params = new URLSearchParams({
    bank: bank.value?.slug ?? '',
    category: meta.directoryLabel,
    product: `${row.value.bank_name} ${formatCategoryLabel(row.value.category_code)}`,
    currentValue: `${row.value.interest_rate.toFixed(2)}% p.a.`
  })
  return `/report-issue?${params.toString()}`
})

const keyInfo = computed(() => {
  if (!row.value) return []
  const ex = extras.value
  const items: { label: string; value: string }[] = []
  if (ex?.minDeposit) items.push({ label: group === 'savings' ? 'Minimum Balance' : 'Minimum Amount', value: ex.minDeposit })
  if (ex?.maxDeposit) items.push({ label: 'Maximum Amount', value: ex.maxDeposit })
  if (ex?.paymentFrequency) items.push({ label: 'Interest Payment', value: ex.paymentFrequency })
  items.push({ label: 'Category', value: formatCategoryLabel(row.value.category_code) })
  if (row.value.tenure_label) items.push({ label: group === 'savings' ? 'Balance Tier' : 'Tenure', value: row.value.tenure_label })
  if (ex?.earlyWithdrawal) items.push({ label: 'Early Withdrawal', value: ex.earlyWithdrawal })
  if (ex?.tax) items.push({ label: 'Tax', value: ex.tax })
  if (!ex) {
    if (row.value.rate_label) items.push({ label: 'Rate Label', value: row.value.rate_label })
    items.push({ label: 'Bank', value: row.value.bank_name })
  }
  items.push({ label: 'Last Verified Scrape', value: fmtRelativeDate(row.value.scraped_at) })
  return items
})

const chartPoints = computed(() => history.value.map((r) => ({ date: r.scraped_at, rate: r.interest_rate })))
const recentChanges = computed(() => {
  const asc = history.value
  const changes: { date: string; oldRate: number; newRate: number }[] = []
  for (let i = 1; i < asc.length; i++) {
    if (asc[i].interest_rate === asc[i - 1].interest_rate) continue
    changes.push({ date: asc[i].scraped_at, oldRate: asc[i - 1].interest_rate, newRate: asc[i].interest_rate })
  }
  return changes.reverse().slice(0, 3)
})
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1080px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&gt;</span>
          <NuxtLink :to="meta.directoryHref" class="hover:text-primary">{{ meta.directoryLabel }}</NuxtLink>
          <span>&gt;</span>
          <span v-if="row" class="font-semibold text-primary">{{ row.bank_name }} {{ formatCategoryLabel(row.category_code) }}</span>
        </nav>

        <p v-if="loading" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">Loading…</p>
        <p v-else-if="!row" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
          This product isn't in our current dataset — it may have been part of a scrape that's since rotated out, or the link is out of date.
          <NuxtLink :to="meta.directoryHref" class="mt-2 block font-bold text-primary hover:underline">Browse {{ meta.directoryLabel }} &rarr;</NuxtLink>
        </p>

        <template v-else>
          <ProductHeroCard
            :bank-name="bank?.displayName ?? row.bank_name"
            :bank-slug="bank?.slug"
            :product-name="`${formatCategoryLabel(row.category_code)}${row.tenure_label || row.rate_label ? ' — ' + productLabel(row) : ''}`"
            :last-updated="fmtDate(row.scraped_at)"
            :rate="`${row.interest_rate.toFixed(2)}%`"
            :payment-frequency="extras?.paymentFrequency"
          />

          <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[2fr_1fr] lg:items-start">
            <div class="flex flex-col gap-4">
              <KeyInformationCard :items="keyInfo" />
              <EligibilityCard v-if="extras?.eligibility?.length" :items="extras.eligibility" />
              <FeesConditionsCard v-if="extras?.feesConditions?.length" :items="extras.feesConditions" />

              <section v-if="!extras && similar.length" class="rounded-[14px] border border-card-border bg-card p-5 shadow-sm">
                <h2 class="text-base font-bold text-navy">Compare Similar {{ meta.directoryLabel }}</h2>
                <p class="-mt-1 mb-3 text-xs text-muted">Other {{ formatCategoryLabel(row.category_code) }} products at {{ row.bank_name }}.</p>
                <div class="flex flex-col gap-2">
                  <NuxtLink
                    v-for="r in similar"
                    :key="r.id"
                    :to="`/products/${group}/${r.id}`"
                    class="flex items-center justify-between gap-3 rounded-lg border border-card-border px-3.5 py-2.5 text-sm transition hover:border-primary"
                  >
                    <span class="min-w-0 truncate text-navy">{{ productLabel(r) }}</span>
                    <strong class="shrink-0 text-primary">{{ r.interest_rate.toFixed(2) }}% p.a.</strong>
                  </NuxtLink>
                </div>
              </section>
            </div>

            <aside class="flex flex-col gap-4">
              <RateHistoryCard
                :points="chartPoints"
                :history-href="`/products/${group}/${id}/history`"
                :recent-changes="recentChanges"
              />
              <SourceVerificationCard
                :source-label="extras?.sourceLabel"
                :source-url="row.source_url"
                :last-verified="fmtDate(row.scraped_at)"
              />
            </aside>
          </div>

          <RelatedProductsSection
            :category-label="meta.directoryLabel"
            :compare-label="compareLabel"
            :compare-href="compareHref"
            :report-url="reportUrl"
          />
        </template>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
