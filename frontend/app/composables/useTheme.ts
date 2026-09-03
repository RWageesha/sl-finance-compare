// Explicit light/dark theme control (not just following the OS's
// prefers-color-scheme) — default is always light, the user can toggle to
// dark, and the choice persists via localStorage. Theme state lives on
// <html data-theme="..."> (see app.vue's CSS, which keys off that
// attribute) rather than a CSS class, so the anti-flash inline script in
// nuxt.config.ts's app.head.script can set it before Vue even hydrates.

export type Theme = 'light' | 'dark'

export function useTheme() {
  // useState (not a plain ref) so every component sharing this composable
  // reads/writes the same value instead of each getting its own copy.
  const theme = useState<Theme>('theme', () => 'light')

  function apply(next: Theme) {
    theme.value = next
    if (import.meta.client) {
      document.documentElement.setAttribute('data-theme', next)
      try {
        localStorage.setItem('theme', next)
      } catch {
        // localStorage unavailable (private browsing, etc.) — the toggle
        // still works for the current page load, it just won't persist.
      }
    }
  }

  function toggle() {
    apply(theme.value === 'dark' ? 'light' : 'dark')
  }

  // Aligns Vue's reactive state with whatever the anti-flash script in
  // nuxt.config.ts already set on <html> before this component mounted —
  // call this once in each page's onMounted.
  function syncFromDom() {
    if (import.meta.client) {
      theme.value = document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light'
    }
  }

  return { theme, toggle, syncFromDom }
}
