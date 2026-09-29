<script setup lang="ts">
useHead({ title: 'Site Editor — Bank Logos — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

import { DIRECTORY_BANKS } from '~/utils/bankDirectory'

const { roleAtLeast } = useAdminAuth()

interface OverrideRow { bank_slug: string; logo_url: string | null; logo_small_url: string | null }

const loading = ref(true)
const error = ref(false)
const overrides = ref<Record<string, OverrideRow>>({})
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    const rows = (await $fetch<{ data: OverrideRow[] }>('/api/v1/admin/bank-logos', { credentials: 'include' })).data ?? []
    overrides.value = Object.fromEntries(rows.map((r) => [r.bank_slug, r]))
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

function overrideFor(slug: string): OverrideRow {
  return overrides.value[slug] ?? { bank_slug: slug, logo_url: null, logo_small_url: null }
}

function defaultLogoSrc(bank: (typeof DIRECTORY_BANKS)[number]) {
  return bank.logoExt ? `/banks/${bank.slug}.${bank.logoExt}` : null
}
function defaultLogoSmallSrc(bank: (typeof DIRECTORY_BANKS)[number]) {
  if (bank.logoSmallExt) return `/banks/${bank.slug}-small.${bank.logoSmallExt}`
  return defaultLogoSrc(bank)
}

const uploadingKey = ref('')
async function upload(slug: string, field: 'logo_url' | 'logo_small_url', e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploadingKey.value = `${slug}-${field}`
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await $fetch<{ url: string }>('/api/v1/admin/media/upload', { method: 'POST', credentials: 'include', body: fd })
    const current = overrideFor(slug)
    await $fetch(`/api/v1/admin/bank-logos/${slug}`, {
      method: 'PUT', credentials: 'include',
      body: { logo_url: field === 'logo_url' ? res.url : (current.logo_url ?? ''), logo_small_url: field === 'logo_small_url' ? res.url : (current.logo_small_url ?? '') }
    })
    toast.value = 'Logo updated.'
    await load()
  } catch {
    toast.value = ''
  } finally {
    uploadingKey.value = ''
  }
}

async function clearOverride(slug: string) {
  try {
    await $fetch(`/api/v1/admin/bank-logos/${slug}`, { method: 'PUT', credentials: 'include', body: { logo_url: '', logo_small_url: '' } })
    toast.value = 'Reverted to default logo.'
    await load()
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
    <SiteEditorTabs active="bank-logos" />

    <p class="mb-4 text-sm text-muted">
      Replace a bank's logo without a code change — uploads apply immediately on the live site. Banks with no override
      keep using their default logo file.
    </p>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <div v-else class="overflow-hidden rounded-card border border-card-border bg-card">
      <div v-for="bank in DIRECTORY_BANKS" :key="bank.slug" class="flex flex-wrap items-center gap-4 border-b border-card-border px-4 py-3 last:border-none">
        <p class="w-48 shrink-0 font-semibold text-navy">{{ bank.displayName }}</p>

        <div class="flex items-center gap-2">
          <img
            :src="overrideFor(bank.slug).logo_url ?? defaultLogoSrc(bank) ?? ''"
            alt="" class="h-10 w-10 rounded border border-card-border object-contain p-1"
          >
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-semibold text-muted">Full Logo</label>
            <input type="file" accept="image/*" class="text-xs" :disabled="!roleAtLeast('admin')" @change="upload(bank.slug, 'logo_url', $event)">
          </div>
        </div>

        <div class="flex items-center gap-2">
          <img
            :src="overrideFor(bank.slug).logo_small_url ?? defaultLogoSmallSrc(bank) ?? ''"
            alt="" class="h-10 w-10 rounded border border-card-border object-contain p-1"
          >
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-semibold text-muted">Small Logo</label>
            <input type="file" accept="image/*" class="text-xs" :disabled="!roleAtLeast('admin')" @change="upload(bank.slug, 'logo_small_url', $event)">
          </div>
        </div>

        <p v-if="uploadingKey.startsWith(bank.slug)" class="text-xs text-muted">Uploading…</p>
        <button
          v-if="roleAtLeast('admin') && (overrideFor(bank.slug).logo_url || overrideFor(bank.slug).logo_small_url)"
          type="button" class="ml-auto text-xs font-semibold text-muted hover:text-red-600"
          @click="clearOverride(bank.slug)"
        >
          Revert to default
        </button>
      </div>
    </div>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
