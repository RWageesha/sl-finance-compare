<script setup lang="ts">
// A real, functional correction form. There's no backend "reports"
// endpoint in this project yet, so submitting opens a pre-filled GitHub
// issue with every field's value — the same reporting mechanism every
// other "Report Incorrect Information" link on the site already uses,
// just now collected through a real form instead of a single link.
import { DIRECTORY_BANKS } from '~/utils/bankDirectory'
import { PRODUCT_TYPE_ORDER, PRODUCT_TYPES } from '~/config/productTypes'

useHead({ title: 'Report Incorrect Information — FindRate LK' })

const route = useRoute()

const CATEGORY_OPTIONS = PRODUCT_TYPE_ORDER.map((slug) => PRODUCT_TYPES[slug].switcherLabel)
const ISSUE_TYPES = ['Rate', 'Fee', 'Product Details', 'Eligibility', 'Other']

// Pre-fill + lock when arriving from a specific product's "Report
// Incorrect Information" link (?bank=hnb&category=Fixed+Deposits&
// product=12-Month+Fixed+Deposit&currentValue=8.25%25+p.a.)
const prefill = computed(() => {
  const bankSlug = route.query.bank ? String(route.query.bank) : ''
  const bank = DIRECTORY_BANKS.find((b) => b.slug === bankSlug)
  return {
    bank: bank?.displayName ?? '',
    category: route.query.category ? String(route.query.category) : '',
    product: route.query.product ? String(route.query.product) : '',
    currentValue: route.query.currentValue ? String(route.query.currentValue) : ''
  }
})
const isLocked = ref(false)
watch(prefill, (p) => { isLocked.value = !!(p.bank && p.category) }, { immediate: true })

const form = reactive({
  bank: '',
  category: '',
  issueType: 'Rate',
  currentValue: '',
  correctValue: '',
  sourceUrl: '',
  description: ''
})
watch(prefill, (p) => {
  form.bank = p.bank
  form.category = p.category
  form.currentValue = p.currentValue
}, { immediate: true })

function startNewReport() {
  isLocked.value = false
  form.bank = ''
  form.category = ''
  form.currentValue = ''
  navigateTo('/report-issue')
}

const submitted = ref(false)
const errors = ref<string[]>([])

function validate(): boolean {
  const missing: string[] = []
  if (!form.bank) missing.push('Select Bank')
  if (!form.category) missing.push('Select Product / Category')
  if (!form.correctValue.trim()) missing.push('Correct Value')
  if (!form.sourceUrl.trim()) missing.push('Source URL')
  errors.value = missing
  return missing.length === 0
}

const submitError = ref('')

// Submits to the real admin-reviewed queue (Admin -> User Reports) —
// this is the actual reporting mechanism now; the GitHub-issue link this
// used before real backend support existed has been retired in favor of
// it actually reaching an admin.
async function submit() {
  if (!validate()) return
  submitError.value = ''
  const productBit = prefill.value.product ? ` — ${prefill.value.product}` : ''
  try {
    await $fetch('/api/v1/user-reports', {
      method: 'POST',
      body: {
        bank_name: form.bank,
        product_label: `${form.category}${productBit}`,
        issue_type: form.issueType,
        current_value: form.currentValue,
        correct_value: form.correctValue,
        source_url: form.sourceUrl,
        description: form.description
      }
    })
    submitted.value = true
  } catch {
    submitError.value = 'Failed to submit your report — please try again.'
  }
}
</script>

<template>
  <div>
    <AppHeader />

    <main class="min-h-screen bg-page px-4 py-8 sm:px-6">
      <div class="mx-auto max-w-[640px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&rsaquo;</span>
          <NuxtLink to="/data" class="hover:text-primary">Data</NuxtLink>
          <span>&rsaquo;</span>
          <span class="font-semibold text-primary">Report Issue</span>
        </nav>

        <div class="text-center">
          <h1 class="text-[28px] font-bold text-navy">Report Incorrect Information</h1>
          <p class="mt-2 text-sm text-muted">Submit a correction request below. Reports are evaluated and updated within 24 business hours.</p>
        </div>

        <form class="mt-6 rounded-[16px] border border-card-border bg-card p-7 shadow-sm" @submit.prevent="submit">
          <p v-if="isLocked" class="mb-5 rounded-lg bg-badge-bg px-3.5 py-2.5 text-xs text-navy">
            Reporting on: <strong>{{ prefill.bank }} — {{ prefill.product || prefill.category }}</strong>.
            Not the right product? <button type="button" class="font-semibold text-primary hover:underline" @click="startNewReport">Start a new report</button>
          </p>

          <div class="flex flex-col gap-4">
            <div>
              <label class="mb-1 block text-sm font-semibold text-navy">Select Bank <span class="text-red-500">*</span></label>
              <select v-model="form.bank" :disabled="isLocked" class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy disabled:bg-page disabled:text-muted">
                <option value="" disabled>Choose a bank</option>
                <option v-for="b in DIRECTORY_BANKS" :key="b.slug" :value="b.displayName">{{ b.displayName }}</option>
              </select>
            </div>

            <div>
              <label class="mb-1 block text-sm font-semibold text-navy">Select Product / Category <span class="text-red-500">*</span></label>
              <select v-model="form.category" :disabled="isLocked" class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy disabled:bg-page disabled:text-muted">
                <option value="" disabled>Choose a category</option>
                <option v-for="c in CATEGORY_OPTIONS" :key="c" :value="c">{{ c }}</option>
              </select>
            </div>

            <div>
              <label class="mb-2 block text-sm font-semibold text-navy">Issue Type <span class="text-red-500">*</span></label>
              <div class="flex flex-wrap gap-x-5 gap-y-2">
                <label v-for="t in ISSUE_TYPES" :key="t" class="flex items-center gap-1.5 text-[13px] text-navy">
                  <input v-model="form.issueType" type="radio" :value="t" name="issueType" class="h-3.5 w-3.5 accent-primary">
                  {{ t }}
                </label>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-sm font-semibold text-navy">Current Displayed Value</label>
                <input v-model="form.currentValue" type="text" readonly placeholder="e.g. 12.50% p.a." class="w-full rounded-lg border border-card-border bg-page px-3 py-2.5 text-sm text-muted">
              </div>
              <div>
                <label class="mb-1 block text-sm font-semibold text-navy">Correct Value <span class="text-red-500">*</span></label>
                <input v-model="form.correctValue" type="text" placeholder="e.g. 11.75%" class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy">
              </div>
            </div>

            <div>
              <label class="mb-1 block text-sm font-semibold text-navy">Source URL (Verification Link) <span class="text-red-500">*</span></label>
              <input v-model="form.sourceUrl" type="url" placeholder="https://..." class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy">
            </div>

            <div>
              <label class="mb-1 block text-sm font-semibold text-navy">Additional Details / Description</label>
              <textarea
                v-model="form.description"
                rows="4"
                placeholder="e.g. The bank revised the rates this morning. The 12.5% rate is now only applicable for senior citizens. Standard rates were dropped to 11.75%."
                class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy"
              />
            </div>

            <ul v-if="errors.length" class="rounded-lg bg-red-50 px-3.5 py-2.5 text-xs text-red-700">
              <li v-for="e in errors" :key="e">Please complete: {{ e }}</li>
            </ul>

            <button type="submit" class="mt-1 w-full rounded-lg bg-primary py-3 text-sm font-bold text-white transition hover:bg-primary/90">
              Submit Report
            </button>

            <p v-if="submitted" class="flex items-center gap-2 rounded-lg bg-emerald-50 px-3.5 py-2.5 text-[13.5px] font-semibold text-emerald-700">
              <span>&#10003;</span> Thank you! We will review within 24 hours.
            </p>
            <p v-if="submitError" class="rounded-lg bg-red-50 px-3.5 py-2.5 text-xs text-red-700">{{ submitError }}</p>
          </div>
        </form>
      </div>
    </main>

    <AppFooter />
  </div>
</template>
