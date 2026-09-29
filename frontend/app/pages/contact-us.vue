<script setup lang="ts">
useHead({ title: 'Contact Us — FindRate LK' })

const { fetchPageContent } = useSiteContent()
const body = ref('')
onMounted(async () => {
  const blocks = await fetchPageContent('contact-us').catch(() => [])
  body.value = blocks.find((b) => b.content_key === 'contact-us-body')?.body ?? ''
})
</script>

<template>
  <div>
    <AppHeader />
    <main class="min-h-screen bg-page px-4 py-10 sm:px-6">
      <div class="mx-auto max-w-[720px]">
        <nav class="mb-4 flex flex-wrap items-center gap-1.5 text-xs text-muted" aria-label="Breadcrumb">
          <NuxtLink to="/" class="hover:text-primary">Home</NuxtLink>
          <span>&rsaquo;</span>
          <span class="font-semibold text-primary">Contact Us</span>
        </nav>
        <h1 class="text-[28px] font-bold text-navy">Contact Us</h1>
        <div class="mt-5 rounded-[16px] border border-card-border bg-card p-7 shadow-sm">
          <p class="whitespace-pre-line text-sm leading-relaxed text-navy">{{ body }}</p>
          <NuxtLink to="/report-issue" class="mt-4 inline-block text-sm font-bold text-primary hover:underline">
            Report Incorrect Information &rarr;
          </NuxtLink>
        </div>
      </div>
    </main>
    <AppFooter />
  </div>
</template>
