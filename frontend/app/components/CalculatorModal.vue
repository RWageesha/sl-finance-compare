<script setup lang="ts">
// Generic overlay shell shared by FdCalculatorModal.vue and
// LoanCalculatorModal.vue — just chrome (backdrop, card, eyebrow/title,
// close button); each calculator owns its own fields/formula in its slot.
defineProps<{
  eyebrow: string
  title: string
}>()
const emit = defineEmits<{ close: [] }>()

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <div class="modal-overlay" @click.self="emit('close')">
      <div class="modal-card" role="dialog" aria-modal="true">
        <div class="modal-header">
          <p class="modal-eyebrow">{{ eyebrow }}</p>
          <button type="button" class="modal-close" aria-label="Close" @click="emit('close')">&times;</button>
        </div>
        <h2 class="modal-title">{{ title }}</h2>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(10, 13, 26, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  z-index: 100;
}
.modal-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  width: 100%;
  max-width: 340px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}
.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}
.modal-eyebrow {
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--accent);
}
.modal-close {
  background: none;
  border: none;
  font-size: 1.3rem;
  line-height: 1;
  color: var(--muted);
  cursor: pointer;
  padding: 0;
}
.modal-close:hover {
  color: var(--text);
}
.modal-title {
  margin: 0.15rem 0 1.1rem;
  font-size: 1.05rem;
  font-weight: 700;
}
</style>
