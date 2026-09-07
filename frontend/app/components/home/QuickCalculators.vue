<script setup lang="ts">
// Self-contained, like SiteFooter.vue's calculator triggers: owns its own
// modal open/closed state rather than taking it as a prop, since nothing
// else on the page needs to know about it.
const showFdCalc = ref(false)
const showLoanCalc = ref(false)

const CALCULATORS = [
  { icon: 'calculator' as const, title: 'Fixed Deposit Calculator', desc: "Maturity values calculated for all current bank rates.", action: () => (showFdCalc.value = true) },
  { icon: 'percent' as const, title: 'Loan EMI Calculator', desc: 'Analyze monthly repayment projections for any amount.', action: () => (showLoanCalc.value = true) }
]
</script>

<template>
  <section class="bg-page px-4 py-7 sm:px-6 sm:py-10">
    <div class="mx-auto max-w-[1080px]">
      <span class="inline-block rounded-pill bg-badge-bg px-3 py-1 text-xs font-bold uppercase tracking-wider text-primary">
        Calculators
      </span>
      <h2 class="mt-3 text-[1.3rem] font-bold text-navy sm:text-[1.6rem]">Quick Calculators</h2>
      <p class="mt-1 text-sm text-muted">Get an instant estimate without leaving the homepage.</p>

      <div class="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-[1.2fr_1fr]">
        <div class="flex flex-col gap-4">
          <button
            v-for="c in CALCULATORS"
            :key="c.title"
            type="button"
            class="flex flex-1 items-center gap-3 rounded-card border border-card-border bg-card p-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-primary hover:shadow-md sm:gap-3.5 sm:p-[18px]"
            @click="c.action"
          >
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-badge-bg text-primary sm:h-11 sm:w-11">
              <Icon :name="c.icon" class="h-4 w-4 sm:h-5 sm:w-5" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-sm font-bold text-navy sm:text-[15px]">{{ c.title }}</h3>
              <p class="mt-0.5 truncate text-xs text-muted">{{ c.desc }}</p>
            </div>
            <span class="shrink-0 text-xs font-semibold text-primary sm:text-sm">Calculate &rarr;</span>
          </button>
        </div>

        <div class="min-h-[180px] overflow-hidden rounded-card lg:min-h-0">
          <img src="/hero/calculator-section.jpg" alt="" class="h-full w-full object-cover">
        </div>
      </div>
    </div>

    <FdCalculatorModal v-if="showFdCalc" @close="showFdCalc = false" />
    <LoanCalculatorModal v-if="showLoanCalc" @close="showLoanCalc = false" />
  </section>
</template>
