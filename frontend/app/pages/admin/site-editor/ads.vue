<script setup lang="ts">
useHead({ title: 'Site Editor — Ads — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface AdSlotOption { id: number; key: string; label: string; kind: string }
interface AdRow {
  id: number; slot_id: number; slot_key: string; ad_type: string; title: string
  image_url: string; target_url: string; sort_order: number; is_active: boolean
  starts_at: string; ends_at: string | null
}

const loading = ref(true)
const error = ref(false)
const slots = ref<AdSlotOption[]>([])
const ads = ref<AdRow[]>([])
const slotFilter = ref('')
const toast = ref('')

async function load() {
  loading.value = true
  error.value = false
  try {
    const [slotsRes, adsRes] = await Promise.all([
      $fetch<{ data: AdSlotOption[] }>('/api/v1/admin/ad-slots', { credentials: 'include' }),
      $fetch<{ data: AdRow[] }>('/api/v1/admin/ads', { credentials: 'include' })
    ])
    slots.value = slotsRes.data ?? []
    ads.value = adsRes.data ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const filtered = computed(() => (slotFilter.value ? ads.value.filter((a) => a.slot_key === slotFilter.value) : ads.value))
function slotLabel(key: string) { return slots.value.find((s) => s.key === key)?.label ?? key }

// datetime-local inputs need "YYYY-MM-DDTHH:mm" with no timezone suffix —
// toISOString() gives UTC with a "Z", so this trims it to the local form
// the <input type="datetime-local"> element expects.
function toLocalInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

interface AdForm {
  slot_id: number; ad_type: string; title: string; image_url: string; target_url: string
  sort_order: number; is_active: boolean; starts_at: string; ends_at: string
}
function blankForm(): AdForm {
  return { slot_id: slots.value[0]?.id ?? 0, ad_type: 'house', title: '', image_url: '', target_url: '', sort_order: 0, is_active: true, starts_at: toLocalInput(new Date().toISOString()), ends_at: '' }
}

const editing = ref<AdRow | null>(null)
const adding = ref(false)
const form = reactive<AdForm>(blankForm())
const uploading = ref(false)
const uploadError = ref('')

function openAdd() {
  Object.assign(form, blankForm())
  adding.value = true
}
function openEdit(row: AdRow) {
  editing.value = row
  Object.assign(form, {
    slot_id: row.slot_id, ad_type: row.ad_type, title: row.title, image_url: row.image_url,
    target_url: row.target_url, sort_order: row.sort_order, is_active: row.is_active,
    starts_at: toLocalInput(row.starts_at), ends_at: toLocalInput(row.ends_at)
  })
}
function closeModal() {
  editing.value = null
  adding.value = false
  uploadError.value = ''
}

function bodyFromForm() {
  return {
    slot_id: form.slot_id,
    ad_type: form.ad_type,
    title: form.title,
    image_url: form.image_url,
    target_url: form.target_url,
    sort_order: form.sort_order,
    is_active: form.is_active,
    starts_at: form.starts_at ? new Date(form.starts_at).toISOString() : undefined,
    ends_at: form.ends_at ? new Date(form.ends_at).toISOString() : null
  }
}

async function save() {
  try {
    if (editing.value) {
      await $fetch(`/api/v1/admin/ads/${editing.value.id}`, { method: 'PATCH', credentials: 'include', body: bodyFromForm() })
      toast.value = 'Ad updated.'
    } else {
      await $fetch('/api/v1/admin/ads', { method: 'POST', credentials: 'include', body: bodyFromForm() })
      toast.value = 'Ad created.'
    }
    closeModal()
    await load()
  } catch {
    toast.value = ''
  }
}

const deleting = ref<AdRow | null>(null)
async function confirmDelete() {
  if (!deleting.value) return
  try {
    await $fetch(`/api/v1/admin/ads/${deleting.value.id}`, { method: 'DELETE', credentials: 'include' })
    toast.value = 'Ad deleted.'
  } catch {
    toast.value = ''
  }
  deleting.value = null
  await load()
}

async function onFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploading.value = true
  uploadError.value = ''
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await $fetch<{ url: string }>('/api/v1/admin/media/upload', { method: 'POST', credentials: 'include', body: fd })
    form.image_url = res.url
  } catch {
    uploadError.value = 'Upload failed — is media storage configured? You can paste an image URL directly instead.'
  } finally {
    uploading.value = false
  }
}

function isExpired(row: AdRow) { return row.ends_at !== null && new Date(row.ends_at) < new Date() }
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Site Editor</h1>
      <button v-if="roleAtLeast('admin')" type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="openAdd">+ Add Ad</button>
    </div>
    <SiteEditorTabs active="ads" />

    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-card border border-card-border bg-card p-3">
      <select v-model="slotFilter" class="rounded-lg border border-card-border px-3 py-1.5 text-sm">
        <option value="">Slot: All</option>
        <option v-for="s in slots" :key="s.id" :value="s.key">{{ s.label }}</option>
      </select>
    </div>

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <div v-else-if="filtered.length === 0" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
      No ads {{ slotFilter ? 'in this slot' : 'yet' }}. Click "+ Add Ad" to create one.
    </div>
    <div v-else class="overflow-x-auto rounded-card border border-card-border bg-card">
      <table class="w-full min-w-[900px] text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
            <th class="px-4 py-2.5 font-semibold">Preview</th>
            <th class="px-4 py-2.5 font-semibold">Title</th>
            <th class="px-4 py-2.5 font-semibold">Slot</th>
            <th class="px-4 py-2.5 font-semibold">Type</th>
            <th class="px-4 py-2.5 font-semibold">Schedule</th>
            <th class="px-4 py-2.5 font-semibold">Status</th>
            <th class="px-4 py-2.5 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in filtered" :key="a.id" class="border-b border-card-border last:border-none">
            <td class="px-4 py-2.5"><img :src="a.image_url" :alt="a.title" class="h-10 w-16 rounded object-cover"></td>
            <td class="px-4 py-2.5 font-semibold text-navy">{{ a.title }}</td>
            <td class="px-4 py-2.5 text-muted">{{ slotLabel(a.slot_key) }}</td>
            <td class="px-4 py-2.5"><span class="rounded-pill bg-page px-2 py-0.5 text-[11px] font-bold uppercase text-navy">{{ a.ad_type }}</span></td>
            <td class="px-4 py-2.5 text-xs text-muted">
              {{ fmtDate(a.starts_at) }} &rarr; {{ a.ends_at ? fmtDate(a.ends_at) : 'no end' }}
            </td>
            <td class="px-4 py-2.5">
              <span
                class="rounded-pill px-2 py-0.5 text-[11px] font-bold"
                :class="!a.is_active ? 'bg-page text-muted' : isExpired(a) ? 'bg-amber-50 text-amber-700' : 'bg-emerald-50 text-emerald-700'"
              >
                {{ !a.is_active ? 'Inactive' : isExpired(a) ? 'Expired' : 'Live' }}
              </span>
            </td>
            <td class="px-4 py-2.5">
              <div class="flex items-center gap-3">
                <button v-if="roleAtLeast('admin')" type="button" class="text-xs font-bold text-primary hover:underline" @click="openEdit(a)">Edit</button>
                <button v-if="roleAtLeast('admin')" type="button" class="text-xs font-bold text-red-600 hover:underline" @click="deleting = a">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="editing || adding" class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black/40 p-4">
      <div class="w-full max-w-md rounded-card bg-card p-6">
        <h3 class="text-base font-bold text-navy">{{ editing ? 'Edit Ad' : 'Add Ad' }}</h3>
        <div class="mt-4 flex max-h-[65vh] flex-col gap-3 overflow-y-auto pr-1">
          <div>
            <label class="mb-1 block text-xs font-semibold text-navy">Slot</label>
            <select v-model.number="form.slot_id" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
              <option v-for="s in slots" :key="s.id" :value="s.id">{{ s.label }}</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-semibold text-navy">Ad Type</label>
            <select v-model="form.ad_type" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
              <option value="house">House (self-promotion)</option>
              <option value="sponsor">Sponsor (paid, outsider)</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-semibold text-navy">Title</label>
            <input v-model="form.title" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm" placeholder="Internal label, not shown as ad copy">
          </div>
          <div>
            <label class="mb-1 block text-xs font-semibold text-navy">Image</label>
            <input type="file" accept="image/*" class="w-full text-xs" @change="onFileChange">
            <p v-if="uploading" class="mt-1 text-xs text-muted">Uploading…</p>
            <p v-if="uploadError" class="mt-1 text-xs text-red-600">{{ uploadError }}</p>
            <input v-model="form.image_url" class="mt-1.5 w-full rounded-lg border border-card-border px-3 py-2 text-sm" placeholder="Or paste an image URL directly">
            <img v-if="form.image_url" :src="form.image_url" alt="" class="mt-1.5 h-16 rounded border border-card-border object-contain">
          </div>
          <div>
            <label class="mb-1 block text-xs font-semibold text-navy">Target URL (where the ad links to)</label>
            <input v-model="form.target_url" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm" placeholder="https://...">
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="mb-1 block text-xs font-semibold text-navy">Starts</label>
              <input v-model="form.starts_at" type="datetime-local" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
            </div>
            <div>
              <label class="mb-1 block text-xs font-semibold text-navy">Ends (optional)</label>
              <input v-model="form.ends_at" type="datetime-local" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="mb-1 block text-xs font-semibold text-navy">Sort Order</label>
              <input v-model.number="form.sort_order" type="number" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
            </div>
            <label class="mt-5 flex items-center gap-2 text-sm text-navy">
              <input v-model="form.is_active" type="checkbox" class="h-4 w-4">
              Active
            </label>
          </div>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-card-border px-4 py-2 text-sm font-semibold" @click="closeModal">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="save">Save</button>
        </div>
      </div>
    </div>

    <ConfirmationModal
      v-if="deleting"
      :message="`Delete the ad “${deleting.title}”? This can't be undone.`"
      confirm-label="Delete"
      @confirm="confirmDelete"
      @cancel="deleting = null"
    />

    <SuccessToast v-if="toast" :message="toast" @dismiss="toast = ''" />
  </AdminShell>
</template>
