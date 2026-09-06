<script setup lang="ts">
// Placeholder rows per tab for this visual pass — real API data (the Go
// backend already has fetchFixedDeposits/fetchSavings/fetchLoans via
// useRatesApi, see RecentRatesTable.vue) gets wired in a follow-up pass.
interface StubRow {
  bank: string
  letter: string
  color: string
  product: string
  rate: string
  updated: string
}

const TABS = [
  { key: 'fd', label: 'Fixed Deposits' },
  { key: 'savings', label: 'Savings' },
  { key: 'loans', label: 'Loan Products' },
  { key: 'cards', label: 'Card Tariffs' }
] as const

const ROWS: Record<(typeof TABS)[number]['key'], StubRow[]> = {
  fd: [
    { bank: 'HNB', letter: 'H', color: '#1E3A8A', product: '12-Month Fixed Deposit (At Maturity)', rate: '11.75%', updated: '2 hours ago' },
    { bank: 'Commercial Bank', letter: 'C', color: '#2563EB', product: '12-Month Fixed Deposit (At Maturity)', rate: '11.50%', updated: '3 hours ago' },
    { bank: 'Sampath Bank', letter: 'S', color: '#EA580C', product: '12-Month Fixed Deposit (Monthly Interest)', rate: '11.25%', updated: '4 hours ago' },
    { bank: 'NDB Bank', letter: 'N', color: '#0EA5E9', product: '12-Month Fixed Deposit (At Maturity)', rate: '11.60%', updated: '6 hours ago' },
    { bank: 'Seylan Bank', letter: 'S', color: '#0F172A', product: '12-Month Fixed Deposit (At Maturity)', rate: '11.00%', updated: '6 hours ago' }
  ],
  savings: [
    { bank: 'HNB', letter: 'H', color: '#1E3A8A', product: 'Standard Savings Account', rate: '4.50%', updated: '2 hours ago' },
    { bank: 'Commercial Bank', letter: 'C', color: '#2563EB', product: 'Standard Savings Account', rate: '4.25%', updated: '3 hours ago' },
    { bank: 'Sampath Bank', letter: 'S', color: '#EA580C', product: "Senior Citizens' Savings", rate: '5.00%', updated: '4 hours ago' },
    { bank: 'Bank of Ceylon', letter: 'B', color: '#CA8A04', product: 'Standard Savings Account', rate: '4.00%', updated: '5 hours ago' },
    { bank: "People's Bank", letter: 'P', color: '#7C2D12', product: 'Standard Savings Account', rate: '4.10%', updated: '7 hours ago' }
  ],
  loans: [
    { bank: 'HNB', letter: 'H', color: '#1E3A8A', product: 'Housing Loan', rate: '13.50%', updated: '3 hours ago' },
    { bank: 'Commercial Bank', letter: 'C', color: '#2563EB', product: 'Personal Loan', rate: '15.00%', updated: '4 hours ago' },
    { bank: 'Bank of Ceylon', letter: 'B', color: '#CA8A04', product: 'Housing Loan', rate: '13.75%', updated: '5 hours ago' },
    { bank: 'Sampath Bank', letter: 'S', color: '#EA580C', product: 'Gold Loan', rate: '11.00%', updated: '6 hours ago' },
    { bank: 'NDB Bank', letter: 'N', color: '#0EA5E9', product: 'Personal Loan', rate: '14.50%', updated: '8 hours ago' }
  ],
  cards: [
    { bank: 'HNB', letter: 'H', color: '#1E3A8A', product: 'Classic Credit Card — Annual Fee', rate: 'LKR 3,000', updated: '1 day ago' },
    { bank: 'Commercial Bank', letter: 'C', color: '#2563EB', product: 'Gold Credit Card — Annual Fee', rate: 'LKR 5,000', updated: '1 day ago' },
    { bank: 'Sampath Bank', letter: 'S', color: '#EA580C', product: 'Platinum Credit Card — Annual Fee', rate: 'LKR 7,500', updated: '2 days ago' },
    { bank: 'Bank of Ceylon', letter: 'B', color: '#CA8A04', product: 'Classic Credit Card — Annual Fee', rate: 'LKR 2,500', updated: '2 days ago' },
    { bank: "People's Bank", letter: 'P', color: '#7C2D12', product: 'Classic Credit Card — Annual Fee', rate: 'LKR 2,000', updated: '3 days ago' }
  ]
}

const activeTab = ref<(typeof TABS)[number]['key']>('fd')
const rows = computed(() => ROWS[activeTab.value])
</script>

<template>
  <section class="bg-page px-4 py-7 sm:px-6 sm:py-10">
    <div class="mx-auto max-w-[1080px] rounded-card border border-card-border bg-card p-4 shadow-sm sm:p-8">
      <span class="inline-block rounded-pill bg-badge-bg px-3 py-1 text-xs font-bold uppercase tracking-wider text-primary">
        Live Data
      </span>
      <h2 class="mt-3 text-[1.3rem] font-bold text-navy sm:text-[1.6rem]">Recently Updated Bank Rates</h2>

      <div class="mt-6 flex flex-wrap gap-2">
        <button
          v-for="tab in TABS"
          :key="tab.key"
          type="button"
          class="rounded-pill px-3 py-1.5 text-xs font-semibold transition sm:px-4 sm:text-sm"
          :class="activeTab === tab.key ? 'bg-primary text-white' : 'bg-page text-navy hover:bg-card-border'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="mt-6 overflow-x-auto">
        <table class="w-full min-w-[360px] border-collapse text-sm">
          <thead>
            <tr class="border-b border-card-border text-left text-xs uppercase tracking-wide text-muted">
              <th class="pb-3 font-semibold">Bank</th>
              <th class="hidden pb-3 font-semibold sm:table-cell">Product</th>
              <th class="pb-3 font-semibold">Rate (P.A.)</th>
              <th class="hidden pb-3 font-semibold sm:table-cell">Last Updated</th>
              <th class="pb-3 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in rows" :key="`${activeTab}-${i}`" class="border-b border-card-border last:border-none">
              <td class="py-3.5 pr-3 sm:pr-4">
                <span class="flex items-center gap-2 sm:gap-2.5">
                  <span
                    class="flex h-6 w-6 shrink-0 items-center justify-center rounded-[8px] text-[11px] font-bold text-white sm:h-7 sm:w-7 sm:text-xs"
                    :style="{ backgroundColor: row.color }"
                  >{{ row.letter }}</span>
                  <span class="truncate font-semibold text-navy">{{ row.bank }}</span>
                </span>
              </td>
              <td class="hidden py-3.5 pr-4 text-navy/80 sm:table-cell">{{ row.product }}</td>
              <td class="py-3.5 pr-3 font-bold text-navy sm:pr-4">{{ row.rate }}</td>
              <td class="hidden py-3.5 pr-4 text-muted sm:table-cell">
                {{ row.updated }}
                <span class="ml-1 inline-block h-1.5 w-1.5 rounded-full bg-emerald-500 align-middle" />
              </td>
              <td class="py-3.5">
                <span class="inline-flex items-center gap-1 rounded-pill bg-emerald-50 px-2 py-1 text-[11px] font-semibold text-emerald-600 sm:px-2.5 sm:text-xs">
                  <Icon name="check" class="h-3 w-3" />
                  Verified
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
