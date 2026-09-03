<script setup lang="ts">
const emit = defineEmits<{ close: [] }>()

const TENURES = [12, 24, 36, 60, 120, 180, 240, 360]

const principal = ref(1000000)
const rate = ref(12.5)
const tenure = ref(60)
const result = ref<number | null>(null)

// Standard reducing-balance EMI formula: P * r * (1+r)^n / ((1+r)^n - 1),
// with r as the monthly rate and n the number of monthly instalments.
function calculate() {
  const p = Number(principal.value) || 0
  const annualRate = Number(rate.value) || 0
  const n = Number(tenure.value) || 0
  const r = annualRate / 100 / 12
  if (n <= 0) {
    result.value = 0
    return
  }
  if (r === 0) {
    result.value = p / n
    return
  }
  const factor = Math.pow(1 + r, n)
  result.value = (p * r * factor) / (factor - 1)
}

function fmtLkr(n: number) {
  return 'Rs. ' + Math.round(n).toLocaleString('en-LK')
}
</script>

<template>
  <CalculatorModal eyebrow="Loan" title="Loan EMI Calculator" @close="emit('close')">
    <div class="field">
      <label for="loan-principal">Loan Amount</label>
      <div class="input-suffix">
        <input id="loan-principal" v-model.number="principal" type="number" min="0" step="1000">
        <span>LKR</span>
      </div>
    </div>
    <div class="field">
      <label for="loan-rate">Interest Rate (Annual)</label>
      <div class="input-suffix">
        <input id="loan-rate" v-model.number="rate" type="number" min="0" step="0.1">
        <span>%</span>
      </div>
    </div>
    <div class="field">
      <label for="loan-tenure">Tenure</label>
      <select id="loan-tenure" v-model.number="tenure">
        <option v-for="m in TENURES" :key="m" :value="m">{{ m }} Months</option>
      </select>
    </div>
    <button type="button" class="calc-btn" @click="calculate">Calculate EMI</button>

    <div v-if="result !== null" class="result">
      <p class="result-label">Estimated Monthly Payment</p>
      <p class="result-value">{{ fmtLkr(result) }}</p>
      <NuxtLink to="/rates?tab=loans" @click="emit('close')">See full calculator &amp; compare banks &rarr;</NuxtLink>
    </div>
  </CalculatorModal>
</template>

<style scoped>
.field {
  margin-bottom: 0.9rem;
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
}
.input-suffix span {
  padding: 0 0.7rem;
  font-size: 0.78rem;
  color: var(--muted);
  white-space: nowrap;
}
input,
select {
  width: 100%;
  border: none;
  background: transparent;
  font: inherit;
  font-size: 0.88rem;
  padding: 0.55rem 0.7rem;
  color: var(--text);
  outline: none;
}
select {
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--bg);
}
.calc-btn {
  width: 100%;
  margin-top: 0.3rem;
  padding: 0.65rem;
  border: none;
  border-radius: 7px;
  background: var(--accent);
  color: #fff;
  font: inherit;
  font-weight: 600;
  font-size: 0.88rem;
  cursor: pointer;
}
.calc-btn:hover {
  filter: brightness(1.08);
}
.result {
  margin-top: 1.1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border);
}
.result-label {
  margin: 0 0 0.25rem;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted);
}
.result-value {
  margin: 0 0 0.5rem;
  font-size: 1.3rem;
  font-weight: 800;
  color: var(--accent);
}
.result a {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
}
.result a:hover {
  text-decoration: underline;
}
</style>
