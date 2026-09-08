<script setup lang="ts">
const props = defineProps<{
  /** Source slugs present in the live index; null until stats load. */
  sources: string[] | null
}>()

// Each line is why that source needs no browser automation — the reason this
// crawler is four HTTP clients instead of four headless Chromes.
const MARKETPLACES = [
  { slug: 'divar', fa: 'دیوار', how: 'آرایه‌ی schema.org/Car را در خود صفحه رندر می‌کند.', unit: 'قیمت به ریال' },
  { slug: 'bama', fa: 'باما', how: 'یک API عمومی JSON برای جست‌وجو دارد.', unit: 'قیمت به تومان' },
  { slug: 'hamrah', fa: 'همراه‌مکانیک', how: 'Next.js است؛ داده داخل __NEXT_DATA__ نشسته.', unit: 'قیمت به تومان' },
  { slug: 'khodro45', fa: 'خودرو۴۵', how: 'یک اندپوینت Django REST است.', unit: 'قیمت به تومان' },
  { slug: 'sheypoor', fa: 'شیپور', how: 'یک CollectionPage از schema.org در خود صفحه رندر می‌کند.', unit: 'قیمت به ریال' },
]

/**
 * Only claim the sources the index actually has.
 *
 * A card for a marketplace that is currently blocking us would be a promise
 * the search results cannot keep, so the list follows the live stats and falls
 * back to all four only while stats are still loading.
 */
const shown = computed(() =>
  props.sources?.length
    ? MARKETPLACES.filter(m => props.sources!.includes(m.slug))
    : MARKETPLACES)
</script>

<template>
  <section class="mt-20" aria-labelledby="src-h">
    <h2 id="src-h" class="text-[clamp(1.4rem,3.6vw,2rem)] font-black tracking-tight">
      چه چیزی را می‌خوانیم
    </h2>
    <p class="mt-2 max-w-[58ch] text-[.95rem] leading-8 text-ink-2">
      آگهی‌ها مال همین سایت‌هاست و هر کارت به آگهی اصلی لینک می‌دهد. ما فقط کنار هم می‌گذاریمشان.
    </p>

    <ul class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <li v-for="m in shown" :key="m.slug" class="glass rounded-2xl p-5">
        <div class="flex items-baseline justify-between gap-2">
          <span class="font-bold text-ink">{{ m.fa }}</span>
          <span class="font-mono text-[.68rem] text-ink-3" dir="ltr">{{ m.slug }}</span>
        </div>
        <p class="mt-2.5 text-[.85rem] leading-7 text-ink-2">{{ m.how }}</p>
        <p class="mt-2 text-[.76rem] text-ink-3">{{ m.unit }}</p>
      </li>
    </ul>

    <p class="mt-4 text-[.82rem] leading-7 text-ink-3">
      هیچ‌کدام به مرورگر خودکار نیاز ندارند؛ یک کلاینت HTTP ساده کافی است — برای همین کل خزنده روی یک سرور کوچک می‌چرخد.
    </p>
  </section>
</template>
