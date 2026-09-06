<script setup lang="ts"></script>

<template>
  <section class="flex min-h-[120px] flex-col items-center justify-center gap-5 bg-[#141E52] px-4 py-8 sm:flex-row sm:justify-between sm:gap-4 sm:px-6">
    <div class="flex items-center gap-3 text-left sm:gap-4">
      <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-primary/40 bg-white/5 text-primary sm:h-12 sm:w-12">
        <Icon name="shield" class="h-5 w-5 sm:h-6 sm:w-6" />
      </div>
      <div>
        <p class="text-sm font-bold text-white sm:text-base">Data Trust &amp; Sourcing</p>
        <p class="mt-0.5 max-w-md text-xs text-white/60 sm:text-sm">
          Rates are monitored and compiled from official Sri Lankan bank declarations. Refreshed systematically every 6 hours.
        </p>
      </div>
    </div>
    <NuxtLink
      to="/rates"
      class="w-full shrink-0 rounded-pill bg-white px-5 py-2.5 text-center text-sm font-semibold text-navy transition hover:bg-white/90 sm:w-auto"
    >
      Transparency Hub &rarr;
    </NuxtLink>
  </section>
</template>
