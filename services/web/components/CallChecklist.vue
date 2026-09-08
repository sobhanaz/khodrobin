<script setup lang="ts">
import type { Offer } from '~/types'

const props = defineProps<{ offers: Offer[] }>()
const f = useFormat()

/**
 * The flags this product computes ARE the questions a buyer should ask on the
 * phone. A warning that stays a warning dies on the page; turned into a
 * question, it gets answered by the seller. Wording mirrors what each code
 * actually means in plausibility.py / build_index.py, not the flag text.
 */
const QUESTIONS: Record<string, string> = {
  zero_km_with_bodywork: 'آگهی می‌گوید صفر کیلومتر ولی بدنه رنگ/تعویض دارد؛ دلیلش را بپرس.',
  not_a_car_yet: 'این حواله/پیش‌فروش است، نه خودروی آماده؛ زمان تحویل و سند را بپرس.',
  instalment_price: 'قیمت اقساطی است؛ قیمت نقدی نهایی را بپرس.',
  zero_km_old_car: 'برای خودرویی با این سن، کارکرد صفر باورکردنی نیست؛ عدد واقعی کیلومتر را بپرس.',
  mileage_vs_age: 'کارکرد نسبت به سن خودرو غیرعادی زیاد است؛ درستی کیلومتر و سابقه‌ی کار را بپرس.',
  mileage_sentinel: 'عدد کارکرد واقعی به نظر نمی‌رسد؛ کیلومتر دقیق را بپرس.',
  price_outlier: 'قیمت با بقیه‌ی آگهی‌های همین خودرو فاصله‌ی زیادی دارد؛ دلیل این اختلاف را بپرس.',
  off_market_sale: 'آگهی فروش دولتی یا سازمانی است؛ مبلغ واقعی پرداختی و شرایط تحویل را بپرس.',
  no_price: 'قیمتی اعلام نشده؛ پیش از هر چیز قیمت قطعی را بپرس.',
}

/**
 * Keyed by code so the same doubt across several offers is asked once, with a
 * count. A code this map does not know keeps its original flag message
 * verbatim — a flag added to the crawler later must surface here on its own,
 * not vanish until someone remembers this file exists.
 */
const items = computed(() => {
  const byCode = new Map<string, { text: string, count: number }>()
  for (const o of props.offers) {
    for (const fl of o.flags ?? []) {
      const cur = byCode.get(fl.code)
      if (cur) cur.count++
      else byCode.set(fl.code, { text: QUESTIONS[fl.code] ?? fl.message, count: 1 })
    }
  }
  return [...byCode.entries()].map(([code, v]) => ({ code, ...v }))
})
</script>

<template>
  <ul v-if="items.length" class="grid gap-2.5">
    <li v-for="it in items" :key="it.code" class="flex items-start gap-2.5">
      <span
        class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border border-warn/25 bg-warn/[.12] text-[.72rem] text-warn"
        aria-hidden="true"
      >؟</span>
      <span class="text-[.88rem] leading-7 text-ink-2">
        {{ it.text }}
        <span v-if="it.count > 1" class="text-[.74rem] text-ink-3">(در {{ f.fa(it.count) }} آگهی)</span>
      </span>
    </li>
  </ul>

  <!-- No flags is a finding, not a blank. Say what we checked and found. -->
  <p v-else class="text-[.88rem] leading-7 text-ink-2">
    <span class="text-good">✓</span>
    در آگهی‌های این خودرو تناقضی پیدا نکردیم؛ باز هم سند، رنگ بدنه و کارکرد را حضوری بررسی کن.
  </p>
</template>
