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
    | 'check'
    | 'menu'
    | 'close'
    | 'database'
    | 'tag'
    | 'flag'
    | 'list'
    | 'gear'
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
    <polyline v-else-if="name === 'check'" points="20,6 9,17 4,12" />
    <g v-else-if="name === 'menu'">
      <line x1="3" y1="6" x2="21" y2="6" />
      <line x1="3" y1="12" x2="21" y2="12" />
      <line x1="3" y1="18" x2="21" y2="18" />
    </g>
    <g v-else-if="name === 'close'">
      <line x1="6" y1="6" x2="18" y2="18" />
      <line x1="18" y1="6" x2="6" y2="18" />
    </g>
    <!-- Admin sidebar: Data Sources -->
    <g v-else-if="name === 'database'">
      <ellipse cx="12" cy="5" rx="8" ry="3" />
      <path d="M4 5v6c0 1.66 3.58 3 8 3s8-1.34 8-3V5" />
      <path d="M4 11v6c0 1.66 3.58 3 8 3s8-1.34 8-3v-6" />
    </g>
    <!-- Admin sidebar: Products -->
    <g v-else-if="name === 'tag'">
      <path d="M20.6 12.6 12.7 20.5a2 2 0 0 1-2.8 0l-6.4-6.4a2 2 0 0 1 0-2.8L11.4 3.3a2 2 0 0 1 1.4-.6H19a2 2 0 0 1 2 2v6.5a2 2 0 0 1-.4 1.8Z" />
      <circle cx="16" cy="8" r="1.5" />
    </g>
    <!-- Admin sidebar: User Reports -->
    <g v-else-if="name === 'flag'">
      <path d="M5 21V4" />
      <path d="M5 4h13l-3 4.5L18 13H5" />
    </g>
    <!-- Admin sidebar: Audit Logs -->
    <g v-else-if="name === 'list'">
      <line x1="8" y1="6" x2="21" y2="6" />
      <line x1="8" y1="12" x2="21" y2="12" />
      <line x1="8" y1="18" x2="21" y2="18" />
      <line x1="3" y1="6" x2="3.01" y2="6" />
      <line x1="3" y1="12" x2="3.01" y2="12" />
      <line x1="3" y1="18" x2="3.01" y2="18" />
    </g>
    <!-- Admin sidebar: Settings -->
    <g v-else-if="name === 'gear'">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.6 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.6a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
    </g>
  </svg>
</template>
