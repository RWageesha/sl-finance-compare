<script setup lang="ts">
// Detail page for rows shown on the (still mock-data-driven) Product
// Directory and Compare pages — every "View Details" link from those two
// systems lands here. Bank Profile's own "Details" links go straight to
// the real, live-API-backed /products/[group]/[id] and
// /products/fixed-deposits/[slug] pages instead; this page exists so the
// mock-data pages never dead-end while they're still mock-data-driven.
// Same visual system as those real pages (dark navy header, card panels)
// so a "View Details" click never feels like a different, lesser site.
import { findSampleProduct } from '~/config/sampleProducts'
import { PRODUCT_TYPES } from '~/config/productTypes'
import { bankColor, bankInitial, bankKey } from '~/utils/bankColors'

const route = useRoute()
const id = String(route.params.id)
const product = findSampleProduct(id)

if (!product) {
  throw createError({ statusCode: 404, statusMessage: 'Product not found', fatal: true })
}

useHead({ title: `${product.bank} — ${product.product ?? product.typeLabel} — FindRate LK` })

const typeConfig = computed(() => PRODUCT_TYPES[product!.type])

const rateValue = computed(() => {
  if (product!.rateValue !== undefined) return product!.rateValue
  const n = Number.parseFloat(product!.rate ?? '')
  return Number.isFinite(n) ? n : null
})

const compareHref = computed(() => {
  const base = typeConfig.value?.compareHref
  if (!base || base === '#') return null
  const params = new URLSearchParams({ fromBank: product!.bank, fromProduct: product!.product ?? product!.typeLabel })
  if (rateValue.value !== null) params.set('rate', String(rateValue.value))
  return `${base}?${params.toString()}`
})

// Whichever descriptive fields this row actually carries — no fabricated
// fields for the ones it doesn't (e.g. gold loans have no maxAmount).
const infoItems = computed(() => {
  const p = product!
  const items: { label: string; value: string }[] = []
  if (p.category) items.push({ label: 'Category', value: p.category })
  if (p.tier) items.push({ label: 'Tier', value: p.tier })
  if (p.tenure) items.push({ label: 'Tenure', value: p.tenure })
  if (p.payment) items.push({ label: 'Interest Payment', value: p.payment })
  if (p.rateType) items.push({ label: 'Rate Type', value: p.rateType })
  if (p.minDeposit) items.push({ label: 'Min. Deposit', value: p.minDeposit })
  if (p.minBalance) items.push({ label: 'Min. Balance', value: p.minBalance })
  if (p.maxAmount) items.push({ label: 'Max Amount', value: p.maxAmount })
  if (p.annualFee) items.push({ label: 'Annual Fee', value: p.annualFee })
  if (p.atmLimit) items.push({ label: 'ATM Daily Limit', value: p.atmLimit })
  if (p.terms) items.push({ label: 'Terms', value: p.terms })
  if (p.illustrativeRate) items.push({ label: 'Illustrative Flat Rate (for EMI estimate)', value: p.illustrativeRate })
  return items
})

// Other banks' products of the same type — mirrors the real detail
// pages' "Compare Similar" section, drawn from the same directory rows
// (not a separate list), excluding this one row.
const similar = computed(() => {
  if (!typeConfig.value) return []
  return typeConfig.value.rows.filter((r) => r.id !== product!.id).slice(0, 6)
})
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-6 sm:px-6">
      <div class="mx-auto max-w-[1080px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted sm:text-sm" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&rsaquo;</span>
          <NuxtLink :to="product.typeHref" class="hover:text-primary">{{ product.typeLabel }}</NuxtLink>
          <span>&rsaquo;</span>
          <span class="font-semibold text-primary">{{ product.bank }} {{ product.product }}</span>
        </nav>

        <!-- Header card -->
        <section class="flex flex-col gap-5 rounded-card bg-navy p-5 sm:flex-row sm:items-start sm:justify-between sm:p-7">
          <div class="min-w-0">
            <div class="mb-3 flex items-center gap-2.5">
              <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
                :style="{ backgroundColor: bankColor(product.bank) }"
              >{{ bankInitial(product.bank) }}</span>
              <span class="text-sm font-bold text-[#7fa8ff]">{{ product.bank }}</span>
            </div>
            <h1 class="text-xl font-extrabold text-white sm:text-2xl">{{ product.product ?? product.typeLabel }}</h1>
            <p v-if="product.verified" class="mt-1.5 text-xs text-[#94a3b8]">Verified {{ product.verified }}</p>
          </div>
          <div class="shrink-0 text-right">
            <p class="text-[10px] font-bold uppercase tracking-wider text-[#94a3b8]">{{ product.tier ? 'APR' : 'Rate (P.A.)' }}</p>
            <p class="text-3xl font-extrabold text-[#7fa8ff] sm:text-[2.1rem]">{{ product.rate }}</p>
            <NuxtLink v-if="compareHref" :to="compareHref" class="mt-2 inline-block rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition hover:bg-primary/90">
              Compare Rates &rarr;
            </NuxtLink>
          </div>
        </section>

        <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[1fr_300px] lg:items-start">
          <div class="flex flex-col gap-4">
            <section class="rounded-card border border-card-border bg-card p-5">
              <h2 class="mb-3 text-base font-bold text-navy">Key Information</h2>
              <div v-if="infoItems.length" class="grid grid-cols-2 gap-4">
                <div v-for="item in infoItems" :key="item.label" class="flex flex-col gap-1">
                  <span class="text-[11px] font-semibold uppercase tracking-wide text-muted">{{ item.label }}</span>
                  <strong class="text-sm text-navy">{{ item.value }}</strong>
                </div>
              </div>
              <p v-if="product.illustrativeRate" class="mt-4 border-t border-card-border pt-3 text-xs leading-relaxed text-muted">
                This product's rate is linked to AWPLR, so the comparison tool above uses a separate flat rate to
                estimate a monthly payment — see "Illustrative Flat Rate" above. It isn't the bank's disclosed rate.
              </p>
              <p class="mt-3 border-t border-card-border pt-3 text-xs leading-relaxed text-muted">
                This is illustrative sample data for FindRate LK's product comparison tools, not a live-verified bank
                disclosure. <NuxtLink :to="product.typeHref" class="font-semibold text-primary hover:underline">Browse the {{ product.typeLabel }} directory &rarr;</NuxtLink>
              </p>
            </section>

            <section v-if="similar.length" class="rounded-card border border-card-border bg-card p-5">
              <h2 class="text-base font-bold text-navy">Compare Similar {{ product.typeLabel }}</h2>
              <p class="-mt-1 mb-3 text-xs text-muted">Other banks' {{ product.typeLabel.toLowerCase() }} products.</p>
              <div class="flex flex-col gap-2">
                <NuxtLink
                  v-for="r in similar"
                  :key="r.id"
                  :to="`/products/sample/${r.id}`"
                  class="flex items-center justify-between gap-3 rounded-lg border border-card-border px-3.5 py-2.5 text-sm transition hover:border-primary"
                  :class="bankKey(r.bank) === bankKey(product.bank) ? 'bg-badge-bg' : ''"
                >
                  <span class="min-w-0 truncate text-navy">{{ r.bank }} — {{ r.product }}</span>
                  <strong class="shrink-0 text-primary">{{ r.rate }}</strong>
                </NuxtLink>
              </div>
            </section>
          </div>

          <aside class="flex flex-col gap-4">
            <section class="rounded-card border border-card-border bg-card p-5">
              <h2 class="mb-2 text-base font-bold text-navy">About This Listing</h2>
              <p class="text-xs leading-relaxed text-muted">
                FindRate LK is building out live, verified data for every product category. Until this category's real
                data feed goes live, this page shows the same sample figures used on the
                {{ product.typeLabel }} directory and comparison pages, so every product listed there has somewhere to
                link to.
              </p>
            </section>
          </aside>
        </div>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
