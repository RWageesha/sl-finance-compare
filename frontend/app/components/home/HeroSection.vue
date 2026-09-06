<script setup lang="ts">
const SUGGESTIONS = [
  { label: '12-month FD', href: '/products/fixed-deposits' },
  { label: 'Home Loans', href: '/compare/housing-loans' },
  { label: 'Savings Account', href: '/compare/savings-accounts' },
  { label: 'Gold Loan', href: '/compare/gold-loans' }
]

const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)

function onSearch() {
  navigateTo(resolveSearchRoute(searchQuery.value))
}

// Called by AppHeader's search button (via pages/index.vue) so the header
// button and the hero's own search box act as one control.
defineExpose({
  focus: () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
    searchInput.value?.focus()
  }
})
</script>

<template>
  <section
    class="relative bg-navy bg-cover bg-center px-4 py-14 text-center sm:px-6 sm:py-20"
    style="background-image: linear-gradient(rgba(15,23,42,0.75), rgba(15,23,42,0.85)), url('/hero/skyline.jpg')"
  >
    <div class="mx-auto max-w-[820px]">
      <span class="inline-block rounded-pill bg-primary/20 px-3 py-1.5 text-[11px] font-bold uppercase tracking-wider text-primary sm:px-4 sm:text-xs">
        Sri Lanka's Financial Data Platform
      </span>

      <h1 class="mt-4 text-[1.7rem] font-extrabold leading-tight text-white sm:mt-5 sm:text-[2.3rem] lg:text-[3rem]">
        Compare Financial Products From Sri Lankan Banks
      </h1>

      <p class="mx-auto mt-3 max-w-[560px] text-sm text-white/70 sm:mt-4 sm:text-[0.95rem]">
        Direct verification of active market rates, limits, and penalties. Zero margins added.
      </p>

      <form class="mx-auto mt-6 flex max-w-[600px] items-center gap-1 rounded-xl bg-white p-1.5 shadow-2xl sm:mt-8 sm:gap-2 sm:p-2" @submit.prevent="onSearch">
        <Icon name="search" class="ml-2 h-4 w-4 shrink-0 text-muted sm:ml-3" />
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          placeholder="Search for FD rates, home loans, savings accounts..."
          autocomplete="off"
          class="min-w-0 flex-1 border-none bg-transparent px-1 py-2 text-xs text-navy outline-none placeholder:text-muted sm:text-sm"
        >
        <button type="submit" class="shrink-0 rounded-lg bg-primary px-4 py-2.5 text-xs font-semibold text-white transition hover:bg-primary/90 sm:px-6 sm:py-3 sm:text-sm">
          Search
        </button>
      </form>

      <div class="mt-4 flex flex-wrap items-center justify-center gap-2 sm:mt-5">
        <span class="mr-1 text-[11px] text-white/50 sm:text-xs">Suggestions:</span>
        <NuxtLink
          v-for="chip in SUGGESTIONS"
          :key="chip.label"
          :to="chip.href"
          class="rounded-pill bg-white/10 px-3 py-1.5 text-[11px] font-medium text-white/90 transition hover:bg-white/20 sm:px-3.5 sm:text-xs"
        >
          {{ chip.label }}
        </NuxtLink>
      </div>
    </div>
  </section>
</template>
