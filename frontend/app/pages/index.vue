<script setup lang="ts">
// Product category cards. "GOLD_LOAN" is a real category we scrape
// (pawning products from ComBank/BOC); "Credit Cards" isn't scraped at
// all (fees/APR data, not an interest rate), so it's shown as
// disabled/coming-soon rather than linking somewhere with no data.
const PRODUCTS = [
  { icon: 'bank', color: 'fd', label: 'Fixed Deposits', desc: 'View & compare rates', href: '/compare/fixed-deposits' },
  { icon: 'home', color: 'housing', label: 'Housing Loans', desc: 'View & compare rates', href: '/compare/housing-loans' },
  { icon: 'savings', color: 'savings', label: 'Savings Accounts', desc: 'View & compare rates', href: '/compare/savings' },
  { icon: 'card', color: 'personal', label: 'Personal Loans', desc: 'View & compare rates', href: '/compare/personal-loans' },
  { icon: 'coin', color: 'gold', label: 'Gold Loans', desc: 'View & compare rates', href: '/compare/gold-loans' },
  { icon: 'card', color: 'muted', label: 'Credit Cards', desc: 'Coming soon', href: '' }
] as const

const COMPARISONS = [
  {
    label: 'Fixed Deposit Rates',
    desc: 'Compare fixed deposit rates across every tenure we track, from every bank.',
    href: '/compare/fixed-deposits',
    chart: 'bar-chart'
  },
  {
    label: 'Housing Loan Rates',
    desc: 'See fixed-rate housing loan pricing side by side across banks.',
    href: '/compare/housing-loans',
    chart: 'line-chart',
    dark: true
  },
  {
    label: 'Savings Account Rates',
    desc: 'Find the highest-yield savings accounts, including balance-tiered rates.',
    href: '/compare/savings',
    chart: 'bar-chart-alt'
  }
] as const

// Very small keyword router: match the search text against the same
// product categories above and jump straight to the right filtered view,
// falling back to the unfiltered rates page.
const ROUTES: { test: RegExp; href: string }[] = [
  { test: /housing|home/, href: '/compare/housing-loans' },
  { test: /personal loan/, href: '/compare/personal-loans' },
  { test: /gold|pawn/, href: '/compare/gold-loans' },
  { test: /loan/, href: '/rates?tab=loans' },
  { test: /saving/, href: '/compare/savings' },
  { test: /fixed deposit|\bfd\b|deposit/, href: '/compare/fixed-deposits' }
]

const showFdCalc = ref(false)
const showLoanCalc = ref(false)

// The homepage's own Quick Calculators section triggers these directly;
// SiteFooter's "Fixed Deposit Calc" / "Housing Loan Calc" links own a
// separate pair of the same modals, since neither needs to know about
// the other's open/closed state.

const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)

function focusSearch() {
  // The search box lives in the hero at the top of the page.
  window.scrollTo({ top: 0, behavior: 'smooth' })
  searchInput.value?.focus()
}

function onSearch() {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) {
    navigateTo('/rates')
    return
  }
  const match = ROUTES.find((r) => r.test.test(q))
  navigateTo(match ? match.href : '/rates')
}
</script>

<template>
  <div>
    <SiteNav @focus-search="focusSearch" />

    <header class="hero">
      <!-- CSS-only skyline silhouette (no external image asset) with a
           dark gradient overlay on top for text legibility. -->
      <svg class="skyline" viewBox="0 0 1200 220" preserveAspectRatio="none" aria-hidden="true">
        <rect x="0" y="120" width="70" height="100" />
        <rect x="80" y="70" width="55" height="150" />
        <rect x="145" y="140" width="60" height="80" />
        <rect x="215" y="40" width="45" height="180" />
        <rect x="270" y="100" width="65" height="120" />
        <rect x="345" y="60" width="50" height="160" />
        <rect x="405" y="130" width="70" height="90" />
        <rect x="485" y="20" width="40" height="200" />
        <rect x="535" y="85" width="60" height="135" />
        <rect x="605" y="115" width="55" height="105" />
        <rect x="670" y="50" width="45" height="170" />
        <rect x="725" y="95" width="65" height="125" />
        <rect x="800" y="35" width="50" height="185" />
        <rect x="860" y="125" width="60" height="95" />
        <rect x="930" y="65" width="55" height="155" />
        <rect x="995" y="105" width="70" height="115" />
        <rect x="1075" y="45" width="45" height="175" />
        <rect x="1130" y="90" width="70" height="130" />
      </svg>
      <div class="wrap">
        <h1>Compare Financial Products From Sri Lankan Banks</h1>
        <p>Interest rates scraped directly from official bank websites and refreshed automatically — no aggregator guesswork.</p>
        <form class="search-form" @submit.prevent="onSearch">
          <input
            ref="searchInput"
            v-model="searchQuery"
            type="text"
            placeholder="What are you looking for? (e.g. Fixed Deposit, Housing Loan…)"
            autocomplete="off"
          >
          <button type="submit">Search</button>
        </form>
      </div>
    </header>

    <section id="products">
      <div class="wrap">
        <p class="eyebrow">Product categories</p>
        <h2 class="section-title">Popular Financial Products</h2>
        <div class="grid-3">
          <ProductCard v-for="p in PRODUCTS" :key="p.label" v-bind="p" />
        </div>
      </div>
    </section>

    <section id="calculators">
      <div class="wrap calc-wrap">
        <div class="calc-copy">
          <p class="eyebrow">Calculators</p>
          <h2 class="section-title">Quick Calculators</h2>
          <p class="section-desc">Get an instant estimate without leaving the homepage. Pick a calculator below.</p>
          <div class="calc-list">
            <div class="calc-row">
              <div class="calc-icon"><Icon name="calculator" /></div>
              <div class="calc-row-copy">
                <h3>Fixed Deposit Calculator</h3>
                <p>Estimate your maturity value across Sri Lankan banks.</p>
              </div>
              <button type="button" class="calc-link" @click="showFdCalc = true">Calculate &rarr;</button>
            </div>
            <div class="calc-row">
              <div class="calc-icon"><Icon name="percent" /></div>
              <div class="calc-row-copy">
                <h3>Loan EMI Calculator</h3>
                <p>Estimate your monthly repayment on any loan.</p>
              </div>
              <button type="button" class="calc-link" @click="showLoanCalc = true">Calculate &rarr;</button>
            </div>
          </div>
        </div>
        <div class="calc-illustration" aria-hidden="true">
          <div class="calc-illustration-icon"><Icon name="calculator" /></div>
          <div class="calc-illustration-icon alt"><Icon name="bar-chart" /></div>
        </div>
      </div>
    </section>

    <section id="comparisons">
      <div class="wrap">
        <p class="eyebrow">Direct comparison</p>
        <h2 class="section-title">Popular Head-to-Head Comparisons</h2>
        <div class="grid-3">
          <CompareCard v-for="c in COMPARISONS" :key="c.label" v-bind="c" />
        </div>
      </div>
    </section>

    <section id="recent-rates">
      <div class="wrap">
        <p class="eyebrow">Real-time updates</p>
        <h2 class="section-title">Recently Updated Bank Rates</h2>
        <RecentRatesTable />
      </div>
    </section>

    <section class="trust-section">
      <div class="wrap">
        <div class="trust-banner">
          <div class="icon"><Icon name="shield" /></div>
          <div class="copy">
            <strong>Data Trust &amp; Sourcing</strong>
            <span>Every rate links back to the bank's own page so you can verify it yourself. Scraped and normalized by an open-source pipeline.</span>
          </div>
          <NuxtLink class="cta" to="/rates">Transparency Hub</NuxtLink>
        </div>
      </div>
    </section>

    <SiteFooter />

    <FdCalculatorModal v-if="showFdCalc" @close="showFdCalc = false" />
    <LoanCalculatorModal v-if="showLoanCalc" @close="showLoanCalc = false" />
  </div>
</template>

<style scoped>
.wrap {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 1.5rem;
}

/* Hero */
.hero {
  position: relative;
  background: linear-gradient(180deg, var(--navy), var(--navy-dark));
  color: #eef1fb;
  padding: 3.2rem 0 3.6rem;
  text-align: center;
  overflow: hidden;
}
.hero .skyline {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  color: rgba(255, 255, 255, 0.06);
  fill: currentColor;
  pointer-events: none;
}
.hero .wrap {
  position: relative;
}
.hero h1 {
  margin: 0 0 0.75rem;
  font-size: clamp(1.6rem, 3.6vw, 2.35rem);
  line-height: 1.25;
  font-weight: 700;
}
.hero p {
  margin: 0 auto 1.75rem;
  max-width: 620px;
  color: #b9c0e2;
  font-size: 0.98rem;
}
.search-form {
  display: flex;
  gap: 0.5rem;
  max-width: 560px;
  margin: 0 auto;
  background: #fff;
  padding: 0.35rem;
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
}
.search-form input {
  flex: 1;
  border: none;
  outline: none;
  padding: 0.6rem 0.7rem;
  font: inherit;
  font-size: 0.9rem;
  border-radius: 7px;
  color: #1c1c1f;
}
.search-form button {
  font: inherit;
  font-weight: 600;
  font-size: 0.88rem;
  padding: 0.6rem 1.2rem;
  border-radius: 7px;
  border: none;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
}

/* Sections */
section {
  padding: 2.75rem 0;
}
.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--accent);
  margin: 0 0 0.35rem;
}
.section-title {
  margin: 0 0 1.5rem;
  font-size: 1.3rem;
  font-weight: 700;
}

.grid-3 {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}
@media (max-width: 720px) {
  .grid-3 {
    grid-template-columns: 1fr;
  }
}

/* Quick Calculators */
.section-desc {
  margin: -0.75rem 0 1.4rem;
  color: var(--muted);
  font-size: 0.9rem;
  max-width: 520px;
}
.calc-wrap {
  display: flex;
  gap: 2rem;
  align-items: stretch;
}
.calc-copy {
  flex: 1 1 60%;
  min-width: 0;
}
.calc-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.calc-row {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem 1.2rem;
}
.calc-icon {
  width: 38px;
  height: 38px;
  flex: none;
  border-radius: 8px;
  background: #e6f0ff;
  color: #1a4fb4;
  display: flex;
  align-items: center;
  justify-content: center;
}
[data-theme='dark'] .calc-icon {
  background: #17335f;
  color: #a8c6ff;
}
.calc-icon svg {
  width: 18px;
  height: 18px;
}
.calc-row-copy {
  flex: 1;
  min-width: 0;
}
.calc-row-copy h3 {
  margin: 0 0 0.15rem;
  font-size: 0.92rem;
  font-weight: 700;
}
.calc-row-copy p {
  margin: 0;
  font-size: 0.8rem;
  color: var(--muted);
}
.calc-link {
  background: none;
  border: none;
  font: inherit;
  font-weight: 600;
  font-size: 0.85rem;
  color: var(--accent);
  cursor: pointer;
  white-space: nowrap;
}
.calc-link:hover {
  text-decoration: underline;
}
.calc-illustration {
  flex: 1 1 40%;
  border-radius: 10px;
  min-height: 220px;
  background: linear-gradient(135deg, var(--bg), var(--border));
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  position: relative;
}
[data-theme='dark'] .calc-illustration {
  background: linear-gradient(135deg, var(--navy-dark), var(--navy));
}
.calc-illustration-icon {
  width: 76px;
  height: 76px;
  border-radius: 14px;
  background: var(--panel);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
}
.calc-illustration-icon svg {
  width: 34px;
  height: 34px;
}
.calc-illustration-icon.alt {
  width: 56px;
  height: 56px;
  position: absolute;
  bottom: 22px;
  right: 28px;
  color: #00707a;
}
.calc-illustration-icon.alt svg {
  width: 26px;
  height: 26px;
}
@media (max-width: 720px) {
  .calc-wrap {
    flex-direction: column;
  }
}

/* Trust banner — sits on a navy section, echoing the hero's color, so the
   white card reads as a highlighted callout against it. */
.trust-section {
  background: linear-gradient(135deg, var(--navy), var(--navy-dark));
}
.trust-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  background: var(--panel);
  border-radius: 10px;
  padding: 1.1rem 1.3rem;
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.2);
}
.trust-banner .icon {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--accent);
  color: #fff;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
}
.trust-banner .icon svg {
  width: 18px;
  height: 18px;
}
.trust-banner .copy {
  flex: 1;
  min-width: 220px;
}
.trust-banner .copy strong {
  display: block;
  font-size: 0.92rem;
  margin-bottom: 0.15rem;
}
.trust-banner .copy span {
  font-size: 0.83rem;
  color: var(--muted);
}
.trust-banner .cta {
  font-size: 0.85rem;
  font-weight: 600;
  padding: 0.55rem 1.1rem;
  border-radius: 7px;
  background: var(--accent);
  color: #fff;
  text-decoration: none;
  white-space: nowrap;
}
</style>
