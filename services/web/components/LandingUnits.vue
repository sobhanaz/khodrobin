<script setup lang="ts">
import type { Spec, Offer } from '~/types'
import { MARKETPLACES } from './LandingSources.vue'

/**
 * Rials against tomans, argued on one real car rather than in the abstract.
 *
 * Every feed calls the field `price`. Some publish rials and some publish
 * tomans, so trusting the field name overstates a rial car tenfold. That is
 * not a claim you settle by reading documentation; it is settled by putting
 * the same car from two sources side by side, which is what this section does,
 * live, with whatever car the index is corroborating best right now.
 *
 * Which source publishes which unit comes from the one table in
 * LandingSources. The first version of this component asked «is it Divar?»,
 * which quietly became wrong the day a second rial source shipped and would
 * have printed «تومان» over a rial figure on the page whose entire argument is
 * that units cannot be guessed from names.
 */
const props = defineProps<{ spec: Spec }>()
const f = useFormat()

const cheapestIn = (unit: 'rial' | 'toman') =>
  props.spec.offers
    .filter(o => MARKETPLACES[o.source]?.unit === unit)
    .reduce<Offer | null>((best, o) => (!best || o.price < best.price ? o : best), null)

const rial = computed(() => cheapestIn('rial'))
const toman = computed(() => cheapestIn('toman'))

/**
 * Offer prices arrive already reconciled to tomans, so the figure the source
 * itself published is recovered by undoing exactly the conversion the
 * normalizer applied: rials are tomans times ten. It is that site's own number
 * in that site's own unit, not a second measurement.
 */
const asPublished = computed(() => (rial.value ? rial.value.price * 10 : 0))

const gap = computed(() =>
  rial.value && toman.value ? Math.abs(rial.value.price - toman.value.price) : 0)

const cheaper = computed(() => {
  if (!rial.value || !toman.value || gap.value === 0) return null
  return rial.value.price < toman.value.price ? rial.value : toman.value
})
</script>

<template>
  <section v-if="rial && toman" class="mt-[var(--space-section)]" aria-labelledby="units-h">
    <h2 id="units-h" class="text-title">
      همه‌ی منبع‌ها اسم این فیلد را <span class="text-accent">price</span> گذاشته‌اند
    </h2>
    <p class="mt-2 max-w-[58ch] text-lead text-ink-2">
      و همه یک واحد را نمی‌گویند. همان خودروی بالا، همان لحظه، آن‌طور که هر منبع منتشرش می‌کند.
    </p>

    <div class="mt-6 overflow-hidden rounded-2xl border border-white/[.07] bg-surface">
      <div class="grid sm:grid-cols-2">
        <div
          v-for="side in [
            { fa: rial.source_fa, n: asPublished, unit: 'ریال', url: rial.url },
            { fa: toman.source_fa, n: toman.price, unit: 'تومان', url: toman.url },
          ]"
          :key="side.fa"
          class="border-b border-white/[.07] p-5 last:border-b-0 sm:border-b-0 sm:border-e sm:last:border-e-0"
        >
          <div class="text-cap text-ink-3">آن‌چه {{ side.fa }} منتشر می‌کند</div>
          <div class="mt-1.5" dir="ltr">
            <span class="font-mono text-[clamp(1.15rem,3.4vw,1.6rem)] font-bold tabular-nums text-ink">{{ f.money(side.n) }}</span>
          </div>
          <div class="mt-1 text-[.8rem] font-bold text-ink-2">{{ side.unit }}</div>
          <a
            :href="side.url"
            target="_blank"
            rel="noopener noreferrer"
            class="mt-2 inline-flex min-h-[44px] items-center text-cap text-ink-3 underline decoration-white/20 underline-offset-4 transition hover:text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >همین آگهی در {{ side.fa }}</a>
        </div>
      </div>

      <div class="border-t border-white/[.07] bg-warn/[.07] px-5 py-4">
        <p class="text-[.9rem] leading-8 text-ink-2">
          <b class="font-bold text-warn">اگر اسم فیلد را باور کنیم</b>، آگهی {{ rial.source_fa }} ده برابر
          {{ toman.source_fa }} است و از هر فهرست قیمتی بیرون می‌افتد.
        </p>
      </div>

      <div class="border-t border-white/[.07] bg-good/[.07] px-5 py-4">
        <p class="text-[.9rem] leading-8 text-ink-2">
          <b class="font-bold text-good">بعد از یکسان‌سازی</b>، هر دو به تومان می‌آیند:
          <span class="mono-nums text-ink">{{ f.money(rial.price) }}</span>
          و
          <span class="mono-nums text-ink">{{ f.money(toman.price) }}</span>.
          <template v-if="cheaper">
            یعنی آگهی {{ cheaper.source_fa }} در واقع {{ f.short(gap) }} تومان ارزان‌تر است.
          </template>
          <template v-else>
            یعنی هر دو دقیقاً یک قیمت‌اند.
          </template>
        </p>
      </div>
    </div>

    <p class="mt-4 max-w-[64ch] text-cap leading-7 text-ink-3">
      سال‌ها هم همین‌طورند: شمسی و میلادی داخل یک فید قاطی می‌شوند، بسته به این‌که خودرو داخلی باشد یا وارداتی.
      هیچ‌کدام از این‌ها را از روی نام فیلد تشخیص نمی‌دهیم؛ با قیمت‌کردن یک خودروی واقعی روی هر منبع تشخیص می‌دهیم.
    </p>
  </section>
</template>
