<script setup lang="ts">
import type { Intent } from '~/types'

const props = defineProps<{ intent: Intent | null }>()
const f = useFormat()

/**
 * What the machine understood, shown back to the user.
 *
 * The point is not decoration: it makes the parse falsifiable. If it reads the
 * query wrongly the user sees exactly where, instead of wondering why the
 * results look off.
 */
const chips = computed(() => {
  const i = props.intent
  if (!i) return []
  const out: { k: string, v: string }[] = []
  if (i.brand_fa) out.push({ k: 'برند', v: i.brand_fa })
  if (i.model_fa) out.push({ k: 'مدل', v: i.model_fa })
  if (i.year_min) out.push({ k: 'از سال', v: f.year(i.year_min) })
  if (i.year_max && i.year_max !== i.year_min) out.push({ k: 'تا سال', v: f.year(i.year_max) })
  if (i.price_max) out.push({ k: 'حداکثر', v: f.short(i.price_max) })
  if (i.price_min) out.push({ k: 'حداقل', v: f.short(i.price_min) })
  if (i.gearbox) out.push({ k: 'گیربکس', v: i.gearbox === 'at' ? 'اتوماتیک' : 'دنده‌ای' })
  return out
})
</script>

<template>
  <div class="mx-auto mt-5 flex min-h-8 max-w-[820px] flex-wrap items-center justify-center gap-2" aria-live="polite">
    <TransitionGroup name="chip">
      <span
        v-for="(c, n) in chips"
        :key="c.k"
        :style="{ transitionDelay: `${n * 45}ms` }"
        class="inline-flex items-center gap-1.5 rounded-full border border-good/30 bg-good/[.12] px-3 py-1 text-[.78rem] text-good"
      >
        <span class="font-mono text-[.66rem] text-ink-3">{{ c.k }}</span>
        <span>{{ c.v }}</span>
      </span>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.chip-enter-active, .chip-leave-active { transition: all .38s var(--ease-out-quint); }
.chip-enter-from, .chip-leave-to { opacity: 0; transform: translateY(6px) scale(.94); }
</style>
