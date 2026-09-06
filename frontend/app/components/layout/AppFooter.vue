<script setup lang="ts">
// New Tailwind-based footer matching the approved Figma. Scoped to the
// homepage for now, alongside AppHeader.vue — see its top comment.
const LINK_COLUMNS = [
  {
    label: 'Directory',
    links: [
      { label: 'All Banks', href: '/banks' },
      { label: 'Fixed Deposits', href: '/products/fixed-deposits' },
      { label: 'Housing Loans', href: '/compare/housing-loans' },
      { label: 'Savings Accounts', href: '/compare/savings-accounts' },
      { label: 'All Products', href: '/products' }
    ]
  },
  {
    label: 'Tools',
    links: [
      { label: 'Compare Rates', href: '/rates' },
      { label: 'Fixed Deposit Calculator', href: '/calculators/fixed-deposit' },
      { label: 'Loan EMI Calculator', href: '/calculators/loan-emi' }
    ]
  },
  {
    label: 'System',
    links: [
      { label: 'Data', href: '/data' },
      { label: 'About Us', href: 'https://github.com/RWageesha/sl-finance-compare#readme', external: true },
      { label: 'Contact Us', href: 'https://github.com/RWageesha/sl-finance-compare#readme', external: true }
    ]
  }
]

// Only GitHub is a real account for this project — X/LinkedIn have no
// destination yet, shown for visual completeness but not real links
// (same honesty convention as the current SiteFooter.vue).
const SOCIALS = [
  { label: 'X', href: null },
  { label: 'LinkedIn', href: null },
  { label: 'GitHub', href: 'https://github.com/RWageesha/sl-finance-compare' }
]
</script>

<template>
  <footer class="bg-navy">
    <div class="mx-auto max-w-[1200px] px-4 py-10 sm:px-6 sm:py-14">
      <div class="grid grid-cols-1 gap-10 md:grid-cols-[1.4fr_1fr_1fr_1fr]">
        <div>
          <NuxtLink to="/" class="flex items-center">
            <img src="/brand/logo-white.png" alt="FindRate LK" class="h-12 w-auto sm:h-14">
          </NuxtLink>
          <p class="mt-4 max-w-xs text-sm leading-relaxed text-white/60">
            A public, transparent directory of financial products and current interest rates across licensed Sri
            Lankan banks. Independently verified, not sponsored.
          </p>
        </div>

        <div v-for="col in LINK_COLUMNS" :key="col.label">
          <h4 class="text-xs font-bold uppercase tracking-wider text-white/40">{{ col.label }}</h4>
          <ul class="mt-4 space-y-3">
            <li v-for="link in col.links" :key="link.label">
              <a
                v-if="'external' in link && link.external"
                :href="link.href"
                target="_blank"
                rel="noopener"
                class="text-sm text-white/80 transition hover:text-[#3B82F6]"
              >{{ link.label }}</a>
              <NuxtLink v-else :to="link.href" class="text-sm text-white/80 transition hover:text-[#3B82F6]">
                {{ link.label }}
              </NuxtLink>
            </li>
          </ul>
        </div>
      </div>

      <div class="mt-12 flex flex-col items-center gap-4 border-t border-white/10 pt-6 text-xs text-white/50 md:flex-row md:justify-between">
        <span>&copy; 2026 FindRate LK. All rights reserved.</span>
        <div class="flex items-center gap-5">
          <span class="cursor-default opacity-60" title="Coming soon">Privacy Policy</span>
          <span class="cursor-default opacity-60" title="Coming soon">Disclaimer</span>
        </div>
        <div class="flex items-center gap-3">
          <a
            v-for="s in SOCIALS"
            :key="s.label"
            :href="s.href ?? undefined"
            :target="s.href ? '_blank' : undefined"
            rel="noopener"
            :aria-label="s.label"
            :title="s.href ? s.label : `No ${s.label} account yet`"
            class="flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-[11px] font-bold text-white/80 transition"
            :class="s.href ? 'hover:bg-primary hover:text-white' : 'cursor-default opacity-40'"
          >
            <svg v-if="s.label === 'GitHub'" viewBox="0 0 24 24" fill="currentColor" class="h-4 w-4">
              <path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.57.1.78-.25.78-.55 0-.27-.01-1.16-.02-2.11-3.2.7-3.87-1.36-3.87-1.36-.53-1.33-1.28-1.69-1.28-1.69-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.76 2.7 1.25 3.36.96.1-.75.4-1.25.73-1.54-2.56-.29-5.25-1.28-5.25-5.7 0-1.26.45-2.29 1.19-3.09-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 5.79 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.12 3.05.74.8 1.18 1.83 1.18 3.09 0 4.43-2.69 5.41-5.26 5.69.42.36.78 1.08.78 2.17 0 1.57-.01 2.83-.01 3.22 0 .3.2.66.79.55A10.53 10.53 0 0 0 23.5 12c0-6.35-5.15-11.5-11.5-11.5Z"/>
            </svg>
            <span v-else>{{ s.label === 'LinkedIn' ? 'in' : 'X' }}</span>
          </a>
        </div>
      </div>
    </div>
  </footer>
</template>
