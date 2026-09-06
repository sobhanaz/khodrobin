<script setup lang="ts">
import type { Spec } from '~/types'

const props = defineProps<{ spec: Spec }>()

const LABELS: Record<string, string> = {
  price: 'قیمت',
  mileage: 'کارکرد',
  age: 'سال ساخت',
  corroborate: 'تأیید چندمنبعی',
  integrity: 'سلامت داده',
}

/**
 * The ranking is a weighted sum and this draws its actual terms. It is not a
 * summary of the score — it *is* the score, so a user can disagree with the
 * ordering on evidence instead of on faith.
 */
const rows = computed(() =>
  Object.entries(props.spec.breakdown ?? {})
    .sort((a, b) => b[1] - a[1])
    .map(([k, v]) => ({ key: k, label: LABELS[k] ?? k, value: v })),
)
const max = computed(() => Math.max(...rows.value.map(r => r.value), 0.001))
</script>

<template>
  <div class="grid gap-2">
    <div
      v-for="(r, i) in rows"
      :key="r.key"
      class="grid grid-cols-[88px_1fr_46px] items-center gap-2.5 text-[.76rem]"
    >
      <span class="text-ink-3">{{ r.label }}</span>
      <span class="h-1.5 overflow-hidden rounded-full bg-surface-2">
        <span
          class="fl block h-full rounded-full bg-accent"
          :style="{ width: `${(r.value / max) * 100}%`, animationDelay: `${i * 60}ms` }"
        />
      </span>
      <span class="text-left font-mono text-[.72rem] text-ink-2" dir="ltr">{{ r.value.toFixed(3) }}</span>
    </div>
  </div>
</template>

<style scoped>
.fl { transform-origin: right center; animation: grow .6s var(--ease-out-quint) both; }
@keyframes grow { from { transform: scaleX(0) } to { transform: scaleX(1) } }
</style>
