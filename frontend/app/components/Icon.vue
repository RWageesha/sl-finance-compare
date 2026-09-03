<script setup lang="ts">
defineProps<{
  name:
    | 'bank'
    | 'home'
    | 'savings'
    | 'card'
    | 'coin'
    | 'search'
    | 'shield'
    | 'bar-chart'
    | 'line-chart'
    | 'bar-chart-alt'
    | 'calculator'
    | 'percent'
    | 'clock'
    | 'grid'
    | 'moon'
    | 'refresh'
    | 'chevron-down'
}>()
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <!-- Fixed Deposits: bank/institution columns -->
    <g v-if="name === 'bank'">
      <polygon points="12,3 21,9 3,9" />
      <line x1="5" y1="10" x2="5" y2="21" />
      <line x1="9" y1="10" x2="9" y2="21" />
      <line x1="15" y1="10" x2="15" y2="21" />
      <line x1="19" y1="10" x2="19" y2="21" />
      <line x1="3" y1="21" x2="21" y2="21" />
    </g>
    <!-- Housing Loans: house -->
    <g v-else-if="name === 'home'">
      <path d="M4 11L12 4l8 7" />
      <rect x="6" y="11" width="12" height="9" />
      <rect x="10" y="15" width="4" height="5" />
    </g>
    <!-- Savings Accounts: stacked coins -->
    <g v-else-if="name === 'savings'">
      <ellipse cx="12" cy="17" rx="7" ry="2.5" />
      <ellipse cx="12" cy="12.5" rx="7" ry="2.5" />
      <ellipse cx="12" cy="8" rx="7" ry="2.5" />
    </g>
    <!-- Personal Loans / Credit Cards -->
    <g v-else-if="name === 'card'">
      <rect x="2" y="5" width="20" height="14" rx="2" />
      <line x1="2" y1="10" x2="22" y2="10" />
    </g>
    <!-- Gold Loans: coin -->
    <g v-else-if="name === 'coin'">
      <circle cx="12" cy="12" r="9" />
      <circle cx="12" cy="12" r="5" />
    </g>
    <!-- Nav search button -->
    <g v-else-if="name === 'search'">
      <circle cx="11" cy="11" r="7" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </g>
    <!-- Trust banner -->
    <path v-else-if="name === 'shield'" d="M12 2 L20 6 L20 12 L12 21 L4 12 L4 6 Z" />
    <!-- Comparison card thumbnails -->
    <g v-else-if="name === 'bar-chart'">
      <rect x="3" y="14" width="3" height="7" />
      <rect x="9" y="9" width="3" height="12" />
      <rect x="15" y="5" width="3" height="16" />
    </g>
    <g v-else-if="name === 'bar-chart-alt'">
      <rect x="3" y="6" width="3" height="15" />
      <rect x="9" y="12" width="3" height="9" />
      <rect x="15" y="3" width="3" height="18" />
    </g>
    <polyline v-else-if="name === 'line-chart'" points="2,17 7,10 12,13 17,5 22,9" />
    <!-- Quick calculators section -->
    <g v-else-if="name === 'calculator'">
      <rect x="4" y="2" width="16" height="20" rx="2" />
      <line x1="8" y1="6" x2="16" y2="6" />
      <line x1="8" y1="11" x2="8.01" y2="11" />
      <line x1="12" y1="11" x2="12.01" y2="11" />
      <line x1="16" y1="11" x2="16.01" y2="11" />
      <line x1="8" y1="15" x2="8.01" y2="15" />
      <line x1="12" y1="15" x2="12.01" y2="15" />
      <line x1="16" y1="15" x2="16.01" y2="15" />
      <line x1="8" y1="19" x2="8.01" y2="19" />
      <line x1="12" y1="19" x2="12.01" y2="19" />
    </g>
    <g v-else-if="name === 'percent'">
      <line x1="19" y1="5" x2="5" y2="19" />
      <circle cx="6.5" cy="6.5" r="2.5" />
      <circle cx="17.5" cy="17.5" r="2.5" />
    </g>
    <!-- Bank directory page -->
    <g v-else-if="name === 'clock'">
      <circle cx="12" cy="12" r="9" />
      <polyline points="12,7 12,12 16,14" />
    </g>
    <g v-else-if="name === 'grid'">
      <rect x="3" y="3" width="7" height="7" rx="1.5" />
      <rect x="14" y="3" width="7" height="7" rx="1.5" />
      <rect x="3" y="14" width="7" height="7" rx="1.5" />
      <rect x="14" y="14" width="7" height="7" rx="1.5" />
    </g>
    <path v-else-if="name === 'moon'" d="M20 14.5A8.5 8.5 0 1 1 9.5 4a7 7 0 0 0 10.5 10.5Z" />
    <g v-else-if="name === 'refresh'">
      <path d="M3 12a9 9 0 0 1 15.3-6.4L21 8" />
      <polyline points="21,3 21,8 16,8" />
      <path d="M21 12a9 9 0 0 1-15.3 6.4L3 16" />
      <polyline points="3,21 3,16 8,16" />
    </g>
    <polyline v-else-if="name === 'chevron-down'" points="6,9 12,15 18,9" />
  </svg>
</template>
