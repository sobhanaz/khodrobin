<script setup lang="ts">
import type { Offer } from '~/types'

defineProps<{ offer: Offer }>()
const f = useFormat()
</script>

<template>
  <div class="border-b border-dashed border-white/[.07] py-2.5 last:border-0">
    <div class="grid grid-cols-[auto_1fr_auto] items-center gap-3">
      <span class="rounded-full border border-good/30 bg-good/[.12] px-2.5 py-[3px] text-[.7rem] text-good">
        {{ offer.source_fa }}
      </span>
      <span class="min-w-0 truncate text-[.86rem] text-ink-2">
        <a :href="offer.url" target="_blank" rel="noopener noreferrer" class="hover:text-ink">
          {{ offer.title || 'بدون عنوان' }}
        </a>
        <span class="text-[.76rem] text-ink-3">
          · {{ offer.mileage_km == null ? 'کارکرد نامشخص' : `${f.money(offer.mileage_km)} km` }}
          <template v-if="offer.city"> · {{ offer.city }}</template>
        </span>
      </span>
      <span class="flex items-center gap-2">
        <span class="font-mono text-[.94rem] tabular-nums" dir="ltr">{{ f.money(offer.price) }}</span>
        <span
          class="rounded-md px-1.5 py-0.5 font-mono text-[.72rem]"
          dir="ltr"
          :class="offer.vs_median_pct < 0 ? 'bg-good/[.12] text-good' : 'bg-accent/[.12] text-accent'"
        >{{ offer.vs_median_pct > 0 ? '+' : '' }}{{ offer.vs_median_pct }}%</span>
      </span>
    </div>
    <div
      v-for="fl in offer.flags || []"
      :key="fl.code"
      class="mt-2 flex items-start gap-2 rounded-lg border border-warn/20 bg-warn/[.12] px-3 py-2 text-[.78rem] text-warn"
    >
      <span aria-hidden="true">⚑</span><span>{{ fl.message }}</span>
    </div>
  </div>
</template>
