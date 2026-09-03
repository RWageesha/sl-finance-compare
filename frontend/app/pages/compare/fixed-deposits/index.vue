<script setup lang="ts">
import { isFdRow } from '~/utils/fdCompare'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Compare Fixed Deposit Rates — OpenFinance LK' })

const { fetchFixedDeposits } = useRatesApi()
const fdRows = ref<ProductRate[]>([])

onMounted(async () => {
  fdRows.value = (await fetchFixedDeposits().catch(() => [])).filter(isFdRow)
})

// Real available tenures only — not a fixed guessed list.
const tenureOptions = computed(() => {
  const values = new Set(fdRows.value.map((r) => r.tenure_value).filter((v): v is number => v != null))
  return [...values].sort((a, b) => a - b)
})

const amount = ref(1000000)
const tenure = ref<number | null>(null)

// Once real tenures load, default to 12 months if it's offered, else the
// first available tenure.
watch(tenureOptions, (opts) => {
  if (tenure.value !== null || opts.length === 0) return
  tenure.value = opts.includes(12) ? 12 : opts[0]
})

function onCompare() {
  navigateTo({
    path: '/compare/fixed-deposits/results',
    query: { amount: String(amount.value), tenure: String(tenure.value ?? '') }
  })
}
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <section class="intro">
          <p class="eyebrow">Financial Comparison</p>
          <h1>Compare Fixed Deposit Rates</h1>
          <p class="sub">Find the best fixed deposit rates from Sri Lankan banks. Compare interest rates and tenures in one place.</p>

          <div class="input-card">
            <div class="field">
              <label for="amount">Deposit Amount (LKR)</label>
              <input id="amount" v-model.number="amount" type="number" min="0" step="10000">
            </div>
            <div class="field">
              <label for="tenure">Tenure</label>
              <select id="tenure" v-model.number="tenure">
                <option v-if="tenureOptions.length === 0" :value="null">Loading…</option>
                <option v-for="t in tenureOptions" :key="t" :value="t">{{ fmtTenure(t) }}</option>
              </select>
            </div>
            <button type="button" class="compare-btn" :disabled="tenure === null" @click="onCompare">Compare Rates</button>
          </div>
        </section>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>

<style scoped>
.wrap {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 1.5rem;
}
main {
  padding: 0.5rem 0 1rem;
}
.intro {
  text-align: center;
  padding: 2.5rem 0 3rem;
}
.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--accent);
  margin: 0 0 0.4rem;
}
.intro h1 {
  margin: 0 0 0.6rem;
  font-size: 2rem;
  font-weight: 800;
}
.sub {
  margin: 0 auto 1.8rem;
  max-width: 560px;
  color: var(--muted);
  font-size: 0.92rem;
}
.input-card {
  display: flex;
  align-items: flex-end;
  gap: 1.5rem;
  max-width: 620px;
  margin: 0 auto;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.3rem 1.5rem;
  text-align: left;
  flex-wrap: wrap;
}
.field {
  flex: 1;
  min-width: 140px;
}
.field label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--muted);
  margin-bottom: 0.35rem;
}
.field input,
.field select {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--bg);
  font: inherit;
  font-size: 0.9rem;
  padding: 0.55rem 0.7rem;
  color: var(--text);
}
.compare-btn {
  font: inherit;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 0.65rem 1.4rem;
  border-radius: 7px;
  border: none;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
  white-space: nowrap;
}
.compare-btn:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
