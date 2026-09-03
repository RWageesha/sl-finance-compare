<script setup lang="ts">
defineProps<{
  icon: 'bank' | 'home' | 'savings' | 'card' | 'coin'
  color: 'fd' | 'housing' | 'savings' | 'personal' | 'gold' | 'muted'
  label: string
  desc: string
  href?: string
}>()
</script>

<template>
  <NuxtLink v-if="href" :to="href" class="product-card">
    <div class="icon" :class="`icon-${color}`"><Icon :name="icon" /></div>
    <div>
      <h3>{{ label }}</h3>
      <p>{{ desc }}</p>
    </div>
  </NuxtLink>
  <div v-else class="product-card disabled">
    <div class="icon" :class="`icon-${color}`"><Icon :name="icon" /></div>
    <div>
      <h3>{{ label }}</h3>
      <p>{{ desc }}</p>
      <span class="soon-tag">Coming soon</span>
    </div>
  </div>
</template>

<style scoped>
.product-card {
  display: flex;
  align-items: flex-start;
  gap: 0.8rem;
  padding: 1.1rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  text-decoration: none;
  color: inherit;
  transition: border-color 0.15s, transform 0.15s;
}
.product-card:hover:not(.disabled) {
  border-color: var(--accent);
  transform: translateY(-1px);
}
.product-card.disabled {
  opacity: 0.6;
  cursor: default;
}
.product-card .icon {
  width: 38px;
  height: 38px;
  flex: none;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.product-card .icon svg {
  width: 18px;
  height: 18px;
}
.icon-fd { background: #e6f0ff; color: #1a4fb4; }
.icon-housing { background: #e8e6fd; color: #3f2fb0; }
.icon-savings { background: #e0f7fa; color: #00707a; }
.icon-personal { background: #e1f0ff; color: #1660b0; }
.icon-gold { background: #fff6da; color: #8a6a00; }
.icon-muted { background: var(--bg); color: var(--muted); border: 1px solid var(--border); }
[data-theme='dark'] .icon-fd { background: #17335f; color: #a8c6ff; }
[data-theme='dark'] .icon-housing { background: #241a5c; color: #bcb2ff; }
[data-theme='dark'] .icon-savings { background: #0d3a3e; color: #7fe0eb; }
[data-theme='dark'] .icon-personal { background: #0f2e57; color: #9cc6ff; }
[data-theme='dark'] .icon-gold { background: #453400; color: #ffd76b; }

.product-card h3 {
  margin: 0 0 0.2rem;
  font-size: 0.95rem;
  font-weight: 700;
}
.product-card p {
  margin: 0;
  font-size: 0.8rem;
  color: var(--muted);
}
.soon-tag {
  display: inline-block;
  margin-top: 0.3rem;
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--muted);
}
</style>
