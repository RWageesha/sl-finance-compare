<script setup lang="ts">
// A component, not a Nuxt layout — this project's app.vue renders
// <NuxtPage /> directly with no <NuxtLayout> wrapper (every public page
// includes its own <AppHeader>/<AppFooter> the same way), so
// definePageMeta({ layout }) would be silently inert here. Every admin
// page wraps its content in this instead, the same pattern as the rest
// of the site.
const { admin, roleAtLeast, logout } = useAdminAuth()
const route = useRoute()

const NAV = [
  { label: 'Dashboard', to: '/admin/dashboard', icon: 'grid', minRole: 'viewer' },
  { label: 'Verification Queue', to: '/admin/verification-queue', icon: 'check', minRole: 'viewer' },
  { label: 'Rate Management', to: '/admin/rate-management', icon: 'percent', minRole: 'viewer' },
  { label: 'Data Sources', to: '/admin/data-sources', icon: 'database', minRole: 'viewer' },
  { label: 'Scraping Jobs', to: '/admin/scraping-jobs', icon: 'clock', minRole: 'viewer' },
  { label: 'Banks', to: '/admin/banks', icon: 'bank', minRole: 'viewer' },
  { label: 'Products', to: '/admin/products', icon: 'tag', minRole: 'viewer' },
  { label: 'User Reports', to: '/admin/user-reports', icon: 'flag', minRole: 'viewer' },
  { label: 'Audit Logs', to: '/admin/audit-logs', icon: 'list', minRole: 'admin' }
]
const visibleNav = computed(() => NAV.filter((n) => roleAtLeast(n.minRole)))

const scraping = ref<{ status: string }[]>([])
onMounted(async () => {
  try {
    const res = await $fetch<{ scraping: { status: string }[] }>('/api/v1/admin/dashboard', { credentials: 'include' })
    scraping.value = res.scraping ?? []
  } catch {
    scraping.value = []
  }
})
const allOnline = computed(() => scraping.value.length > 0 && scraping.value.every((s) => s.status !== 'failed'))
const statusLabel = computed(() => {
  if (scraping.value.length === 0) return 'No scrapers recorded yet'
  return allOnline.value ? `All ${scraping.value.length} scrapers online` : `${scraping.value.filter((s) => s.status === 'failed').length} scraper(s) failing`
})
</script>

<template>
  <div class="flex min-h-screen bg-page">
    <aside class="fixed inset-y-0 left-0 flex w-60 flex-col border-r border-card-border bg-white">
      <div class="p-5">
        <NuxtLink to="/admin/dashboard" class="flex items-center gap-2">
          <img src="/brand/logo-black.png" alt="" class="h-6 w-auto">
          <span class="text-sm font-extrabold text-navy">FindRate LK</span>
        </NuxtLink>
        <p class="mt-0.5 text-[10px] font-bold uppercase tracking-wider text-muted">Admin Portal</p>
      </div>

      <nav class="flex-1 space-y-0.5 px-3">
        <NuxtLink
          v-for="item in visibleNav"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-semibold transition"
          :class="route.path.startsWith(item.to) ? 'bg-primary text-white' : 'text-navy hover:bg-page'"
        >
          <Icon :name="item.icon" class="h-4 w-4 shrink-0" />
          {{ item.label }}
        </NuxtLink>
      </nav>

      <div class="border-t border-card-border p-4">
        <p class="mb-2 text-[10px] font-bold uppercase tracking-wider text-muted">System Status</p>
        <p class="flex items-center gap-1.5 text-xs text-navy">
          <span class="h-1.5 w-1.5 rounded-full" :class="allOnline ? 'bg-emerald-500' : 'bg-red-500'" />
          {{ statusLabel }}
        </p>
      </div>

      <div class="flex items-center justify-between border-t border-card-border p-4">
        <div class="min-w-0">
          <p class="truncate text-sm font-bold text-navy">{{ admin?.username }}</p>
          <p class="truncate text-[11px] text-muted">{{ admin?.role }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <NuxtLink to="/admin/settings" aria-label="Settings" class="flex h-7 w-7 items-center justify-center rounded-lg text-muted hover:bg-page hover:text-navy">
            <Icon name="gear" class="h-4 w-4" />
          </NuxtLink>
          <button type="button" class="rounded-lg border border-card-border px-2.5 py-1.5 text-xs font-semibold text-navy hover:border-primary" @click="logout">
            Logout
          </button>
        </div>
      </div>
    </aside>

    <main class="ml-60 flex-1 p-6">
      <slot />
    </main>
  </div>
</template>
