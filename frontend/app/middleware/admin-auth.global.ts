// Every /admin/* route except /admin/login requires a valid session.
// This app fetches everything client-side (no SSR data fetching
// anywhere in the project), so this check runs client-only too — an
// unauthenticated visit still gets Nuxt's static shell, then bounces to
// login the moment this resolves, same pattern as every other guarded
// page in this codebase.
export default defineNuxtRouteMiddleware(async (to) => {
  if (!to.path.startsWith('/admin')) return
  if (to.path === '/admin/login') return
  if (import.meta.server) return

  const { refresh } = useAdminAuth()
  const ok = await refresh()
  if (!ok) return navigateTo('/admin/login')
})
