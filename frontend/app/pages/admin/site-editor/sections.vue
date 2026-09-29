<script setup lang="ts">
useHead({ title: 'Site Editor — Homepage Sections — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface SectionRow {
  id: number; page: string; section_key: string; label: string
  is_visible: boolean; layout_variant: string; background_image_url: string | null; sort_order: number
}

const loading = ref(true)
const error = ref(false)
const sections = ref<SectionRow[]>([])
const toast = ref('')
const saveError = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    sections.value = (await $fetch<{ data: SectionRow[] }>('/api/v1/admin/site-sections?page=home', { credentials: 'include' })).data ?? []
    sections.value.sort((a, b) => a.sort_order - b.sort_order)
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

// Hero is the only section with a real alternate layout today (a plain
// static banner vs. a rotating slider) — every other section only ever
// toggles visible/hidden, so the variant dropdown only shows for it.
const VARIANTS: Record<string, { value: string; label: string }[]> = {
  hero: [
    { value: 'default', label: 'Static' },
    { value: 'slider', label: 'Slider' }
  ]
}

async function saveSection(s: SectionRow) {
  saveError.value = ''
  try {
    await $fetch(`/api/v1/admin/site-sections/${s.id}`, {
      method: 'PATCH', credentials: 'include',
      body: { is_visible: s.is_visible, layout_variant: s.layout_variant, background_image_url: s.background_image_url ?? '', sort_order: s.sort_order }
    })
    toast.value = `${s.label} updated.`
  } catch (err) {
    toast.value = ''
    saveError.value = `Failed to update "${s.label}": ${fetchErrorMessage(err)}`
    await load()
  }
}

function toggleVisible(s: SectionRow) {
  s.is_visible = !s.is_visible
  saveSection(s)
}
function changeVariant(s: SectionRow, variant: string) {
  s.layout_variant = variant
  saveSection(s)
}

const uploadingHeroImage = ref(false)
async function uploadHeroImage(s: SectionRow, e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploadingHeroImage.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await $fetch<{ url: string }>('/api/v1/admin/media/upload', { method: 'POST', credentials: 'include', body: fd })
    s.background_image_url = res.url
    await saveSection(s)
  } catch (err) {
    toast.value = ''
    saveError.value = `Failed to upload hero image: ${fetchErrorMessage(err)}`
  } finally {
    uploadingHeroImage.value = false
  }
}
async function resetHeroImage(s: SectionRow) {
  s.background_image_url = null
  await saveSection(s)
}

function move(index: number, dir: -1 | 1) {
  const target = index + dir
  if (target < 0 || target >= sections.value.length) return
  const a = sections.value[index]
  const b = sections.value[target]
  const tmp = a.sort_order
  a.sort_order = b.sort_order
  b.sort_order = tmp
  sections.value.sort((x, y) => x.sort_order - y.sort_order)
  saveSection(a)
  saveSection(b)
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Site Editor</h1>
    </div>
    <SiteEditorTabs active="sections" />

    <p class="mb-4 text-sm text-muted">
      Show or hide homepage sections, reorder them, and switch layout variants where one exists — changes appear on the
      live site immediately, no redeploy needed.
    </p>

    <p v-if="saveError" class="mb-4 rounded-lg bg-red-50 px-3.5 py-2.5 text-xs font-semibold text-red-700">{{ saveError }}</p>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <div v-else class="overflow-hidden rounded-card border border-card-border bg-card">
      <div
        v-for="(s, i) in sections"
        :key="s.id"
        class="flex flex-wrap items-center gap-3 border-b border-card-border px-4 py-3 last:border-none"
      >
        <div class="flex flex-col gap-0.5">
          <button type="button" class="text-muted hover:text-primary disabled:opacity-30" :disabled="i === 0" @click="move(i, -1)">&uarr;</button>
          <button type="button" class="text-muted hover:text-primary disabled:opacity-30" :disabled="i === sections.length - 1" @click="move(i, 1)">&darr;</button>
        </div>
        <p class="min-w-[220px] flex-1 font-semibold text-navy">{{ s.label }}</p>

        <select
          v-if="VARIANTS[s.section_key]"
          :value="s.layout_variant"
          class="rounded-lg border border-card-border px-2.5 py-1.5 text-xs"
          :disabled="!roleAtLeast('admin')"
          @change="changeVariant(s, ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="v in VARIANTS[s.section_key]" :key="v.value" :value="v.value">{{ v.label }}</option>
        </select>

        <button
          type="button"
          class="relative h-5 w-9 rounded-full transition disabled:cursor-not-allowed disabled:opacity-50"
          :class="s.is_visible ? 'bg-primary' : 'bg-card-border'"
          :disabled="!roleAtLeast('admin')"
          @click="toggleVisible(s)"
        >
          <span class="absolute top-0.5 h-4 w-4 rounded-full bg-white transition" :class="s.is_visible ? 'left-4' : 'left-0.5'" />
        </button>
        <span class="w-16 text-xs font-semibold" :class="s.is_visible ? 'text-emerald-700' : 'text-muted'">
          {{ s.is_visible ? 'Visible' : 'Hidden' }}
        </span>

        <div v-if="s.section_key === 'hero'" class="flex w-full items-center gap-2 pl-8 pt-1">
          <img :src="s.background_image_url ?? '/hero/skyline.jpg'" alt="" class="h-10 w-16 rounded border border-card-border object-cover">
          <input type="file" accept="image/*" class="text-xs" :disabled="!roleAtLeast('admin')" @change="uploadHeroImage(s, $event)">
          <span v-if="uploadingHeroImage" class="text-xs text-muted">Uploading…</span>
          <button v-if="s.background_image_url" type="button" class="text-xs font-semibold text-muted hover:text-red-600" @click="resetHeroImage(s)">Reset to default</button>
        </div>
      </div>
    </div>

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
