<script setup lang="ts">
// The Ad Manager wizard — ported from the approved prototype
// (https://claude.ai/artifact/2oJBpefBtXefmJpySw5ene) onto FindRate LK's
// real pages/slots. Same 5 steps, fields, and validation rules as the
// prototype; file inputs upload through the real media endpoint instead
// of the prototype's base64-data-URL preview trick, and the Placement
// step's "pages" are this site's real ad_slots grouped by page_key
// (migrations/008_ad_wizard.sql) instead of the prototype's placeholder
// PAGES array.
useHead({ title: 'Site Editor — Ads — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const { roleAtLeast } = useAdminAuth()

interface AdSlotOption { id: number; key: string; label: string; page_key: string; orientation: 'h' | 'v' | 'overlay'; size: string }
interface CreativeRow { id: number; position: number; media_url: string; target_url: string; alt_text: string | null; poster_url: string | null }
interface PlacementRow { slot_id: number; slot_key: string; page_key: string }
interface AdRow {
  id: number; title: string; ad_type: string; advertiser: string | null
  layout: 'horizontal' | 'vertical'; style: string; settings: Record<string, unknown>
  devices: string; sort_order: number; weight: number; is_active: boolean
  starts_at: string; ends_at: string | null; creatives: CreativeRow[]; placements: PlacementRow[]
}

const PAGE_LABELS: Record<string, string> = {
  sitewide: 'Site-wide (every page)',
  home: 'Home',
  banks: 'Bank Profile',
  products: 'Product & Compare Lists'
}

type StyleKey = 'image' | 'gif' | 'video' | 'slider' | 'shared' | 'overlay'
interface StyleDef { name: string; desc: string; min: number; max: number; accept: string }
const STYLES: Record<StyleKey, StyleDef> = {
  image: { name: 'Image', desc: 'One static image (JPG, PNG, WebP)', min: 1, max: 1, accept: 'image/png,image/jpeg,image/webp' },
  gif: { name: 'GIF', desc: 'One animated GIF', min: 1, max: 1, accept: 'image/gif' },
  video: { name: 'Video', desc: 'Muted autoplay video with a poster image', min: 1, max: 1, accept: 'video/mp4,video/webm' },
  slider: { name: 'Slider', desc: '3 to 5 slides that rotate automatically', min: 3, max: 5, accept: 'image/*' },
  shared: { name: 'Shared slot', desc: '2 or 3 square images or GIFs side by side, small gap between', min: 2, max: 3, accept: 'image/*' },
  overlay: { name: 'Overlay', desc: "Covers the page when it opens; the visitor closes it to continue", min: 1, max: 1, accept: 'image/*,video/mp4,video/webm' }
}
const STEP_NAMES = ['Format', 'Creative', 'Placement', 'Schedule & Rotation', 'Review']

// ===== List state =====
const loading = ref(true)
const error = ref(false)
const slots = ref<AdSlotOption[]>([])
const ads = ref<AdRow[]>([])
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

const slotById = computed(() => Object.fromEntries(slots.value.map((s) => [s.id, s])))
// Pages for the wizard's Placement step: every page group that has at
// least one non-overlay slot, each with its visible ad spaces and its
// dedicated overlay slot id.
const pageGroups = computed(() => {
  const byPage: Record<string, { slots: AdSlotOption[]; overlaySlotId: number | null }> = {}
  for (const s of slots.value) {
    if (!byPage[s.page_key]) byPage[s.page_key] = { slots: [], overlaySlotId: null }
    if (s.orientation === 'overlay') byPage[s.page_key].overlaySlotId = s.id
    else byPage[s.page_key].slots.push(s)
  }
  return Object.entries(byPage)
    .filter(([, g]) => g.slots.length > 0)
    .map(([key, g]) => ({ key, label: PAGE_LABELS[key] ?? key, slots: g.slots, overlaySlotId: g.overlaySlotId }))
})

function placementSummary(row: AdRow): string {
  if (row.style === 'overlay') {
    const pages = [...new Set(row.placements.map((p) => PAGE_LABELS[p.page_key] ?? p.page_key))]
    return `${pages.join(', ')} (overlay)`
  }
  return row.placements.map((p) => slotById.value[p.slot_id]?.label ?? p.slot_key).join(', ')
}

// ===== Wizard state =====
const pad = (n: number) => String(n).padStart(2, '0')
function nowLocal(): string {
  const d = new Date()
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
function toLocalInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

interface CreativeItem { src: string; fileName: string; alt: string; url: string; poster: string; uploading: boolean }
function newItem(): CreativeItem {
  return { src: '', fileName: '', alt: '', url: '', poster: '', uploading: false }
}

interface WizardState {
  editingId: number | null
  step: number
  maxReached: number
  layout: 'h' | 'v'
  style: StyleKey | null
  items: CreativeItem[]
  sliderInterval: number
  sharedGap: number
  overlay: { delay: number; closeAfter: number; freq: string }
  pages: string[]
  slots: number[]
  title: string
  adType: string
  advertiser: string
  start: string
  end: string
  devices: string
  sortOrder: number
  weight: number
  active: boolean
}
function blankWizard(): WizardState {
  return {
    editingId: null, step: 0, maxReached: 0, layout: 'h', style: null, items: [],
    sliderInterval: 5, sharedGap: 12, overlay: { delay: 2, closeAfter: 3, freq: 'session' },
    pages: [], slots: [], title: '', adType: 'house', advertiser: '', start: nowLocal(), end: '',
    devices: 'all', sortOrder: 0, weight: 5, active: true
  }
}
const W = reactive(blankWizard())
const errors = ref<string[]>([])
const showWizard = ref(false)
const saving = ref(false)
// Nothing stopped clicking Next while a file was still mid-upload to
// Supabase Storage — the item's src is still empty at that instant, so
// validation should already block it, but disabling Next/Save outright
// removes the timing window entirely rather than relying on a race
// between the click handler and the upload's own async completion.
const anyUploading = computed(() => W.items.some((it) => it.uploading))

const isUrl = (s: string) => /^https?:\/\/[^\s.]+\.[^\s]+/i.test(s || '')

function openAdd() {
  Object.assign(W, blankWizard())
  errors.value = []
  showWizard.value = true
}
function openEdit(row: AdRow) {
  if (!STYLES[row.style as StyleKey]) return
  Object.assign(W, {
    editingId: row.id, step: 0, maxReached: 4,
    layout: row.layout === 'vertical' ? 'v' : 'h',
    style: row.style as StyleKey,
    items: row.creatives
      .slice()
      .sort((a, b) => a.position - b.position)
      .map((c) => ({ src: c.media_url, fileName: '', alt: c.alt_text || '', url: c.target_url, poster: c.poster_url || '', uploading: false })),
    sliderInterval: Number(row.settings?.interval_seconds) || 5,
    sharedGap: Number(row.settings?.gap_px) || 12,
    overlay: {
      delay: Number(row.settings?.show_after_seconds) || 2,
      closeAfter: Number(row.settings?.close_after_seconds) || 3,
      freq: String(row.settings?.frequency || 'session')
    },
    pages: [...new Set(row.placements.map((p) => p.page_key))],
    slots: row.style === 'overlay' ? [] : row.placements.map((p) => p.slot_id),
    title: row.title, adType: row.ad_type, advertiser: row.advertiser || '',
    start: toLocalInput(row.starts_at), end: toLocalInput(row.ends_at),
    devices: row.devices, sortOrder: row.sort_order, weight: row.weight, active: row.is_active
  })
  errors.value = []
  showWizard.value = true
}
function closeWizard() {
  showWizard.value = false
}

// ===== Validation (mirrors the prototype's validate(step) exactly) =====
function validateStep(step: number): string[] {
  const e: string[] = []
  if (step === 0) {
    if (!W.style) e.push('Choose an ad style.')
  }
  if (step === 1 && W.style) {
    const st = STYLES[W.style]
    if (W.items.length < st.min || W.items.length > st.max) {
      e.push(st.min === st.max ? `${st.name} needs exactly ${st.min} item.` : `${st.name} needs ${st.min} to ${st.max} items (you have ${W.items.length}).`)
    }
    W.items.forEach((it, i) => {
      const lbl = W.items.length > 1 ? `Item ${i + 1}: ` : ''
      if (!it.src) e.push(`${lbl}upload a file or paste a file URL.`)
      if (!isUrl(it.url)) e.push(`${lbl}add a target URL starting with http:// or https://.`)
    })
    if (W.style === 'slider' && (W.sliderInterval < 2 || W.sliderInterval > 30)) e.push('Slide interval must be between 2 and 30 seconds.')
  }
  if (step === 2) {
    if (!W.pages.length) e.push('Choose at least one page.')
    if (W.style !== 'overlay' && !W.slots.length) e.push('Choose at least one ad space on the selected pages.')
    if (W.style === 'overlay' && W.overlay.closeAfter < 0) e.push('Close button delay cannot be negative.')
  }
  if (step === 3) {
    if (!W.title.trim()) e.push('Add a title so you can find this ad later.')
    if (W.adType !== 'house' && !W.advertiser.trim()) e.push('Add the advertiser name for sponsored and partner ads.')
    if (!W.start) e.push('Set a start date and time.')
    if (W.end && W.start && W.end <= W.start) e.push('End must be after start.')
    if (W.weight < 1 || W.weight > 10) e.push('Weight must be between 1 and 10.')
  }
  return e
}

function go(to: number) {
  if (to > W.step) {
    for (let i = W.step; i < to; i++) {
      const e = validateStep(i)
      if (e.length) { W.step = i; errors.value = e; return }
    }
  }
  errors.value = []
  W.step = to
  W.maxReached = Math.max(W.maxReached, to)
}
function back() {
  if (W.step === 0) { closeWizard(); return }
  go(W.step - 1)
}

function selectLayout(v: 'h' | 'v') {
  W.layout = v
  W.slots = []
}
function selectStyle(v: StyleKey) {
  if (W.style !== v) {
    W.style = v
    W.items = Array.from({ length: STYLES[v].min }, newItem)
    if (v === 'overlay') W.slots = []
  }
}
function addItem() {
  if (W.style && W.items.length < STYLES[W.style].max) W.items.push(newItem())
}
function removeItem(i: number) {
  if (W.style && W.items.length > STYLES[W.style].min) W.items.splice(i, 1)
}
function moveItemUp(i: number) {
  if (i > 0) [W.items[i - 1], W.items[i]] = [W.items[i], W.items[i - 1]]
}

async function onItemFileChange(i: number, e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 25 * 1024 * 1024) {
    errors.value = ['File is larger than 25 MB. Compress it and try again.']
    return
  }
  const item = W.items[i]
  item.uploading = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await $fetch<{ url: string }>('/api/v1/admin/media/upload', { method: 'POST', credentials: 'include', body: fd })
    item.src = res.url
    item.fileName = file.name
  } catch (err) {
    errors.value = [fetchErrorMessage(err, 'Upload failed.')]
  } finally {
    item.uploading = false
  }
}

function togglePage(pageKey: string) {
  if (W.pages.includes(pageKey)) {
    W.pages = W.pages.filter((p) => p !== pageKey)
    const group = pageGroups.value.find((g) => g.key === pageKey)
    if (group) W.slots = W.slots.filter((id) => !group.slots.some((s) => s.id === id))
  } else {
    W.pages.push(pageKey)
  }
}
function slotFits(s: AdSlotOption): boolean {
  return s.orientation === W.layout
}
function toggleSlot(id: number) {
  W.slots = W.slots.includes(id) ? W.slots.filter((x) => x !== id) : [...W.slots, id]
}
function fittingSlotIds(pageKey: string): number[] {
  const group = pageGroups.value.find((g) => g.key === pageKey)
  return group ? group.slots.filter(slotFits).map((s) => s.id) : []
}
function allSlotsSelected(pageKey: string): boolean {
  const fit = fittingSlotIds(pageKey)
  return fit.length > 0 && fit.every((id) => W.slots.includes(id))
}
function toggleAllSlots(pageKey: string) {
  const fit = fittingSlotIds(pageKey)
  W.slots = allSlotsSelected(pageKey) ? W.slots.filter((id) => !fit.includes(id)) : [...new Set([...W.slots, ...fit])]
}

function buildSettings(): Record<string, unknown> {
  if (!W.style) return {}
  if (W.style === 'slider') return { interval_seconds: W.sliderInterval }
  if (W.style === 'shared') return { gap_px: W.sharedGap }
  if (W.style === 'overlay') return { show_after_seconds: W.overlay.delay, close_after_seconds: W.overlay.closeAfter, frequency: W.overlay.freq }
  if (W.style === 'video') return { autoplay: true, muted: true, loop: true }
  return {}
}
function buildSlotIds(): number[] {
  if (W.style === 'overlay') {
    return W.pages
      .map((pk) => pageGroups.value.find((g) => g.key === pk)?.overlaySlotId)
      .filter((id): id is number => id !== null && id !== undefined)
  }
  return W.slots
}
function buildPayload() {
  return {
    title: W.title.trim(),
    ad_type: W.adType,
    advertiser: W.adType === 'house' ? '' : W.advertiser.trim(),
    layout: W.layout === 'h' ? 'horizontal' : 'vertical',
    style: W.style,
    settings: buildSettings(),
    devices: W.devices,
    sort_order: W.sortOrder,
    weight: W.weight,
    is_active: W.active,
    starts_at: W.start ? new Date(W.start).toISOString() : undefined,
    ends_at: W.end ? new Date(W.end).toISOString() : null,
    creatives: W.items.map((it, i) => ({
      position: i + 1, media_url: it.src, target_url: it.url, alt_text: it.alt || null,
      ...(W.style === 'video' ? { poster_url: it.poster || null } : {})
    })),
    slot_ids: buildSlotIds()
  }
}

async function save() {
  if (saving.value || anyUploading.value) return
  for (let i = 0; i < 4; i++) {
    const e = validateStep(i)
    if (e.length) { W.step = i; errors.value = e; return }
  }
  saving.value = true
  try {
    if (W.editingId) {
      await $fetch(`/api/v1/admin/ads/${W.editingId}`, { method: 'PATCH', credentials: 'include', body: buildPayload() })
      toast.value = 'Ad updated.'
    } else {
      await $fetch('/api/v1/admin/ads', { method: 'POST', credentials: 'include', body: buildPayload() })
      toast.value = 'Ad created.'
    }
    closeWizard()
    await load()
  } catch (err) {
    errors.value = [fetchErrorMessage(err, 'Failed to save the ad.')]
  } finally {
    saving.value = false
  }
}

const deleting = ref<AdRow | null>(null)
async function confirmDelete() {
  if (!deleting.value) return
  try {
    await $fetch(`/api/v1/admin/ads/${deleting.value.id}`, { method: 'DELETE', credentials: 'include' })
    toast.value = 'Ad deleted.'
  } catch (err) {
    toast.value = ''
    console.error(fetchErrorMessage(err))
  }
  deleting.value = null
  await load()
}
</script>

<template>
  <AdminShell>
    <div class="mb-5 flex items-center justify-between">
      <h1 class="text-xl font-bold text-navy">Site Editor</h1>
      <button v-if="roleAtLeast('admin')" type="button" class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90" @click="openAdd">+ Add Ad</button>
    </div>
    <SiteEditorTabs active="ads" />

    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" @retry="load" />
    <div v-else-if="ads.length === 0" class="rounded-card border border-card-border bg-card p-10 text-center text-sm text-muted">
      No ads yet. Click "+ Add Ad" to create one.
    </div>
    <div v-else class="overflow-x-auto rounded-card border border-card-border bg-card">
      <table class="w-full min-w-[920px] text-sm">
        <thead>
          <tr class="border-b border-card-border text-left text-[11px] uppercase tracking-wide text-muted">
            <th class="px-4 py-2.5 font-semibold">Preview</th>
            <th class="px-4 py-2.5 font-semibold">Title</th>
            <th class="px-4 py-2.5 font-semibold">Style</th>
            <th class="px-4 py-2.5 font-semibold">Placement</th>
            <th class="px-4 py-2.5 font-semibold">Rotation</th>
            <th class="px-4 py-2.5 font-semibold">Status</th>
            <th class="px-4 py-2.5 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in ads" :key="a.id" class="border-b border-card-border last:border-none">
            <td class="px-4 py-2.5"><img v-if="a.creatives[0]" :src="a.creatives[0].media_url" :alt="a.title" class="h-10 w-16 rounded object-cover"></td>
            <td class="px-4 py-2.5 font-semibold text-navy">{{ a.title }}</td>
            <td class="px-4 py-2.5">
              <span class="rounded-pill bg-page px-2 py-0.5 text-[11px] font-bold uppercase text-navy">{{ a.style }}</span>
              <span class="ml-1 text-[11px] text-muted">{{ a.ad_type }}{{ a.advertiser ? ` — ${a.advertiser}` : '' }}</span>
            </td>
            <td class="px-4 py-2.5 text-xs text-muted">{{ placementSummary(a) }}</td>
            <td class="px-4 py-2.5 text-xs text-muted">sort {{ a.sort_order }}, weight {{ a.weight }}</td>
            <td class="px-4 py-2.5">
              <span class="rounded-pill px-2 py-0.5 text-[11px] font-bold" :class="a.is_active ? 'bg-emerald-50 text-emerald-700' : 'bg-page text-muted'">
                {{ a.is_active ? 'Active' : 'Paused' }}
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

    <!-- ===== Wizard modal ===== -->
    <div v-if="showWizard" class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black/40 p-4">
      <div class="flex max-h-[90vh] w-full max-w-2xl flex-col rounded-card bg-card shadow-xl">
        <div class="border-b border-card-border px-6 pt-5">
          <h3 class="text-lg font-bold text-navy">{{ W.editingId ? 'Edit Ad' : 'Add Ad' }}</h3>
          <nav class="mt-3 flex gap-1 overflow-x-auto">
            <button
              v-for="(name, i) in STEP_NAMES" :key="name" type="button"
              class="flex shrink-0 items-center gap-1.5 border-b-2 px-2.5 py-2 text-xs font-semibold"
              :class="i === W.step ? 'border-primary text-primary' : i < W.maxReached ? 'border-transparent text-emerald-700' : 'border-transparent text-muted'"
              :disabled="i > W.maxReached"
              @click="go(i)"
            >
              <span
                class="flex h-4 w-4 items-center justify-center rounded-full text-[10px]"
                :class="i === W.step ? 'bg-primary text-white' : i < W.maxReached ? 'bg-emerald-100 text-emerald-700' : 'bg-page text-muted'"
              >{{ i < W.maxReached && i !== W.step ? '✓' : i + 1 }}</span>
              {{ name }}
            </button>
          </nav>
        </div>

        <div class="flex-1 overflow-y-auto px-6 py-5">
          <ul v-if="errors.length" class="mb-4 rounded-lg bg-red-50 px-3.5 py-2.5 text-xs text-red-700">
            <li v-for="e in errors" :key="e">{{ e }}</li>
          </ul>

          <!-- Step 0: Format -->
          <div v-if="W.step === 0" class="flex flex-col gap-5">
            <div>
              <h4 class="text-sm font-bold text-navy">Layout</h4>
              <p class="mb-2 text-xs text-muted">Decides which ad spaces this ad can go into.</p>
              <div class="grid grid-cols-2 gap-2.5">
                <button
                  type="button" class="rounded-lg border-2 p-3 text-left"
                  :class="W.layout === 'h' ? 'border-primary bg-badge-bg' : 'border-card-border'"
                  @click="selectLayout('h')"
                >
                  <b class="block text-sm text-navy">Horizontal</b>
                  <small class="text-xs text-muted">Wide banners across the page</small>
                </button>
                <button
                  type="button" class="rounded-lg border-2 p-3 text-left"
                  :class="W.layout === 'v' ? 'border-primary bg-badge-bg' : 'border-card-border'"
                  @click="selectLayout('v')"
                >
                  <b class="block text-sm text-navy">Vertical</b>
                  <small class="text-xs text-muted">Tall ads in side rails</small>
                </button>
              </div>
            </div>
            <div>
              <h4 class="text-sm font-bold text-navy">Style</h4>
              <p class="mb-2 text-xs text-muted">What the visitor sees.</p>
              <div class="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
                <button
                  v-for="(def, key) in STYLES" :key="key" type="button"
                  class="rounded-lg border-2 p-3 text-left"
                  :class="W.style === key ? 'border-primary bg-badge-bg' : 'border-card-border'"
                  @click="selectStyle(key as StyleKey)"
                >
                  <b class="block text-sm text-navy">{{ def.name }}</b>
                  <small class="text-xs text-muted">{{ def.desc }}</small>
                </button>
              </div>
            </div>
          </div>

          <!-- Step 1: Creative -->
          <div v-else-if="W.step === 1 && W.style" class="flex flex-col gap-4">
            <p class="text-xs text-muted">
              {{ W.items.length }} of {{ STYLES[W.style].max }} used, minimum {{ STYLES[W.style].min }}.
              <span v-if="STYLES[W.style].max > 1">Each {{ W.style === 'slider' ? 'slide' : 'tile' }} has its own link.</span>
            </p>
            <div v-if="W.style === 'shared'" class="rounded-lg bg-badge-bg px-3 py-2 text-xs text-navy">
              Use square files (1:1, e.g. 300×300). Tiles sit {{ W.layout === 'h' ? 'in a row' : 'in a column' }} inside one ad space.
            </div>
            <div v-if="W.style === 'overlay'" class="rounded-lg bg-badge-bg px-3 py-2 text-xs text-navy">
              Recommended: {{ W.layout === 'h' ? '800×450 landscape' : '450×800 portrait' }}. On phones it scales to fit the screen.
            </div>

            <div v-for="(it, i) in W.items" :key="i" class="rounded-lg border border-card-border bg-page p-3.5">
              <div class="mb-2 flex items-center justify-between text-xs font-semibold text-navy">
                <span>{{ STYLES[W.style].max > 1 ? `${W.style === 'slider' ? 'Slide' : 'Tile'} ${i + 1}` : 'Ad file' }}</span>
                <div class="flex gap-2">
                  <button v-if="STYLES[W.style].max > 1 && W.items.length > STYLES[W.style].min" type="button" class="text-[11px] font-bold text-red-600" @click="removeItem(i)">Remove</button>
                  <button v-if="STYLES[W.style].max > 1 && i > 0" type="button" class="text-[11px] font-bold text-primary" @click="moveItemUp(i)">Move up</button>
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <input type="file" :accept="STYLES[W.style].accept" class="text-xs" @change="onItemFileChange(i, $event)">
                <span v-if="it.uploading" class="text-xs text-muted">Uploading…</span>
                <span v-else-if="it.fileName" class="text-xs text-muted">{{ it.fileName }}</span>
              </div>
              <input v-model="it.src" type="url" placeholder="Or paste a file URL" class="mt-2 w-full rounded-lg border border-card-border px-3 py-1.5 text-sm">
              <img v-if="it.src && W.style !== 'video'" :src="it.src" alt="" class="mt-2 max-h-28 rounded border border-dashed border-card-border object-contain">
              <video v-if="it.src && W.style === 'video'" :src="it.src" class="mt-2 max-h-28 w-full rounded border border-dashed border-card-border" muted controls />
              <div class="mt-2 grid grid-cols-2 gap-2.5">
                <div><label class="mb-0.5 block text-[11px] font-semibold text-navy">Target URL</label><input v-model="it.url" type="url" placeholder="https://..." class="w-full rounded-lg border border-card-border px-3 py-1.5 text-sm"></div>
                <div><label class="mb-0.5 block text-[11px] font-semibold text-navy">Alt text</label><input v-model="it.alt" type="text" placeholder="Screen-reader description" class="w-full rounded-lg border border-card-border px-3 py-1.5 text-sm"></div>
              </div>
              <div v-if="W.style === 'video'" class="mt-2">
                <label class="mb-0.5 block text-[11px] font-semibold text-navy">Poster image URL (shown before the video loads)</label>
                <input v-model="it.poster" type="url" placeholder="https://..." class="w-full rounded-lg border border-card-border px-3 py-1.5 text-sm">
              </div>
            </div>
            <button v-if="STYLES[W.style].max > 1 && W.items.length < STYLES[W.style].max" type="button" class="self-start rounded-lg border border-card-border px-3 py-1.5 text-xs font-bold text-navy" @click="addItem">
              Add {{ W.style === 'slider' ? 'slide' : 'tile' }}
            </button>

            <div v-if="W.style === 'slider'">
              <label class="mb-0.5 block text-xs font-semibold text-navy">Seconds per slide</label>
              <input v-model.number="W.sliderInterval" type="number" min="2" max="30" class="w-32 rounded-lg border border-card-border px-3 py-1.5 text-sm">
            </div>
            <div v-if="W.style === 'shared'">
              <label class="mb-0.5 block text-xs font-semibold text-navy">Gap between tiles (px)</label>
              <input v-model.number="W.sharedGap" type="number" min="4" max="32" class="w-32 rounded-lg border border-card-border px-3 py-1.5 text-sm">
            </div>
          </div>

          <!-- Step 2: Placement -->
          <div v-else-if="W.step === 2" class="flex flex-col gap-4">
            <div>
              <h4 class="text-sm font-bold text-navy">Pages</h4>
              <p class="mb-2 text-xs text-muted">Pick every page this ad should appear on.</p>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="g in pageGroups" :key="g.key" type="button"
                  class="rounded-pill border-2 px-3.5 py-1.5 text-xs font-semibold"
                  :class="W.pages.includes(g.key) ? 'border-primary bg-badge-bg text-primary' : 'border-card-border text-navy'"
                  @click="togglePage(g.key)"
                >
                  {{ g.label }}
                </button>
              </div>
            </div>

            <div v-if="W.style === 'overlay'" class="rounded-lg bg-badge-bg px-3.5 py-3 text-xs text-navy">
              Overlays cover the whole page, so they don't use an ad space.
            </div>
            <div v-if="W.style === 'overlay'" class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <label class="mb-0.5 block text-xs font-semibold text-navy">Show after (seconds)</label>
                <input v-model.number="W.overlay.delay" type="number" min="0" max="60" class="w-full rounded-lg border border-card-border px-3 py-1.5 text-sm">
              </div>
              <div>
                <label class="mb-0.5 block text-xs font-semibold text-navy">Close button appears after (s)</label>
                <input v-model.number="W.overlay.closeAfter" type="number" min="0" max="15" class="w-full rounded-lg border border-card-border px-3 py-1.5 text-sm">
              </div>
              <div>
                <label class="mb-0.5 block text-xs font-semibold text-navy">Show to each visitor</label>
                <select v-model="W.overlay.freq" class="w-full rounded-lg border border-card-border px-3 py-1.5 text-sm">
                  <option value="every">Every page view</option>
                  <option value="session">Once per visit</option>
                  <option value="day">Once per day</option>
                  <option value="once">Only once</option>
                </select>
              </div>
            </div>

            <template v-else-if="W.pages.length">
              <h4 class="text-sm font-bold text-navy">Ad spaces</h4>
              <p class="text-xs text-muted">Only {{ W.layout === 'h' ? 'horizontal' : 'vertical' }} spaces fit this ad. Change the layout in step 1 to use the others.</p>
              <div v-for="g in pageGroups.filter((pg) => W.pages.includes(pg.key))" :key="g.key" class="overflow-hidden rounded-lg border border-card-border">
                <div class="flex items-center justify-between bg-page px-3.5 py-2">
                  <span class="text-sm font-semibold text-navy">{{ g.label }}</span>
                  <button v-if="fittingSlotIds(g.key).length" type="button" class="text-[11px] font-bold text-primary" @click="toggleAllSlots(g.key)">
                    {{ allSlotsSelected(g.key) ? 'Clear' : 'Select all that fit' }}
                  </button>
                </div>
                <label
                  v-for="s in g.slots" :key="s.id"
                  class="flex items-center gap-2.5 border-t border-card-border px-3.5 py-2.5"
                  :class="slotFits(s) ? 'cursor-pointer' : 'cursor-not-allowed opacity-50'"
                >
                  <input type="checkbox" class="h-4 w-4" :checked="W.slots.includes(s.id)" :disabled="!slotFits(s)" @change="toggleSlot(s.id)">
                  <div>
                    <div class="text-sm text-navy">{{ s.label }}</div>
                    <div class="text-[11px] text-muted">{{ s.orientation === 'h' ? 'Horizontal' : 'Vertical' }}, {{ s.size }}{{ slotFits(s) ? '' : ' — does not fit this layout' }}</div>
                  </div>
                </label>
              </div>
            </template>
          </div>

          <!-- Step 3: Schedule & Rotation -->
          <div v-else-if="W.step === 3" class="flex flex-col gap-4">
            <div>
              <label class="mb-1 block text-xs font-semibold text-navy">Title</label>
              <input v-model="W.title" type="text" placeholder="Internal label, not shown to visitors" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="mb-1 block text-xs font-semibold text-navy">Ad type</label>
                <select v-model="W.adType" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
                  <option value="house">House (self-promotion)</option>
                  <option value="sponsored">Sponsored (paid)</option>
                  <option value="partner">Partner</option>
                </select>
              </div>
              <div v-if="W.adType !== 'house'">
                <label class="mb-1 block text-xs font-semibold text-navy">Advertiser</label>
                <input v-model="W.advertiser" type="text" placeholder="Company name" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm">
              </div>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div><label class="mb-1 block text-xs font-semibold text-navy">Starts</label><input v-model="W.start" type="datetime-local" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
              <div><label class="mb-1 block text-xs font-semibold text-navy">Ends (optional)</label><input v-model="W.end" type="datetime-local" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
            </div>
            <div>
              <label class="mb-1 block text-xs font-semibold text-navy">Show on</label>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="[v, n] in [['all', 'All devices'], ['desktop', 'Desktop only'], ['mobile', 'Mobile only']]" :key="v" type="button"
                  class="rounded-pill border-2 px-3.5 py-1.5 text-xs font-semibold"
                  :class="W.devices === v ? 'border-primary bg-badge-bg text-primary' : 'border-card-border text-navy'"
                  @click="W.devices = v"
                >{{ n }}</button>
              </div>
            </div>
            <div class="rounded-lg border border-card-border bg-page p-3.5">
              <h4 class="text-sm font-bold text-navy">When a space has more than one ad</h4>
              <p class="mb-2 text-xs text-muted">Ads in the same space rotate across page loads. Weight decides how often this ad appears compared with the others (weight 6 vs 3 means twice as often).</p>
              <div class="grid grid-cols-2 gap-3">
                <div><label class="mb-1 block text-xs font-semibold text-navy">Sort order</label><input v-model.number="W.sortOrder" type="number" min="0" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
                <div><label class="mb-1 block text-xs font-semibold text-navy">Weight (1–10)</label><input v-model.number="W.weight" type="number" min="1" max="10" class="w-full rounded-lg border border-card-border px-3 py-2 text-sm"></div>
              </div>
              <label class="mt-3 flex items-center gap-2 text-sm font-medium text-navy">
                <input v-model="W.active" type="checkbox" class="h-4 w-4"> Active
              </label>
            </div>
          </div>

          <!-- Step 4: Review -->
          <div v-else-if="W.step === 4">
            <table class="w-full text-sm">
              <tbody>
                <tr class="border-b border-card-border"><td class="w-1/3 py-2 text-muted">Title</td><td class="py-2 text-navy">{{ W.title }}</td></tr>
                <tr class="border-b border-card-border"><td class="py-2 text-muted">Ad type</td><td class="py-2 text-navy">{{ W.adType }}{{ W.advertiser ? ` — ${W.advertiser}` : '' }}</td></tr>
                <tr class="border-b border-card-border"><td class="py-2 text-muted">Format</td><td class="py-2 text-navy">{{ W.layout === 'h' ? 'Horizontal' : 'Vertical' }}, {{ W.style ? STYLES[W.style].name : '' }}</td></tr>
                <tr class="border-b border-card-border"><td class="py-2 text-muted">Files</td><td class="py-2 text-navy">{{ W.items.length }}</td></tr>
                <tr class="border-b border-card-border">
                  <td class="py-2 text-muted">Placement</td>
                  <td class="py-2 text-navy">
                    <span v-if="W.style === 'overlay'">{{ W.pages.map((p) => PAGE_LABELS[p] ?? p).join(', ') }} (overlay)</span>
                    <span v-else>{{ W.slots.map((id) => slotById[id]?.label).filter(Boolean).join(', ') }}</span>
                  </td>
                </tr>
                <tr class="border-b border-card-border"><td class="py-2 text-muted">Runs</td><td class="py-2 text-navy">{{ W.start.replace('T', ' ') }} &rarr; {{ W.end ? W.end.replace('T', ' ') : 'no end date' }}</td></tr>
                <tr class="border-b border-card-border"><td class="py-2 text-muted">Devices</td><td class="py-2 text-navy">{{ W.devices }}</td></tr>
                <tr class="border-b border-card-border"><td class="py-2 text-muted">Rotation</td><td class="py-2 text-navy">sort order {{ W.sortOrder }}, weight {{ W.weight }}</td></tr>
                <tr><td class="py-2 text-muted">Status</td><td class="py-2 text-navy">{{ W.active ? 'Active' : 'Paused' }}</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="flex justify-between gap-2 border-t border-card-border bg-page px-6 py-3.5">
          <button type="button" class="rounded-lg border border-card-border bg-card px-4 py-2 text-sm font-semibold text-navy" @click="back">{{ W.step === 0 ? 'Cancel' : 'Back' }}</button>
          <button
            v-if="W.step < 4" type="button" :disabled="anyUploading"
            class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
            @click="go(W.step + 1)"
          >
            {{ anyUploading ? 'Uploading…' : 'Next' }}
          </button>
          <button
            v-else type="button" :disabled="anyUploading || saving"
            class="rounded-lg bg-primary px-4 py-2 text-sm font-bold text-white hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
            @click="save"
          >
            {{ saving ? 'Saving…' : anyUploading ? 'Uploading…' : (W.editingId ? 'Save changes' : 'Save ad') }}
          </button>
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
