<script setup lang="ts">
// Site-wide top nav — used on every page. Compare/Calculators are homepage
// sections, so they're written as "/#id" rather than "#id": from the
// homepage that's a same-page jump, and from any other page it navigates
// home first and then jumps (Nuxt's default scroll behavior handles the
// hash), so the same links work correctly everywhere this component is
// mounted. Banks and Products are real standalone pages.
//
// The search button doesn't know what "search" means on every page (only
// the homepage has the hero search box), so it just emits — the homepage
// focuses its own input; a future page without one can navigate home.
defineEmits<{ 'focus-search': [] }>()

const CALCULATORS = [
  { label: 'Fixed Deposit Calculator', href: '/calculators/fixed-deposit' },
  { label: 'Loan EMI Calculator', href: '/calculators/loan-emi' }
]

const open = ref(false)
const dropdownEl = ref<HTMLElement | null>(null)

function onDocClick(e: MouseEvent) {
  if (dropdownEl.value && !dropdownEl.value.contains(e.target as Node)) open.value = false
}

onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))

// Closing on route change (rather than only on outside-click) means
// picking a calculator from the menu doesn't leave it visibly stuck open
// during the page transition.
const route = useRoute()
watch(() => route.fullPath, () => {
  open.value = false
})
</script>

<template>
  <nav class="site-nav">
    <div class="wrap">
      <NuxtLink class="brand" to="/">
        <img src="/brand/logo.png" alt="FindRate" class="logo-img logo-light">
        <img src="/brand/logo-white.png" alt="FindRate" class="logo-img logo-dark">
      </NuxtLink>
      <div class="nav-links">
        <NuxtLink to="/banks">Banks</NuxtLink>
        <NuxtLink to="/products">Products</NuxtLink>
        <NuxtLink to="/#comparisons">Compare</NuxtLink>
        <div ref="dropdownEl" class="nav-dropdown">
          <button type="button" class="dropdown-trigger" :aria-expanded="open" @click="open = !open">
            Calculators
            <Icon name="chevron-down" />
          </button>
          <div v-if="open" class="dropdown-menu">
            <NuxtLink v-for="c in CALCULATORS" :key="c.href" :to="c.href" class="dropdown-item">{{ c.label }}</NuxtLink>
          </div>
        </div>
        <NuxtLink to="/rates">Data</NuxtLink>
      </div>
      <ThemeToggle />
      <button type="button" class="nav-search-btn" aria-label="Search" @click="$emit('focus-search')">
        <Icon name="search" />
      </button>
    </div>
  </nav>
</template>

<style scoped>
.wrap {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 1.5rem;
}
.site-nav {
  background: var(--panel);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 10;
}
.site-nav .wrap {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  padding-top: 0.9rem;
  padding-bottom: 0.9rem;
}
.brand {
  display: flex;
  align-items: center;
}
.logo-img {
  height: 26px;
  width: auto;
  display: block;
}
.logo-dark {
  display: none;
}
:root[data-theme='dark'] .logo-light {
  display: none;
}
:root[data-theme='dark'] .logo-dark {
  display: block;
}
.nav-links {
  display: flex;
  gap: 1.4rem;
  margin-left: 0.5rem;
  flex: 1;
}
.nav-links a {
  font-size: 0.88rem;
  color: var(--muted);
  text-decoration: none;
}
.nav-links a:hover {
  color: var(--text);
}

.nav-dropdown {
  position: relative;
}
.dropdown-trigger {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  font-size: 0.88rem;
  color: var(--muted);
  cursor: pointer;
}
.dropdown-trigger:hover,
.dropdown-trigger[aria-expanded='true'] {
  color: var(--text);
}
.dropdown-trigger svg {
  width: 11px;
  height: 11px;
}
.dropdown-menu {
  position: absolute;
  top: calc(100% + 0.7rem);
  left: 0;
  min-width: 210px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.15);
  padding: 0.4rem;
  z-index: 20;
}
.dropdown-item {
  display: block;
  padding: 0.55rem 0.7rem;
  border-radius: 6px;
  font-size: 0.85rem;
  color: var(--text);
  text-decoration: none;
}
.dropdown-item:hover {
  background: var(--bg);
  color: var(--accent);
}

.nav-search-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 7px;
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--muted);
  cursor: pointer;
  flex: none;
}
.nav-search-btn:hover {
  color: var(--text);
  border-color: var(--accent);
}
.nav-search-btn svg {
  width: 15px;
  height: 15px;
}
</style>
