<script setup lang="ts">
// Site-wide footer — used on every page. Self-contained: it owns its own
// calculator-modal triggers rather than taking them as props/emits, since
// "open the FD calculator" doesn't need to coordinate with anything else
// on whatever page this is mounted on.
const showFdCalc = ref(false)
const showLoanCalc = ref(false)
</script>

<template>
  <footer>
    <div class="wrap">
      <div class="footer-grid">
        <div class="footer-brand">
          <h4 class="brand-title">FindRate LK</h4>
          <p>A public, transparent directory of financial institutions and current interest rates in Sri Lanka. Powered by open banking standards.</p>
        </div>
        <div>
          <h4>Directory</h4>
          <ul>
            <li><NuxtLink to="/rates">All Banks</NuxtLink></li>
            <!-- We don't categorize tracked banks by license/type, so
                 these three stay visual-only until that data exists. -->
            <li><span class="footer-soon" title="Coming soon">Licensed Commercial</span></li>
            <li><span class="footer-soon" title="Coming soon">Development Banks</span></li>
            <li><span class="footer-soon" title="Coming soon">Savings Banks</span></li>
          </ul>
        </div>
        <div>
          <h4>Tools</h4>
          <ul>
            <li><NuxtLink to="/rates">Compare Rates</NuxtLink></li>
            <li><button type="button" class="footer-link-btn" @click="showFdCalc = true">Fixed Deposit Calc</button></li>
            <li><button type="button" class="footer-link-btn" @click="showLoanCalc = true">Housing Loan Calc</button></li>
            <li><a href="https://github.com/RWageesha/sl-finance-compare#readme" target="_blank" rel="noopener">API Documentation</a></li>
          </ul>
        </div>
        <div>
          <h4>System</h4>
          <ul>
            <li><span class="footer-soon" title="Coming soon">Data Status</span></li>
            <li><a href="https://github.com/RWageesha/sl-finance-compare/tree/main/scraper" target="_blank" rel="noopener">Sources</a></li>
            <li><a href="https://github.com/RWageesha/sl-finance-compare#readme" target="_blank" rel="noopener">Methodology</a></li>
            <li><span class="footer-soon" title="Coming soon">Contact Support</span></li>
          </ul>
        </div>
      </div>
      <div class="footer-bottom">
        <span>&copy; 2026 FindRate LK. Rates are informational, not financial advice.</span>
        <div class="footer-bottom-right">
          <span class="footer-soon" title="Coming soon">Privacy Policy</span>
          <span class="footer-soon" title="Coming soon">Disclaimer</span>
          <div class="socials">
            <!-- Only GitHub is a real account for this project — the other
                 two have no destination yet, so they're shown for visual
                 consistency but aren't real links (title says so). -->
            <span class="social-btn disabled" title="No X/Twitter account yet">X</span>
            <span class="social-btn disabled" title="No LinkedIn account yet">in</span>
            <a class="social-btn" href="https://github.com/RWageesha/sl-finance-compare" target="_blank" rel="noopener" title="GitHub">Gh</a>
          </div>
        </div>
      </div>
    </div>

    <FdCalculatorModal v-if="showFdCalc" @close="showFdCalc = false" />
    <LoanCalculatorModal v-if="showLoanCalc" @close="showLoanCalc = false" />
  </footer>
</template>

<style scoped>
.wrap {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 1.5rem;
}
footer {
  background: var(--footer-bg);
  color: var(--footer-text);
  padding: 2.5rem 0 1.5rem;
  margin-top: 1rem;
}
.footer-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
}
@media (max-width: 720px) {
  .footer-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
.footer-grid h4 {
  margin: 0 0 0.7rem;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--footer-muted);
}
.footer-grid ul {
  list-style: none;
  margin: 0;
  padding: 0;
}
.footer-grid li {
  margin-bottom: 0.5rem;
}
.footer-grid a {
  font-size: 0.85rem;
  color: var(--footer-text);
  text-decoration: none;
}
.footer-grid a:hover {
  text-decoration: underline;
}
.footer-brand .brand-title {
  color: var(--footer-text);
  font-size: 0.95rem;
  text-transform: none;
  letter-spacing: normal;
}
.footer-brand p {
  margin: 0.6rem 0 0;
  font-size: 0.83rem;
  color: var(--footer-muted);
  line-height: 1.5;
}
.footer-link-btn {
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  font-size: 0.85rem;
  color: var(--footer-text);
  cursor: pointer;
  text-align: left;
}
.footer-link-btn:hover {
  text-decoration: underline;
}
.footer-soon {
  font-size: 0.85rem;
  color: var(--footer-muted);
  opacity: 0.6;
  cursor: default;
}
.footer-bottom {
  margin-top: 2rem;
  padding-top: 1.2rem;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 0.78rem;
  color: var(--footer-muted);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.footer-bottom-right {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}
.footer-bottom-right .footer-soon {
  font-size: 0.78rem;
}
.socials {
  display: flex;
  gap: 0.5rem;
}
.social-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.08);
  color: var(--footer-text);
  font-size: 0.68rem;
  font-weight: 700;
  text-decoration: none;
}
.social-btn:not(.disabled):hover {
  background: var(--accent);
}
.social-btn.disabled {
  opacity: 0.4;
  cursor: default;
}
</style>
