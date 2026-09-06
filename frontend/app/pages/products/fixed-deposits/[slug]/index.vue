<script setup lang="ts">
import { isFdRow, parseFdSlug, findFdRow, fdSlug } from '~/utils/fdCompare'
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import { findDetailExtras } from '~/config/productDetailExtras'
import type { ProductRate } from '~/composables/useRatesApi'

const route = useRoute()
const slug = String(route.params.slug)
const parsed = parseFdSlug(slug)
if (!parsed) {
  throw createError({ statusCode: 404, statusMessage: 'Product not found', fatal: true })
}

const { fetchFixedDeposits, fetchRateHistory } = useRatesApi()
const fdRows = ref<ProductRate[]>([])
const loading = ref(true)

const row = ref<ProductRate | null>(null)
// Oldest first, as the API returns it — every scrape ever recorded for
// this exact product line, not just the latest one `row` holds.
const history = ref<ProductRate[]>([])

onMounted(async () => {
  try {
    fdRows.value = (await fetchFixedDeposits().catch(() => [])).filter(isFdRow)
    row.value = findFdRow(fdRows.value, parsed!) ?? null
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

const bank = computed(() => DIRECTORY_BANKS.find((b) => b.slug === parsed!.bankSlug))
const extras = computed(() => findDetailExtras(slug))

useHead({
  title: computed(() =>
    row.value ? `${bank.value?.displayName ?? row.value.bank_name} — ${fmtTenure(row.value.tenure_value ?? 0)} Fixed Deposit — FindRate LK` : 'Fixed Deposit — FindRate LK'
  )
})

// Other tenures of the same product (same bank + category) — real, drawn
// from the same fetch, not a separate "similar products" algorithm.
const similar = computed(() => {
  if (!row.value) return []
  return fdRows.value
    .filter((r) => r.bank_name === row.value!.bank_name && r.category_code === row.value!.category_code && r.id !== row.value!.id)
    .sort((a, b) => (a.tenure_value ?? 0) - (b.tenure_value ?? 0))
})

function similarHref(r: ProductRate): string {
  return `/products/fixed-deposits/${fdSlug(parsed!.bankSlug, r.category_code, r.tenure_value ?? 0)}`
}

const compareHref = computed(() => {
  if (!row.value || !bank.value) return '/compare/fixed-deposits'
  const params = new URLSearchParams({
    fromBank: bank.value.slug,
    fromProduct: `${fmtTenure(row.value.tenure_value ?? 0)} Fixed Deposit`,
    rate: row.value.interest_rate.toFixed(2),
    tenure: String(row.value.tenure_value ?? '')
  })
  if (row.value.min_amount) params.set('amount', String(row.value.min_amount))
  return `/compare/fixed-deposits?${params.toString()}`
})

const reportUrl = computed(() => {
  if (!row.value || !bank.value) return '/report-issue'
  const params = new URLSearchParams({
    bank: bank.value.slug,
    category: 'Fixed Deposits',
    product: `${fmtTenure(row.value.tenure_value ?? 0)} Fixed Deposit`,
    currentValue: `${row.value.interest_rate.toFixed(2)}% p.a.`
  })
  return `/report-issue?${params.toString()}`
})

// The 6-pair Key Information grid — only the pairs backed by real data
// or (for HNB's 12-month FD, so far) this product's detail extras.
// Anything neither source has is simply omitted, not shown blank.
const keyInfo = computed(() => {
  if (!row.value) return []
  const ex = extras.value
  const items: { label: string; value: string }[] = []
  if (ex?.minDeposit) items.push({ label: 'Minimum Deposit', value: ex.minDeposit })
  if (ex?.maxDeposit) items.push({ label: 'Maximum Deposit', value: ex.maxDeposit })
  if (ex?.paymentFrequency) items.push({ label: 'Interest Payment', value: ex.paymentFrequency })
  items.push({ label: 'Tenure', value: fmtTenure(row.value.tenure_value ?? 0) })
  if (ex?.earlyWithdrawal) items.push({ label: 'Early Withdrawal', value: ex.earlyWithdrawal })
  if (ex?.tax) items.push({ label: 'Tax', value: ex.tax })
  if (!ex) {
    items.push({ label: 'Category', value: formatCategoryLabel(row.value.category_code) })
    if (row.value.rate_label) items.push({ label: 'Rate Label', value: row.value.rate_label })
    items.push({ label: 'Bank', value: row.value.bank_name })
  }
  items.push({ label: 'Last Verified Scrape', value: fmtRelativeDate(row.value.scraped_at) })
  return items
})

const chartPoints = computed(() => history.value.map((r) => ({ date: r.scraped_at, rate: r.interest_rate })))
// Most-recent-first, paired against the scrape immediately before it —
// real deltas from stored history, skipping pairs with no actual change.
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
          <NuxtLink to="/products/fixed-deposits" class="hover:text-primary">Fixed Deposits</NuxtLink>
          <span>&gt;</span>
          <span v-if="bank" class="font-semibold text-primary">{{ bank.displayName.replace(/\s*\(.*\)/, '') }} {{ row ? fmtTenure(row.tenure_value ?? 0) : '' }} Fixed Deposit</span>
        </nav>

        <p v-if="loading" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">Loading…</p>
        <p v-else-if="!row" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
          This product isn't in our current dataset — it may have been part of a scrape that's since rotated out, or the link is out of date.
          <NuxtLink to="/products/fixed-deposits" class="mt-2 block font-bold text-primary hover:underline">Browse Fixed Deposits &rarr;</NuxtLink>
        </p>

        <template v-else>
          <ProductHeroCard
            :bank-name="bank?.displayName ?? row.bank_name"
            :bank-slug="bank?.slug"
            :product-name="`${fmtTenure(row.tenure_value ?? 0)} Fixed Deposit`"
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
                <h2 class="text-base font-bold text-navy">Compare Similar Fixed Deposits</h2>
                <p class="-mt-1 mb-3 text-xs text-muted">Other tenures of the same {{ formatCategoryLabel(row.category_code) }} product at {{ row.bank_name }}.</p>
                <div class="flex flex-col gap-2">
                  <NuxtLink
                    v-for="r in similar"
                    :key="r.id"
                    :to="similarHref(r)"
                    class="flex items-center justify-between rounded-lg border border-card-border px-3.5 py-2.5 text-sm transition hover:border-primary"
                  >
                    <span class="text-navy">{{ fmtTenure(r.tenure_value ?? 0) }}</span>
                    <strong class="text-primary">{{ r.interest_rate.toFixed(2) }}% p.a.</strong>
                  </NuxtLink>
                </div>
              </section>
            </div>

            <aside class="flex flex-col gap-4">
              <RateHistoryCard
                :points="chartPoints"
                :history-href="`/products/fixed-deposits/${slug}/history`"
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
            category-label="Fixed Deposits"
            compare-label="Compare FD"
            :compare-href="compareHref"
            :report-url="reportUrl"
          />
        </template>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
