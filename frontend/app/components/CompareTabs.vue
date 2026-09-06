<script setup lang="ts">
// Shared tab bar linking between the category-specific comparison pages.
// Fixed Deposits isn't included — it has its own distinct flow (amount +
// tenure input, cards, table view) that doesn't fit this single-list
// pattern, and it's already reachable from its own entry points.
defineProps<{
  active: 'housing-loans' | 'personal-loans' | 'gold-loans' | 'savings'
}>()

const TABS = [
  { id: 'housing-loans', label: 'Housing Loans', href: '/compare/housing-loans' },
  { id: 'personal-loans', label: 'Personal Loans', href: '/compare/personal-loans' },
  { id: 'gold-loans', label: 'Gold Loans', href: '/compare/gold-loans' },
  { id: 'savings', label: 'Savings Accounts', href: '/compare/savings-accounts' }
] as const
</script>

<template>
  <nav class="compare-tabs">
    <NuxtLink v-for="tab in TABS" :key="tab.id" :to="tab.href" class="tab" :class="{ active: tab.id === active }">
      {{ tab.label }}
    </NuxtLink>
  </nav>
</template>

<style scoped>
.compare-tabs {
  display: flex;
  gap: 1.5rem;
  border-bottom: 1px solid var(--border);
  margin-bottom: 1.3rem;
  flex-wrap: wrap;
}
.tab {
  padding: 0.7rem 0.1rem;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--muted);
  text-decoration: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab:hover {
  color: var(--text);
}
.tab.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}
</style>
