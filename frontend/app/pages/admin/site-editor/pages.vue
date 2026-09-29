<script setup lang="ts">
useHead({ title: 'Site Editor — Page Text — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface ContentBlock { id: number; page: string; content_key: string; label: string; body: string }

const PAGES = [
  { key: 'about-us', label: 'About Us' },
  { key: 'contact-us', label: 'Contact Us' }
]

const loading = ref(true)
const error = ref(false)
const blocks = ref<ContentBlock[]>([])
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    const results = await Promise.all(
      PAGES.map((p) => $fetch<{ data: ContentBlock[] }>(`/api/v1/admin/site-content?page=${p.key}`, { credentials: 'include' }))
    )
    blocks.value = results.flatMap((r) => r.data ?? [])
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const drafts = reactive<Record<number, string>>({})
watch(blocks, (b) => { for (const block of b) drafts[block.id] = block.body }, { immediate: true })

async function save(block: ContentBlock) {
  try {
    await $fetch(`/api/v1/admin/site-content/${block.id}`, { method: 'PATCH', credentials: 'include', body: { body: drafts[block.id] } })
    toast.value = `${block.label} updated.`
    block.body = drafts[block.id]
  } catch {
    toast.value = ''
  }
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Site Editor</h1>
    </div>
    <SiteEditorTabs active="pages" />

    <p class="mb-4 text-sm text-muted">
      Plain-text content blocks for pages that don't have their own admin section — appears on the live site
      immediately, no redeploy needed.
    </p>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <div v-else class="flex flex-col gap-4">
      <div v-for="p in PAGES" :key="p.key" class="rounded-card border border-card-border bg-card p-4">
        <h3 class="mb-3 text-sm font-bold text-navy">{{ p.label }}</h3>
        <div v-for="block in blocks.filter((b) => b.page === p.key)" :key="block.id" class="flex flex-col gap-2">
          <label class="text-xs font-semibold text-muted">{{ block.label }}</label>
          <textarea
            v-model="drafts[block.id]"
            rows="4"
            class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"
            :disabled="!roleAtLeast('admin')"
          />
          <button
            v-if="roleAtLeast('admin')"
            type="button"
            class="self-end rounded-lg bg-primary px-4 py-1.5 text-xs font-bold text-white hover:bg-primary/90 disabled:opacity-50"
            :disabled="drafts[block.id] === block.body"
            @click="save(block)"
          >
            Save
          </button>
        </div>
      </div>
    </div>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
