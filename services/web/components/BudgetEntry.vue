<script setup lang="ts">
import type { Spec } from '~/types'

/**
 * «چقدر بودجه داری؟» — the other way into the product.
 *
 * `specs` is a sample the host page has ALREADY fetched (the landing sample,
 * or the current budget's own results). The presets are read off it rather
 * than fetched, because a preset row is not worth a second 300KB response and
 * a number nobody in the index is asking is not worth offering.
 */
const props = defineProps<{ specs?: Spec[] | null }>()

const router = useRouter()
const f = useFormat()
const fieldId = useId()
const headingId = useId()

const MAX_MILLIONS = 100_000

const digits = ref('')
const amount = computed(() => Number(digits.value || '0'))
const valid = computed(() => amount.value >= 1 && amount.value <= MAX_MILLIONS)

/** What the field shows. The number is Persian; there is no reason to type it twice. */
const shown = computed(() => (digits.value ? f.year(amount.value) : ''))

function onInput(e: Event) {
  const el = e.target as HTMLInputElement
  digits.value = toEnDigits(el.value).replace(/\D/g, '').slice(0, 6)
  // Vue only patches the DOM when the bound value CHANGES, so a rejected
  // keystroke — a letter, a second dot — left the character sitting in the
  // field while the model had already dropped it. Writing the value back is
  // what makes the field show what the component actually holds.
  // Only when the input actually rejected something. Assigning .value resets
  // the selection, so doing it unconditionally sent the caret to the end on
  // every keystroke and made editing the middle of a number impossible.
  if (el.value !== shown.value) el.value = shown.value
}

function submit() {
  if (!valid.value) return
  router.push(`/budget/${amount.value}`)
}

/*
 * Presets are round numbers people say out loud, filtered by what the index
 * holds right now: the ladder is the vocabulary, the live sample picks which
 * rungs are worth offering. Computing the numbers instead would produce
 * «۲٬۰۵۰ میلیون», which is a percentile, not a budget anybody has in mind.
 */
// Every rung is exact under f.short's one decimal («۱٫۲ میلیارد», never a
// rounded «۱٫۳» standing in for 1250), which is why the chips can print it raw.
const LADDER = [200, 300, 400, 500, 650, 800, 1000, 1200, 1500, 2000, 2500, 3000, 4000, 5000, 7000, 10000]

const presets = computed<number[]>(() => {
  const list = props.specs ?? []
  if (!list.length) return []
  // From the cheapest asking price in the sample up to its middle car: below
  // the first, nothing is for sale; above the second, this is no longer the
  // market most people are shopping in.
  const lo = Math.min(...list.map(s => s.min_price)) / 1e6
  const medians = list.map(s => s.median_price).sort((a, b) => a - b)
  const hi = medians[Math.floor(medians.length / 2)]! / 1e6
  const rungs = LADDER.filter(v => v >= lo && v <= hi)
  // One rung is not a choice; the row hides rather than pretending to offer one.
  if (rungs.length < 2) return []
  const step = (rungs.length - 1) / 3
  return [...new Set([0, 1, 2, 3].map(i => rungs[Math.round(i * step)]!))]
})
</script>

<template>
  <section class="rounded-2xl border border-white/[.07] bg-surface p-5 sm:p-6" :aria-labelledby="headingId">
    <h2 :id="headingId" class="text-head">با چه بودجه‌ای دنبال خودرویی؟</h2>
    <p class="mt-1 text-cap text-ink-2">
      عدد را به میلیون تومان بنویس. ما می‌گوییم با آن چه خودروهایی در بازار هست.
    </p>

    <form class="mt-4 flex flex-col gap-2 sm:flex-row sm:items-end" @submit.prevent="submit">
      <div class="flex-1">
        <label :for="fieldId" class="mb-1 block text-[.8rem] text-ink-2">بودجه (میلیون تومان)</label>
        <div
          class="flex items-center gap-2 rounded-xl border border-white/[.12] bg-bg-2 px-3
                 duration-250 [transition-property:border-color,box-shadow]
                 focus-within:border-focus/55 focus-within:shadow-[0_0_0_4px_rgba(107,116,230,.16)]"
        >
          <input
            :id="fieldId"
            :value="shown"
            type="text"
            inputmode="numeric"
            autocomplete="off"
            enterkeyhint="go"
            placeholder="۸۵۰"
            class="min-h-[48px] w-full min-w-0 bg-transparent text-base text-ink outline-none placeholder:text-ink-3"
            @input="onInput"
          >
          <span class="shrink-0 text-[.82rem] text-ink-3">میلیون</span>
        </div>
      </div>

      <button
        type="submit"
        class="min-h-[48px] shrink-0 rounded-xl bg-accent px-6 font-bold text-white transition
               active:scale-[.98] hover:brightness-110
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      >
        ببین چی می‌شود خرید
      </button>
    </form>

    <p class="mt-2 min-h-5 text-[.76rem] text-ink-3" aria-live="polite">
      <template v-if="!digits">مثلاً «۸۵۰» یعنی ۸۵۰ میلیون تومان.</template>
      <template v-else-if="valid">
        یعنی <span class="mono-nums">{{ f.money(amount * 1_000_000) }}</span> تومان.
      </template>
      <template v-else>عدد را به میلیون بنویس؛ «۸۵۰» یعنی ۸۵۰ میلیون تومان.</template>
    </p>

    <!-- No sample, no presets. A hardcoded row of round numbers would be the
         one thing on this page not backed by the live index. -->
    <div v-if="presets.length" class="mt-3 flex flex-wrap gap-2">
      <NuxtLink
        v-for="p in presets"
        :key="p"
        :to="`/budget/${p}`"
        class="glass flex min-h-[44px] items-center rounded-full px-4 text-[.82rem] text-ink-2 transition
               hover:-translate-y-0.5 hover:text-ink
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      >{{ f.short(p * 1_000_000) }}</NuxtLink>
    </div>
  </section>
</template>
