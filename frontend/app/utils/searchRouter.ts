// Shared keyword router for every search box on the site (hero, header
// mobile menu, ...) — matches free-text search against the product
// categories and returns the right filtered view, falling back to the
// unfiltered rates page.
const ROUTES: { test: RegExp; href: string }[] = [
  { test: /housing|home/, href: '/compare/housing-loans' },
  { test: /personal loan/, href: '/compare/personal-loans' },
  { test: /gold|pawn/, href: '/compare/gold-loans' },
  { test: /loan/, href: '/rates?tab=loans' },
  { test: /saving/, href: '/compare/savings-accounts' },
  { test: /fixed deposit|\bfd\b|deposit/, href: '/products/fixed-deposits' }
]

export function resolveSearchRoute(query: string): string {
  const q = query.toLowerCase().trim()
  if (!q) return '/rates'
  const match = ROUTES.find((r) => r.test.test(q))
  // A recognized keyword jumps straight to its category page; anything
  // else (a bank name, a rate, free text) goes to the real Search
  // Results page instead of silently falling back to the unfiltered
  // rates table with no indication nothing specific was found.
  return match ? match.href : `/search?q=${encodeURIComponent(query.trim())}`
}
