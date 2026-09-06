import type { Config } from 'tailwindcss'

// Design tokens for the new Tailwind-based homepage (components/home/* and
// components/layout/AppHeader.vue + AppFooter.vue), matching the approved
// Figma exactly. The rest of the site still uses the CSS-custom-property
// theme in app.vue (light/dark toggle) — these are separate, fixed-palette
// tokens for the new component set only.
export default <Partial<Config>>{
  theme: {
    extend: {
      // Tailwind's defaults already match sm/md/lg/xl/2xl below (640, 768,
      // 1024, 1280, 1536) — declared explicitly rather than left implicit
      // so the scale is visible in one place; 3xl is the one genuinely
      // custom addition, for very wide desktop layouts.
      screens: {
        sm: '640px',
        md: '768px',
        lg: '1024px',
        xl: '1280px',
        '2xl': '1536px',
        '3xl': '1760px'
      },
      colors: {
        primary: '#2563EB',
        navy: '#0F172A',
        page: '#F8FAFC',
        card: '#FFFFFF',
        'card-border': '#E2E8F0',
        muted: '#64748B',
        'badge-bg': '#EFF6FF'
      },
      borderRadius: {
        card: '16px',
        pill: '999px'
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'Roboto', 'Helvetica', 'Arial', 'sans-serif']
      }
    }
  }
}
