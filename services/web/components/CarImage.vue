<script setup lang="ts">
defineProps<{ src: string | null, alt: string }>()

/**
 * Images are hotlinked from each marketplace's own CDN. Any of them can 404 or
 * be slow, so a broken image must degrade to a calm placeholder rather than a
 * torn icon. `no-referrer` keeps our URLs out of their logs.
 *
 * The intrinsic 4:3 dimensions match the wrapper's aspect ratio. The wrapper
 * already reserves the space, so this is not what prevents layout shift; it is
 * what keeps the box the right shape in the moment before the stylesheet has
 * applied, and on any reader that ignores it.
 */
const broken = ref(false)
</script>

<template>
  <div class="relative aspect-4/3 w-28 shrink-0 overflow-hidden rounded-xl bg-surface-2 sm:w-36">
    <img
      v-if="src && !broken"
      :src="src"
      :alt="alt"
      width="400"
      height="300"
      loading="lazy"
      decoding="async"
      referrerpolicy="no-referrer"
      class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
      @error="broken = true"
    >
    <div v-else class="flex size-full items-center justify-center text-ink-3" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" class="size-7 opacity-60">
        <path d="M4 16l2-6h12l2 6M4 16h16M4 16v2m16-2v2M7 16v-1m10 1v-1" stroke-linecap="round" />
      </svg>
    </div>
  </div>
</template>
