<script setup lang="ts">
import { emiPayment, fmtLkr } from '~/utils/fdCompare'
import { amortizationSchedule } from '~/utils/calculatorMath'
import type { ProductRate } from '~/composables/useRatesApi'

useHead({ title: 'Loan EMI Calculator — FindRate LK' })

const { fetchLoans } = useRatesApi()
const rows = ref<ProductRate[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    rows.value = (await fetchLoans().catch(() => [])).filter((r) => r.category_code === 'HOUSING_LOAN' || r.category_code === 'PERSONAL_LOAN')
  } finally {
    loading.value = false
  }
})

interface ProductOption {
  key: string
  label: string
  row: ProductRate
}

// Real product picker. Unlike Fixed Deposits, loan rows don't carry a
// clean numeric tenure (only free-text bands like "Up to 10 years"), so
// selecting one locks the Interest Rate to the bank's real disclosed rate
// but leaves Loan Tenure as a user-chosen assumption for the projection —
// there's no single real number to lock it to.
const productOptions = computed<ProductOption[]>(() => {
  return rows.value
    .map((r, i): ProductOption => {
      const variant = [r.tenure_label, r.rate_label].filter(Boolean).join(' — ')
      const label = `${r.bank_name} — ${formatCategoryLabel(r.category_code)}${variant ? ` — ${variant}` : ''} — ${r.interest_rate.toFixed(2)}%`
      return { key: `${i}`, label, row: r }
    })
    .sort((a, b) => a.label.localeCompare(b.label))
})

const selectedKey = ref('')
const selectedOption = computed(() => productOptions.value.find((o) => o.key === selectedKey.value) ?? null)
const isCustom = computed(() => !selectedOption.value)

const principal = ref(5000000)
const rate = ref(12.5)
const tenureYears = ref(20)

watch(selectedOption, (opt) => {
  if (opt) rate.value = opt.row.interest_rate
})

function useCustom() {
  selectedKey.value = ''
}

const hasCalculated = ref(false)
function calculate() {
  hasCalculated.value = true
}

const tenureMonths = computed(() => Math.round(tenureYears.value * 12))
const emi = computed(() => emiPayment(principal.value, rate.value, tenureMonths.value))
const totalPayable = computed(() => emi.value * tenureMonths.value)
const totalInterest = computed(() => totalPayable.value - principal.value)
const schedule = computed(() => amortizationSchedule(principal.value, rate.value, tenureMonths.value, 5))

// Simple donut split (principal vs interest of total payable) — a visual
// of the real computed numbers above, not a separate claim.
const principalShare = computed(() => (totalPayable.value > 0 ? (principal.value / totalPayable.value) * 100 : 0))
</script>

<template>
  <div>
    <SiteNav />

    <main>
      <div class="wrap">
        <p class="eyebrow">Calculators</p>
        <h1>Loan EMI Calculator</h1>
        <p class="sub">Compute your equated monthly installment and view an amortization timeline for housing and personal loans.</p>

        <div class="calc-grid">
          <div class="panel">
            <h2>Loan Parameters</h2>

            <div class="field">
              <label for="product">Select a Product (optional)</label>
              <select id="product" v-model="selectedKey">
                <option value="">Custom (enter manually)</option>
                <option v-for="o in productOptions" :key="o.key" :value="o.key">{{ o.label }}</option>
              </select>
              <p v-if="loading" class="field-hint">Loading real products…</p>
              <p v-else-if="selectedOption" class="field-hint">
                Rate locked to {{ selectedOption.row.bank_name }}'s real disclosed rate, updated {{ fmtRelativeDate(selectedOption.row.scraped_at) }}.
                Their real term is a band (<em>{{ selectedOption.row.tenure_label || 'not specified' }}</em>) rather than a single number — set Loan Tenure yourself below.
                <button type="button" class="link-btn" @click="useCustom">Use custom instead</button>
              </p>
            </div>

            <div class="field">
              <label for="amount">Loan Amount</label>
              <div class="input-suffix">
                <input id="amount" v-model.number="principal" type="number" min="0" step="50000">
                <span>LKR</span>
              </div>
            </div>

            <div class="field-row">
              <div class="field">
                <label for="rate">Interest Rate (Annual)</label>
                <div class="input-suffix">
                  <input id="rate" v-model.number="rate" type="number" min="0" step="0.1" :disabled="!isCustom">
                  <span>%</span>
                </div>
              </div>
              <div class="field">
                <label for="tenure">Loan Tenure</label>
                <div class="input-suffix">
                  <input id="tenure" v-model.number="tenureYears" type="number" min="1" max="30" step="1">
                  <span>Years</span>
                </div>
              </div>
            </div>

            <button type="button" class="calc-btn" @click="calculate">Calculate EMI</button>
          </div>

          <div class="panel result-panel">
            <template v-if="!hasCalculated">
              <div class="placeholder">
                <p>Fill in the details on the left and click <strong>Calculate EMI</strong> to see your projected repayments.</p>
              </div>
            </template>
            <template v-else>
              <div class="result-head">
                <div>
                  <p class="result-eyebrow">Monthly EMI (Installment)</p>
                  <p class="result-value">{{ fmtLkr(emi) }}</p>
                </div>
                <div class="donut" :style="{ '--pct': principalShare }" :title="`${principalShare.toFixed(0)}% principal, ${(100 - principalShare).toFixed(0)}% interest`" />
              </div>

              <div class="stat-pair">
                <div>
                  <span class="stat-label">Total Interest Payable</span>
                  <strong class="stat-value">{{ fmtLkr(totalInterest) }}</strong>
                </div>
                <div>
                  <span class="stat-label">Total Amount Payable</span>
                  <strong class="stat-value">{{ fmtLkr(totalPayable) }}</strong>
                </div>
              </div>

              <p class="chart-title">Amortization Schedule (First {{ schedule.length }} Year{{ schedule.length === 1 ? '' : 's' }})</p>
              <div class="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Year</th>
                      <th>Principal Paid</th>
                      <th>Interest Paid</th>
                      <th>Remaining Balance</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="y in schedule" :key="y.year">
                      <td>Year {{ y.year }}</td>
                      <td>{{ fmtLkr(y.principalPaid) }}</td>
                      <td>{{ fmtLkr(y.interestPaid) }}</td>
                      <td>{{ fmtLkr(y.remainingBalance) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <p class="disclaimer">EMI calculations use standard reducing-balance amortization math. Actual lending terms may include processing fees, stamp duties, insurance, and floating vs. fixed rate structures not reflected here — confirm with the bank before deciding.</p>
            </template>
          </div>
        </div>

        <div class="cta-bar">
          <span>Planning your borrowing?</span>
          <NuxtLink to="/rates?tab=loans">Compare personal and housing loan rates from all tracked banks &rarr;</NuxtLink>
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
  grid-template-columns: 1fr 1.2fr;
  gap: 1.2rem;
  align-items: start;
}
@media (max-width: 780px) {
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
  line-height: 1.45;
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
  max-width: 280px;
}

.result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
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
  margin: 0;
  font-size: 1.7rem;
  font-weight: 800;
  color: var(--accent);
}
.donut {
  --pct: 70;
  flex: none;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: conic-gradient(var(--accent) calc(var(--pct) * 1%), var(--border) 0);
}

.stat-pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.8rem;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  padding: 0.8rem 0;
  margin-bottom: 1rem;
}
.stat-label {
  display: block;
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
  margin-bottom: 0.2rem;
}
.stat-value {
  font-size: 0.95rem;
  font-weight: 700;
}

.chart-title {
  margin: 0 0 0.6rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--muted);
}
.table-wrap {
  overflow-x: auto;
  margin-bottom: 1rem;
}
table {
  width: 100%;
  border-collapse: collapse;
  min-width: 380px;
  font-size: 0.82rem;
}
th,
td {
  padding: 0.5rem 0.6rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
}
th {
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
}
tbody tr:last-child td {
  border-bottom: none;
}
.disclaimer {
  font-size: 0.74rem;
  color: var(--muted);
  line-height: 1.5;
  background: var(--bg);
  border-radius: 8px;
  padding: 0.8rem 0.9rem;
  margin: 0;
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
