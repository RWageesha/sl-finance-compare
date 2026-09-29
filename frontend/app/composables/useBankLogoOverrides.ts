// Module-level singleton (like useProductLookup.ts) so every BankLogo on
// a page shares one fetch instead of each instance hitting the API.
import type { BankLogoOverride } from '~/composables/useSiteContent'

const overrides = ref<Record<string, BankLogoOverride>>({})
const loaded = ref(false)

export function useBankLogoOverrides() {
  async function ensureLoaded() {
    if (loaded.value) return
    loaded.value = true
    const { fetchBankLogoOverrides } = useSiteContent()
    const rows = await fetchBankLogoOverrides().catch(() => [])
    overrides.value = Object.fromEntries(rows.map((r) => [r.bank_slug, r]))
  }
  return { overrides, ensureLoaded }
}
