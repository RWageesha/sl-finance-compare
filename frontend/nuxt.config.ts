// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  app: {
    head: {
      title: 'OpenFinance LK',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' }
      ],
      script: [
        {
          // Runs before Vue hydrates, so a returning visitor who chose
          // dark mode doesn't see a flash of the light-default page first.
          // Default is light (nothing to do) unless "dark" was explicitly
          // stored by useTheme.ts's toggle.
          innerHTML: `(function(){try{if(localStorage.getItem('theme')==='dark'){document.documentElement.setAttribute('data-theme','dark');}}catch(e){}})();`,
          type: 'text/javascript'
        }
      ]
    }
  },

  // Pages fetch relative "/api/v1/..." paths. In production the static
  // build is served by the same Go binary as the API (same origin, no
  // proxy needed). In dev, Nuxt runs on :3000 and the Go API on :8080 —
  // this proxies /api/* to it so the relative-path fetches work
  // identically in both environments without any CORS setup on the Go side.
  nitro: {
    devProxy: {
      '/api': {
        target: 'http://localhost:8080/api',
        changeOrigin: true
      }
    },
    prerender: {
      routes: [
        '/', '/rates', '/banks', '/products',
        '/compare/fixed-deposits', '/compare/fixed-deposits/results', '/compare/fixed-deposits/table',
        '/compare/housing-loans', '/compare/personal-loans', '/compare/gold-loans', '/compare/savings',
        '/calculators/fixed-deposit', '/calculators/loan-emi'
      ],
      // The homepage links to /api/v1/fixed-deposits (the live Go API,
      // useful once deployed) — without this, the static-site crawler
      // treats it as an internal page to prerender and fails, since that
      // route only exists on the Go server, not in this static build.
      ignore: ['/api']
    },
    // `nuxt generate` writes the static build straight into ../web, which
    // cmd/api/main.go serves unchanged via http.FileServer(http.Dir("web")) —
    // see README's "Frontend (Nuxt.js)" section. web/ is a build artifact
    // from here on; never hand-edit it.
    output: {
      publicDir: '../web'
    }
  }
})
