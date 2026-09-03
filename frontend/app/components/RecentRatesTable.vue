<script setup lang="ts">
import type { ProductRate } from '~/composables/useRatesApi'

interface RecentRow extends ProductRate {
  kind: 'Fixed Deposit' | 'Savings' | 'Loan'
}

const { fetchFixedDeposits, fetchSavings, fetchLoans } = useRatesApi()

const rows = ref<RecentRow[]>([])
const loading = ref(true)
const failed = ref(false)

function productLabel(r: RecentRow): string {
  if (r.kind === 'Fixed Deposit') {
    return `${fmtTenure(r.tenure_value ?? 0)} Fixed Deposit`
  }
  return r.product_name + (r.tenure_label ? ` (${r.tenure_label})` : '')
}

onMounted(async () => {
  try {
    const [fd, savings, loans] = await Promise.all([
      fetchFixedDeposits().catch(() => []),
      fetchSavings().catch(() => []),
      fetchLoans().catch(() => [])
    ])

    const tagged: RecentRow[] = [
      ...fd.map((r) => ({ ...r, kind: 'Fixed Deposit' as const })),
      ...savings.map((r) => ({ ...r, kind: 'Savings' as const })),
      ...loans.map((r) => ({ ...r, kind: 'Loan' as const }))
    ]

    // A pure "most recent scraped_at" sort tends to cluster on whichever
    // bank/product-type happened to be scraped last, showing 8 near-
    // identical rows from one bank. Taking the single most recent row per
    // (bank, product type) first gives a representative spread instead.
    const latestPerGroup = new Map<string, RecentRow>()
    for (const row of tagged) {
      const key = `${row.bank_name}|${row.kind}`
      const existing = latestPerGroup.get(key)
      if (!existing || new Date(row.scraped_at) > new Date(existing.scraped_at)) {
        latestPerGroup.set(key, row)
      }
    }

    rows.value = [...latestPerGroup.values()]
      .sort((a, b) => new Date(b.scraped_at).getTime() - new Date(a.scraped_at).getTime())
      .slice(0, 8)
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Bank</th>
          <th>Product</th>
          <th>Rate</th>
          <th>Last updated</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td colspan="5" class="empty">Loading…</td>
        </tr>
        <tr v-else-if="failed || rows.length === 0">
          <td colspan="5" class="empty">No data yet — the scraper hasn't run against this database.</td>
        </tr>
        <tr v-for="r in rows" :key="`${r.kind}-${r.id}`">
          <td>
            <span class="bank-cell">
              <span class="bank-avatar">{{ r.bank_code.charAt(0) }}</span>{{ r.bank_name }}
            </span>
          </td>
          <td>{{ productLabel(r) }}</td>
          <td class="rate">{{ r.interest_rate.toFixed(2) }}% p.a.</td>
          <td>{{ fmtRelativeDate(r.scraped_at) }}</td>
          <td class="action">
            <a v-if="r.source_url" :href="r.source_url" target="_blank" rel="noopener">Verify &rarr;</a>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.88rem;
  min-width: 600px;
}
th,
td {
  padding: 0.7rem 0.9rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
th {
  color: var(--muted);
  font-weight: 600;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
tbody tr:last-child td {
  border-bottom: none;
}
td.rate {
  font-weight: 700;
  text-align: right;
}
td.action {
  text-align: right;
}
td.action a {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
}
td.action a:hover {
  text-decoration: underline;
}
.bank-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.bank-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--accent);
  color: #fff;
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 0.68rem;
  font-weight: 700;
}
.empty {
  text-align: center;
  color: var(--muted);
  padding: 2rem;
  white-space: normal;
}
</style>
