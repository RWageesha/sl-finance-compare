<script setup lang="ts">
import { isFdRow, bankSlugForApiName, fdSlug, fmtLkr } from '~/utils/fdCompare'
import { calcFd } from '~/utils/calculatorMath'
import type { FdFrequency } from '~/utils/calculatorMath'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Fixed Deposit Calculator — FindRate LK' })

const { fetchFixedDeposits } = useRatesApi()
const fdRows = ref<ProductRate[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    fdRows.value = (await fetchFixedDeposits().catch(() => [])).filter(isFdRow)
  } finally {
    loading.value = false
  }
})

interface ProductOption {
  key: string
  label: string
  row: ProductRate
}

// Real product picker: each option is one real scraped FD row. Selecting
// one locks Rate and Tenure to that row's real values — only Deposit
// Amount stays freely editable, per the actual request (pick a real
// product, only vary how much you're putting in).
const productOptions = computed<ProductOption[]>(() => {
  return fdRows.value
    .map((r): ProductOption | null => {
      const bankSlug = bankSlugForApiName(r.bank_name)
      if (!bankSlug) return null
      const key = fdSlug(bankSlug, r.category_code, r.tenure_value ?? 0)
      const label = `${r.bank_name} — ${formatCategoryLabel(r.category_code)} — ${fmtTenure(r.tenure_value ?? 0)} — ${r.interest_rate.toFixed(2)}%`
      return { key, label, row: r }
    })
    .filter((o): o is ProductOption => o !== null)
    .sort((a, b) => a.label.localeCompare(b.label))
})

const selectedKey = ref('')
const selectedOption = computed(() => productOptions.value.find((o) => o.key === selectedKey.value) ?? null)
const isCustom = computed(() => !selectedOption.value)

const principal = ref(1000000)
const rate = ref(8.25)
const tenure = ref(12)
const frequency = ref<FdFrequency>('maturity')
const whtPct = ref(5)

watch(selectedOption, (opt) => {
  if (opt) {
    rate.value = opt.row.interest_rate
    tenure.value = opt.row.tenure_value ?? 0
  }
})

function useCustom() {
  selectedKey.value = ''
}

const hasCalculated = ref(false)
function calculate() {
  hasCalculated.value = true
}

const result = computed(() => calcFd(principal.value, rate.value, tenure.value, whtPct.value, frequency.value))

// Growth Projection bars (maturity mode only) — 9 evenly spaced points
// from 0 to the full tenure, using the same simple-interest formula as
// the result above.
const growthBars = computed(() => {
  if (frequency.value !== 'maturity') return []
  const POINTS = 9
  return Array.from({ length: POINTS }, (_, i) => {
    const m = (tenure.value * i) / (POINTS - 1)
    return principal.value * (1 + (rate.value / 100) * (m / 12))
  })
})
const maxBar = computed(() => Math.max(...growthBars.value, 1))
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <p class="eyebrow">Fixed Deposit</p>
        <h1>Fixed Deposit Calculator</h1>
        <p class="sub">Estimate your monthly or maturity returns for fixed deposits across banks FindRate LK tracks in Sri Lanka.</p>

        <div class="calc-grid">
          <div class="panel">
            <h2>Investment Parameters</h2>

            <div class="field">
              <label for="product">Select a Product (optional)</label>
              <select id="product" v-model="selectedKey">
                <option value="">Custom (enter manually)</option>
                <option v-for="o in productOptions" :key="o.key" :value="o.key">{{ o.label }}</option>
              </select>
              <p v-if="loading" class="field-hint">Loading real products…</p>
              <p v-else-if="selectedOption" class="field-hint">
                Using {{ selectedOption.row.bank_name }}'s real disclosed rate, updated {{ fmtRelativeDate(selectedOption.row.scraped_at) }}.
                <button type="button" class="link-btn" @click="useCustom">Use custom instead</button>
              </p>
            </div>

            <div class="field">
              <label for="principal">Principal Amount</label>
              <div class="input-suffix">
                <input id="principal" v-model.number="principal" type="number" min="0" step="10000">
                <span>LKR</span>
              </div>
            </div>

            <div class="field-row">
              <div class="field">
                <label for="rate">Interest Rate (Annual)</label>
                <div class="input-suffix">
                  <input id="rate" v-model.number="rate" type="number" min="0" step="0.05" :disabled="!isCustom">
                  <span>%</span>
                </div>
              </div>
              <div class="field">
                <label for="tenure">Tenure</label>
                <div class="input-suffix">
                  <input id="tenure" v-model.number="tenure" type="number" min="1" step="1" :disabled="!isCustom">
                  <span>Months</span>
                </div>
              </div>
            </div>

            <div class="field-row">
              <div class="field">
                <label for="frequency">Interest Payout</label>
                <select id="frequency" v-model="frequency">
                  <option value="maturity">At Maturity</option>
                  <option value="monthly">Monthly</option>
                  <option value="annual">Annually</option>
                </select>
              </div>
              <div class="field">
                <label for="wht">Withholding Tax</label>
                <div class="input-suffix">
                  <input id="wht" v-model.number="whtPct" type="number" min="0" max="100" step="0.5">
                  <span>%</span>
                </div>
              </div>
            </div>
            <p class="tax-note">Sri Lanka applies withholding tax to interest income — adjust this if your rate differs, or set it to 0 to see the pre-tax figures.</p>

            <button type="button" class="calc-btn" @click="calculate">Calculate Yield</button>
          </div>

          <div class="panel result-panel">
            <template v-if="!hasCalculated">
              <div class="placeholder">
                <p>Fill in the details on the left and click <strong>Calculate Yield</strong> to see your estimated returns.</p>
              </div>
            </template>
            <template v-else>
              <p class="result-eyebrow">{{ frequency === 'maturity' ? 'Estimated Maturity Value' : 'Estimated Total Return' }}</p>
              <p class="result-value">{{ fmtLkr(result.netTotal) }}</p>

              <div class="breakdown">
                <div class="row"><span>Principal Amount</span><strong>{{ fmtLkr(result.principal) }}</strong></div>
                <div class="row"><span>Estimated Interest Earned</span><strong>{{ fmtLkr(result.totalInterest) }}</strong></div>
                <div class="row"><span>Withholding Tax ({{ whtPct }}%)</span><strong class="neg">&minus; {{ fmtLkr(result.tax) }}</strong></div>
                <div class="row total"><span>Net Amount After Tax</span><strong>{{ fmtLkr(result.netTotal) }}</strong></div>
              </div>

              <template v-if="frequency === 'maturity'">
                <p class="chart-title">Growth Projection</p>
                <div class="bar-chart">
                  <div v-for="(v, i) in growthBars" :key="i" class="bar" :style="{ height: `${(v / maxBar) * 100}%` }" />
                </div>
                <div class="bar-labels"><span>0 Months</span><span>{{ tenure }} Months</span></div>
              </template>
              <template v-else>
                <div class="payout-note">
                  You'll receive <strong>{{ fmtLkr(result.periodicPayoutAfterTax ?? 0) }}</strong>
                  {{ frequency === 'monthly' ? 'every month' : 'every year' }}
                  for {{ result.periodsCount }} {{ frequency === 'monthly' ? 'months' : 'years' }},
                  plus your {{ fmtLkr(result.principal) }} principal back at maturity.
                </div>
              </template>
            </template>
          </div>
        </div>

        <div class="cta-bar">
          <span>Rates look competitive?</span>
          <NuxtLink to="/compare/fixed-deposits">Compare FD rates from all banks in Sri Lanka &rarr;</NuxtLink>
        </div>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>

<style scoped>
.wrap {
  max-width: 900px;
  margin: 0 auto;
  padding: 0 1.5rem;
}
main {
  padding: 1.5rem 0 1rem;
}
.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--accent);
  margin: 0 0 0.3rem;
}
h1 {
  margin: 0 0 0.4rem;
  font-size: 1.6rem;
  font-weight: 800;
}
.sub {
  margin: 0 0 1.4rem;
  max-width: 640px;
  color: var(--muted);
  font-size: 0.88rem;
}

.calc-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.2rem;
  align-items: start;
}
@media (max-width: 720px) {
  .calc-grid {
    grid-template-columns: 1fr;
  }
}
.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.3rem 1.4rem;
}
.panel h2 {
  margin: 0 0 1rem;
  font-size: 0.95rem;
  font-weight: 700;
}

.field {
  margin-bottom: 0.9rem;
  flex: 1;
  min-width: 0;
}
.field-row {
  display: flex;
  gap: 0.9rem;
}
label {
  display: block;
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
  margin-bottom: 0.3rem;
}
select,
input {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--bg);
  font: inherit;
  font-size: 0.88rem;
  padding: 0.55rem 0.7rem;
  color: var(--text);
}
input:disabled {
  opacity: 0.75;
  cursor: not-allowed;
}
.input-suffix {
  display: flex;
  align-items: center;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--bg);
  overflow: hidden;
}
.input-suffix input {
  flex: 1;
  border: none;
  border-radius: 0;
}
.input-suffix span {
  padding: 0 0.7rem;
  font-size: 0.78rem;
  color: var(--muted);
  white-space: nowrap;
}
.field-hint {
  margin: 0.4rem 0 0;
  font-size: 0.76rem;
  color: var(--muted);
  line-height: 1.4;
}
.link-btn {
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  font-size: 0.76rem;
  font-weight: 600;
  color: var(--accent);
  cursor: pointer;
  text-decoration: underline;
}
.tax-note {
  margin: -0.3rem 0 1rem;
  font-size: 0.74rem;
  color: var(--muted);
  line-height: 1.4;
}

.calc-btn {
  width: 100%;
  padding: 0.7rem;
  border: none;
  border-radius: 7px;
  background: var(--accent);
  color: #fff;
  font: inherit;
  font-weight: 700;
  font-size: 0.92rem;
  cursor: pointer;
}
.calc-btn:hover {
  filter: brightness(1.08);
}

.result-panel {
  min-height: 100%;
}
.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 220px;
  text-align: center;
}
.placeholder p {
  margin: 0;
  font-size: 0.88rem;
  color: var(--muted);
  max-width: 260px;
}
.result-eyebrow {
  margin: 0 0 0.2rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted);
}
.result-value {
  margin: 0 0 1rem;
  font-size: 1.7rem;
  font-weight: 800;
  color: var(--accent);
}
.breakdown {
  border-top: 1px solid var(--border);
  padding-top: 0.8rem;
  margin-bottom: 1rem;
}
.row {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
  padding: 0.35rem 0;
}
.row span {
  color: var(--muted);
}
.row.total {
  border-top: 1px solid var(--border);
  margin-top: 0.3rem;
  padding-top: 0.6rem;
  font-weight: 700;
}
.neg {
  color: #c0392b;
}
[data-theme='dark'] .neg {
  color: #ff8a80;
}

.chart-title {
  margin: 0 0 0.6rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--muted);
}
.bar-chart {
  display: flex;
  align-items: flex-end;
  gap: 0.4rem;
  height: 90px;
  background: var(--bg);
  border-radius: 8px;
  padding: 0.6rem;
}
.bar {
  flex: 1;
  background: var(--accent);
  border-radius: 3px 3px 0 0;
  min-height: 4px;
  opacity: 0.85;
}
.bar:last-child {
  opacity: 1;
}
.bar-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--muted);
  margin-top: 0.4rem;
}
.payout-note {
  font-size: 0.85rem;
  color: var(--text);
  line-height: 1.6;
  background: var(--bg);
  border-radius: 8px;
  padding: 0.9rem 1rem;
}
.payout-note strong {
  color: var(--accent);
}

.cta-bar {
  margin: 1.5rem 0 2.5rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 0.9rem 1.2rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  font-size: 0.85rem;
}
.cta-bar span {
  color: var(--muted);
}
.cta-bar a {
  color: var(--accent);
  font-weight: 600;
  text-decoration: none;
}
.cta-bar a:hover {
  text-decoration: underline;
}
</style>
