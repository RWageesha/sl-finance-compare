<script setup lang="ts">
definePageMeta({ layout: false })
useHead({ title: 'Admin Login — FindRate LK', meta: [{ name: 'robots', content: 'noindex, nofollow' }] })

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await $fetch('/api/v1/admin/login', {
      method: 'POST',
      body: { username: username.value, password: password.value },
      credentials: 'include'
    })
    await navigateTo('/admin/dashboard')
  } catch (e: any) {
    error.value = e?.data?.error || 'Invalid username or password'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-page px-4">
    <div class="w-full max-w-sm rounded-card border border-card-border bg-card p-8 shadow-sm">
      <div class="text-center">
        <img src="/brand/logo-black.png" alt="FindRate LK" class="mx-auto h-8 w-auto">
        <p class="mt-2 text-xs font-bold uppercase tracking-wider text-muted">Admin Portal</p>
      </div>

      <form class="mt-6 flex flex-col gap-4" @submit.prevent="submit">
        <div>
          <label class="mb-1 block text-sm font-semibold text-navy">Username</label>
          <input v-model="username" type="text" required autocomplete="username" class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy">
        </div>
        <div>
          <label class="mb-1 block text-sm font-semibold text-navy">Password</label>
          <input v-model="password" type="password" required autocomplete="current-password" class="w-full rounded-lg border border-card-border bg-white px-3 py-2.5 text-sm text-navy">
        </div>

        <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700">{{ error }}</p>

        <button type="submit" :disabled="loading" class="mt-1 rounded-lg bg-primary py-2.5 text-sm font-bold text-white transition hover:bg-primary/90 disabled:opacity-60">
          {{ loading ? 'Signing In…' : 'Sign In' }}
        </button>
      </form>
    </div>
  </div>
</template>
