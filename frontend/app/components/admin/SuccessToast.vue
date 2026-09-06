<script setup lang="ts">
// Auto-dismissing toast by default; pass persistent to keep it up with a
// "Continue" button instead (used the couple of places this doubles as a
// small confirmation rather than a passive notice).
const props = withDefaults(defineProps<{ message: string; persistent?: boolean }>(), { persistent: false })
const emit = defineEmits<{ dismiss: [] }>()
const visible = ref(true)

if (!props.persistent) {
  setTimeout(() => {
    visible.value = false
    emit('dismiss')
  }, 3000)
}
</script>

<template>
  <Transition name="fade">
    <div
      v-if="visible"
      class="fixed bottom-6 right-6 z-50 flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 shadow-lg"
    >
      <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-xs font-bold text-white">&#10003;</span>
      <div>
        <p class="text-sm font-bold text-emerald-800">Success!</p>
        <p class="text-xs text-emerald-700">{{ message }}</p>
      </div>
      <button
        v-if="persistent"
        type="button"
        class="ml-2 rounded-lg bg-primary px-3 py-1.5 text-xs font-bold text-white hover:bg-primary/90"
        @click="visible = false; emit('dismiss')"
      >
        Continue
      </button>
    </div>
  </Transition>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
