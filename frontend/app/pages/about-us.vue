<script setup lang="ts">
useHead({ title: 'About Us — FindRate LK' })

const { fetchPageContent } = useSiteContent()
const body = ref('')
onMounted(async () => {
  const blocks = await fetchPageContent('about-us').catch(() => [])
  body.value = blocks.find((b) => b.content_key === 'about-us-body')?.body ?? ''
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
          <span class="font-semibold text-primary">About Us</span>
        </nav>
        <h1 class="text-[28px] font-bold text-navy">About Us</h1>
        <div class="mt-5 rounded-[16px] border border-card-border bg-card p-7 shadow-sm">
          <p class="whitespace-pre-line text-sm leading-relaxed text-navy">{{ body }}</p>
        </div>
      </div>
    </main>
    <AppFooter />
  </div>
</template>
