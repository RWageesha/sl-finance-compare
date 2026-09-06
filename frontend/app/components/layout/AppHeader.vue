<script setup lang="ts">
// New Tailwind-based site header matching the approved Figma exactly —
// dark navy bar, white logo, and every top-level nav item opens a
// dropdown. Dropdown CONTENT below is a reasonable stub (real pages that
// already exist elsewhere in the app); replace with final copy once
// content specs are given.
//
// Scoped to the homepage for now (pages/index.vue) — the rest of the site
// still uses SiteNav.vue's light-theme header. Swapping every page over
// is a separate pass.
defineEmits<{ 'focus-search': [] }>()

interface NavChild {
  label: string
  href: string
}
interface NavItem {
  label: string
  href?: string
  children: NavChild[]
}

const NAV_ITEMS: NavItem[] = [
  {
    label: 'Banks',
    href: '/banks',
    children: [
      { label: 'All Banks', href: '/banks' },
      { label: 'Bank of Ceylon', href: '/banks/boc' },
      { label: 'Commercial Bank', href: '/banks/combank' },
      { label: 'Sampath Bank', href: '/banks/sampath' },
      { label: 'HNB', href: '/banks/hnb' }
    ]
  },
  {
    label: 'Products',
    href: '/products',
    children: [
      { label: 'Fixed Deposits', href: '/products/fixed-deposits' },
      { label: 'Savings Accounts', href: '/products/savings-accounts' },
      { label: 'Housing Loans', href: '/products/housing-loans' },
      { label: 'Personal Loans', href: '/products/personal-loans' },
      { label: 'Gold Loans', href: '/products/gold-loans' },
      { label: 'Credit Cards', href: '/products/credit-cards' },
      { label: 'Debit Cards', href: '/products/debit-cards' }
    ]
  },
  {
    label: 'Compare',
    href: '/#comparisons',
    children: [
      { label: 'Fixed Deposit Rates', href: '/products/fixed-deposits' },
      { label: 'Housing Loan Rates', href: '/compare/housing-loans' },
      { label: 'Savings Account Rates', href: '/compare/savings-accounts' }
    ]
  },
  {
    label: 'Calculators',
    children: [
      { label: 'Fixed Deposit Calculator', href: '/calculators/fixed-deposit' },
      { label: 'Loan EMI Calculator', href: '/calculators/loan-emi' }
    ]
  },
  {
    label: 'Data',
    href: '/rates',
    children: [
      { label: 'All Rates', href: '/rates' },
      { label: 'Rate History', href: '/products' }
    ]
  }
]

const openIndex = ref<number | null>(null)
const navEl = ref<HTMLElement | null>(null)
const mobileOpen = ref(false)
const mobileExpanded = ref<number | null>(null)
const mobileSearchQuery = ref('')

function onMobileSearch() {
  navigateTo(resolveSearchRoute(mobileSearchQuery.value))
  mobileOpen.value = false
}

function toggle(i: number) {
  openIndex.value = openIndex.value === i ? null : i
}

function toggleMobile(i: number) {
  mobileExpanded.value = mobileExpanded.value === i ? null : i
}

function onDocClick(e: MouseEvent) {
  if (navEl.value && !navEl.value.contains(e.target as Node)) openIndex.value = null
}

onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))

const route = useRoute()
watch(() => route.fullPath, () => {
  openIndex.value = null
  mobileOpen.value = false
  mobileExpanded.value = null
})
</script>

<template>
  <header class="bg-navy">
    <div class="mx-auto grid h-20 max-w-[1200px] grid-cols-[auto_1fr_auto] items-center gap-4 px-4 sm:px-6 md:grid-cols-[1fr_auto_1fr]">
      <NuxtLink to="/" class="flex shrink-0 items-center justify-self-start">
        <img src="/brand/logo-white.png" alt="FindRate LK" class="h-12 w-auto sm:h-14">
      </NuxtLink>

      <nav ref="navEl" class="col-start-2 hidden items-center justify-self-center gap-7 md:flex">
        <div v-for="(item, i) in NAV_ITEMS" :key="item.label" class="relative">
          <button
            type="button"
            class="flex items-center gap-1 text-sm text-white/85 transition hover:text-white"
            :aria-expanded="openIndex === i"
            @click="toggle(i)"
          >
            {{ item.label }}
            <Icon name="chevron-down" class="h-3 w-3" />
          </button>

          <div
            v-if="openIndex === i"
            class="absolute left-1/2 top-[calc(100%+0.75rem)] z-20 min-w-[220px] -translate-x-1/2 rounded-card border border-card-border bg-card py-2 shadow-xl"
          >
            <NuxtLink
              v-for="child in item.children"
              :key="child.href"
              :to="child.href"
              class="block px-4 py-2 text-sm text-navy transition hover:bg-page hover:text-primary"
            >
              {{ child.label }}
            </NuxtLink>
          </div>
        </div>
      </nav>

      <div class="col-start-2 flex justify-self-end md:col-start-3">
        <button
          type="button"
          aria-label="Search"
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-white/20 text-white transition hover:bg-white/10"
          @click="$emit('focus-search')"
        >
          <Icon name="search" class="h-4 w-4" />
        </button>

        <button
          type="button"
          aria-label="Toggle menu"
          :aria-expanded="mobileOpen"
          class="ml-2 flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-white/20 text-white transition hover:bg-white/10 md:hidden"
          @click="mobileOpen = !mobileOpen"
        >
          <Icon :name="mobileOpen ? 'close' : 'menu'" class="h-4 w-4" />
        </button>
      </div>
    </div>

    <div v-if="mobileOpen" class="border-t border-white/10 px-4 py-3 md:hidden">
      <form class="mb-3 flex items-center gap-2 rounded-pill bg-white/10 px-4 py-2.5" @submit.prevent="onMobileSearch">
        <Icon name="search" class="h-4 w-4 shrink-0 text-white/50" />
        <input
          v-model="mobileSearchQuery"
          type="text"
          placeholder="Search for products, banks, or rates..."
          autocomplete="off"
          class="min-w-0 flex-1 border-none bg-transparent text-sm text-white outline-none placeholder:text-white/50"
        >
      </form>
      <div v-for="(item, i) in NAV_ITEMS" :key="item.label" class="border-b border-white/10 last:border-none">
        <button
          type="button"
          class="flex w-full items-center justify-between py-3 text-sm font-medium text-white/85"
          :aria-expanded="mobileExpanded === i"
          @click="toggleMobile(i)"
        >
          {{ item.label }}
          <Icon name="chevron-down" class="h-3 w-3 transition" :class="{ 'rotate-180': mobileExpanded === i }" />
        </button>
        <div v-if="mobileExpanded === i" class="flex flex-col gap-1 pb-3 pl-3">
          <NuxtLink
            v-for="child in item.children"
            :key="child.href"
            :to="child.href"
            class="py-1.5 text-sm text-white/60 transition hover:text-white"
          >
            {{ child.label }}
          </NuxtLink>
        </div>
      </div>
    </div>
  </header>
</template>
