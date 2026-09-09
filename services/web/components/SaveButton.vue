<script setup lang="ts">
import type { Spec } from '~/types'

const props = defineProps<{ spec: Spec, name: string }>()
const { isSaved, save, unsave } = useSaved()

const on = computed(() => isSaved(props.spec.key))

/**
 * The baseline we freeze, or nothing.
 *
 * Below three offers the median is the mean of two asking prices nobody is
 * actually paying, and the index says so with median_reliable. Freezing that
 * as the baseline would give /saved a number to subtract from and a percentage
 * to quote, and both would be fiction dressed as a measurement. null travels
 * instead, and the saved page says it has no baseline rather than printing a
 * confident ۰٪.
 *
 * median_reliable rides on the API payload but not on the Spec type this run,
 * so it is read through a narrowing — types.ts is owned by another change in
 * flight. Same trick as SimilarCars.
 */
const baseline = computed(() => {
  const s = props.spec as Spec & { median_reliable?: boolean }
  return s.median_reliable === false ? null : s.median_price
})

function toggle() {
  if (on.value) unsave(props.spec.key)
  else save({ key: props.spec.key, name: props.name, median: baseline.value })
}
</script>

<template>
  <!--
    The visible word is the action and the accessible name is the same action
    plus the car, so the label a voice-control user says out loud is actually
    on the button. Naming the car matters on a results page: fifteen identical
    «ذخیره» buttons in a row tell a screen reader nothing about which car.

    No network call, so no pending state exists to design — the toggle is the
    write, and it either happened or the browser refused to store at all.
  -->
  <button
    type="button"
    :aria-pressed="on"
    :aria-label="on ? `حذف ${name} از ذخیره‌ها` : `ذخیره‌ی ${name}`"
    class="inline-flex min-h-[44px] items-center gap-1.5 rounded-full border px-3.5 text-[.78rem] transition
           focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
    :class="on
      ? 'border-accent/35 bg-accent/[.12] text-accent'
      : 'border-white/[.07] bg-surface-2 text-ink-2 hover:border-white/[.12] hover:text-ink'"
    @click="toggle"
  >
    <svg
      viewBox="0 0 24 24"
      class="size-[15px] shrink-0"
      :fill="on ? 'currentColor' : 'none'"
      stroke="currentColor"
      stroke-width="1.5"
      aria-hidden="true"
    >
      <path d="M12 3.7l2.6 5.3 5.8.85-4.2 4.1 1 5.8L12 17l-5.2 2.75 1-5.8-4.2-4.1 5.8-.85z" stroke-linejoin="round" />
    </svg>
    {{ on ? 'حذف' : 'ذخیره' }}
  </button>
</template>
