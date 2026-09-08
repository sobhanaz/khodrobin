<script lang="ts">
/**
 * Which marketplace publishes what, and in which unit.
 *
 * This table is the single place the front end states a per-source fact, and
 * LandingUnits reads the unit from here rather than assuming «anything that is
 * not Divar quotes tomans». That assumption was wrong the moment a second rial
 * source shipped, and the section it powers is the one arguing that unit facts
 * cannot be guessed. Units follow services/crawler/normalize.py, which settled
 * them by pricing the same car on each source rather than by reading a label.
 */
export const MARKETPLACES: Record<string, { fa: string, how: string, unit: 'rial' | 'toman' }> = {
  divar: { fa: 'دیوار', how: 'آرایه‌ی schema.org/Car را در خود صفحه رندر می‌کند.', unit: 'rial' },
  bama: { fa: 'باما', how: 'یک API عمومی JSON برای جست‌وجو دارد.', unit: 'toman' },
  hamrah: { fa: 'همراه‌مکانیک', how: 'Next.js است؛ داده داخل __NEXT_DATA__ نشسته.', unit: 'toman' },
  khodro45: { fa: 'خودرو۴۵', how: 'یک اندپوینت Django REST است.', unit: 'toman' },
  sheypoor: { fa: 'شیپور', how: 'یک CollectionPage از schema.org در خود صفحه رندر می‌کند.', unit: 'rial' },
}

export const unitFa = (unit: 'rial' | 'toman') => (unit === 'rial' ? 'ریال' : 'تومان')
</script>

<script setup lang="ts">
const props = defineProps<{
  /** Source slugs present in the live index; null while stats are unavailable. */
  sources: string[] | null
}>()

/**
 * Only the sources the live index actually has.
 *
 * This used to fall back to the full hardcoded list whenever stats were null,
 * which is not «while loading» but permanently, whenever the fetch fails. It
 * rendered a «شیپور» card explaining how that source is crawled for a source
 * with nothing in the index. A section that cannot be sourced hides itself,
 * the same way the live-numbers band already does.
 */
const shown = computed(() =>
  (props.sources ?? [])
    .filter(slug => slug in MARKETPLACES)
    .map(slug => ({ slug, ...MARKETPLACES[slug], unit_fa: unitFa(MARKETPLACES[slug].unit) })))
</script>

<template>
  <section v-if="shown.length" class="mt-[var(--space-section)]" aria-labelledby="src-h">
    <h2 id="src-h" class="text-title">چه چیزی را می‌خوانیم</h2>
    <p class="mt-2 max-w-[58ch] text-lead text-ink-2">
      آگهی‌ها مال همین سایت‌هاست و هر کارت به آگهی اصلی لینک می‌دهد. ما فقط کنار هم می‌گذاریمشان.
      این فهرست از خود فهرستِ زنده می‌آید، نه از یک جدول نوشته‌شده.
    </p>

    <!-- A ledger, not a card row: sources that differ only in how their data
         is fetched read better as rows you can scan down one column. -->
    <ul class="mt-6 divide-y divide-white/[.07] overflow-hidden rounded-2xl border border-white/[.07] bg-surface">
      <li
        v-for="m in shown"
        :key="m.slug"
        class="grid gap-x-4 gap-y-1 p-4 sm:grid-cols-[11rem_minmax(0,1fr)_5rem] sm:items-baseline sm:p-5"
      >
        <span class="flex items-baseline gap-2">
          <span class="font-bold text-ink">{{ m.fa }}</span>
          <span class="font-mono text-[.68rem] text-ink-3" dir="ltr">{{ m.slug }}</span>
        </span>
        <span class="text-[.88rem] leading-7 text-ink-2">{{ m.how }}</span>
        <span class="text-cap text-ink-3 sm:text-left">قیمت به {{ m.unit_fa }}</span>
      </li>
    </ul>

    <p class="mt-4 max-w-[64ch] text-cap leading-7 text-ink-3">
      هیچ‌کدام به مرورگر خودکار نیاز ندارند؛ یک کلاینت HTTP ساده کافی است، و برای همین کل خزنده
      روی یک سرور کوچک می‌چرخد.
    </p>
  </section>
</template>
