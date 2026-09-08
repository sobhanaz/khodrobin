<script setup lang="ts">
import type { StatsResponse } from '~/types'

const props = defineProps<{ stats: StatsResponse | null }>()
const f = useFormat()

/**
 * Every figure here is read live from /api/v1/stats.
 *
 * A zero on this band is never a fact about the market — it is a fact about a
 * request that failed, or an index built before the field existed. So a
 * missing figure drops its own tile and an empty band removes itself: a proof
 * band reading «۰ آگهی» disproves the product it exists to prove.
 */
const tiles = computed(() => {
  const s = props.stats?.stats
  if (!s) return []
  return [
    { n: s.listings_captured, label: 'آگهی خوانده‌شده', note: 'ردیف خام، پیش از حذف تکراری‌ها' },
    { n: s.duplicates_collapsed, label: 'تکراری حذف‌شده', note: 'یک آگهی که دوباره خوانده شده' },
    { n: s.specs, label: 'خودروی یکتا', note: 'برند، مدل، تیپ، گیربکس، سال' },
    { n: s.sources.length, label: 'منبع', note: 'همه از راه HTTP ساده' },
  ].filter(t => t.n > 0)
})
</script>

<template>
  <section v-if="tiles.length" class="mt-4" aria-labelledby="proof-h">
    <h2 id="proof-h" class="sr-only">وضعیت زنده‌ی فهرست</h2>

    <div class="glass rounded-2xl p-4 sm:p-5">
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div v-for="t in tiles" :key="t.label" class="rounded-xl bg-white/[.03] p-3.5 sm:p-4">
          <div class="mono-nums text-[clamp(1.15rem,3.6vw,1.5rem)] font-bold leading-tight text-ink">
            {{ f.fa(t.n) }}
          </div>
          <div class="mt-1 text-[.8rem] font-bold text-ink-2">{{ t.label }}</div>
          <div class="mt-0.5 text-[.72rem] leading-6 text-ink-3">{{ t.note }}</div>
        </div>
      </div>

      <p class="mt-3.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[.76rem] text-ink-3">
        <span class="inline-flex size-1.5 shrink-0 rounded-full bg-good" aria-hidden="true" />
        <span>این اعداد همین حالا از فهرست زنده خوانده شدند، نه از یک اسلاید.</span>
        <a
          href="/api/v1/stats"
          target="_blank"
          rel="noopener noreferrer"
          class="font-mono text-ink-2 underline decoration-white/20 underline-offset-4 hover:text-ink"
          dir="ltr"
        >/api/v1/stats ↗</a>
      </p>
    </div>
  </section>
</template>
