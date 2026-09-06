<script setup lang="ts">
import type { Spec } from '~/types'

const props = defineProps<{ spec: Spec, delay: number }>()
const f = useFormat()

/**
 * Every offer is a tick at its real position between the cheapest and dearest
 * price; the median is the taller marker. This is a distribution plot, not a
 * progress bar — a spec with one price collapses to a single centred tick
 * instead of pretending to have a range.
 */
const span = computed(() => props.spec.max_price - props.spec.min_price)
const at = (p: number) => (span.value > 0 ? ((p - props.spec.min_price) / span.value) * 100 : 50)
</script>

<template>
  <div class="px-5 pb-4">
    <div class="relative h-[26px]" dir="ltr">
      <span
        class="axis absolute inset-x-0 top-3 h-[3px] rounded-full opacity-50"
        :style="{ animationDelay: `${delay + 120}ms` }"
      />
      <span
        v-for="(o, n) in spec.offers"
        :key="n"
        class="tick absolute top-[7px] h-[13px] w-[3px] -translate-x-1/2 rounded-full bg-ink-2"
        :style="{ left: `${at(o.price)}%`, animationDelay: `${delay + 160 + n * 40}ms` }"
      />
      <span
        class="tick absolute top-[2px] h-[23px] w-[3px] -translate-x-1/2 rounded-full bg-ink shadow-[0_0_0_3px_var(--color-bg-2)]"
        :style="{ left: `${at(spec.median_price)}%`, animationDelay: `${delay + 300}ms` }"
      />
    </div>
    <div class="mt-0.5 flex justify-between font-mono text-[.68rem] text-ink-3" dir="ltr">
      <span class="flex flex-col">
        <span class="font-sans text-[.64rem] opacity-75">ارزان‌ترین</span>
        {{ f.money(spec.min_price) }}
      </span>
      <span class="flex flex-col items-end">
        <span class="font-sans text-[.64rem] opacity-75">گران‌ترین</span>
        {{ f.money(spec.max_price) }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.axis {
  background: linear-gradient(90deg, var(--color-good), var(--color-warn), var(--color-accent));
  transform-origin: left center;
  animation: grow .6s var(--ease-out-quint) both;
}
@keyframes grow { from { transform: scaleX(0) } to { transform: scaleX(1) } }
.tick { opacity: 0; animation: tick .4s var(--ease-out-quint) forwards; }
@keyframes tick { to { opacity: .85 } }
</style>
