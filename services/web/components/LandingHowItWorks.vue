<script setup lang="ts">
const props = defineProps<{
  /**
   * Share of captured rows that were the same ad crawled again, from the live
   * index. Null when stats have not loaded, and the sentence then says the
   * thing without the figure rather than shipping a remembered «۴۰٪» that the
   * index may no longer agree with.
   */
  collapsedPct: number | null
  /**
   * How many sources the index actually holds, also live.
   *
   * This step used to say «چهار» in prose. There are five adapters written and
   * the number has changed twice; a page that states it from memory is wrong
   * the next time someone adds one. Null drops the figure, keeps the sentence.
   */
  sourceCount: number | null
}>()

const f = useFormat()

const steps = computed(() => [
  {
    tag: 'crawl',
    title: 'جمع‌آوری',
    body: props.sourceCount
      ? `هر سه ساعت ${f.fa(props.sourceCount)} بازار آگهی را می‌خوانیم، بدون مرورگر خودکار؛ همه‌شان داده‌ی ساختاریافته منتشر می‌کنند.`
      : 'هر سه ساعت بازارهای آگهی را می‌خوانیم، بدون مرورگر خودکار؛ همه‌شان داده‌ی ساختاریافته منتشر می‌کنند.',
    // The soft block is the honest headline of this step: a 200 that means no.
    note: 'دیوار وقتی تصمیم بگیرد تند می‌خوانی، JSON-LD را از پاسخ برمی‌دارد و همچنان ۲۰۰ برمی‌گرداند. پس خروجی را می‌سنجیم، نه کد وضعیت را.',
  },
  {
    tag: 'normalize',
    title: 'یکسان‌سازی',
    body: 'دیوار قیمت را به ریال می‌دهد و بقیه به تومان؛ سال شمسی و میلادی هم داخل یک فید قاطی می‌شوند.',
    note: props.collapsedPct === null
      ? 'بخش بزرگی از ردیف‌های خوانده‌شده همان آگهی است که دوباره خوانده شده. قبل از آنکه چیزی شمرده شود حذفشان می‌کنیم.'
      : `${f.fa(props.collapsedPct)}٪ از ردیف‌های خوانده‌شده همان آگهی است که دوباره خوانده شده. قبل از آنکه چیزی شمرده شود حذفشان می‌کنیم، وگرنه میانه‌ی قیمت را جابه‌جا می‌کنند.`,
  },
  {
    tag: 'rank',
    title: 'رتبه‌بندی',
    body: 'مرتب‌سازی در Go انجام می‌شود و قطعی است: همان جست‌وجو، همان ترتیب، بدون مدل.',
    note: 'هر کارت امتیازش را با تفکیک عامل‌ها برمی‌گرداند، پس «پشت صحنه» چیزی را حدس نمی‌زند؛ همان عددهایی را نشان می‌دهد که رتبه با آن‌ها ساخته شده.',
  },
  {
    tag: 'explain',
    title: 'توضیح',
    body: 'یک مدل زبانی در دو جمله می‌گوید چرا این گزینه بهتر است.',
    note: 'بعد نگهبان همان دو جمله را روی پنج محور با داده‌ی ورودی می‌سنجد و هر چیزی را که نتواند راستی‌آزمایی کند رد می‌کند.',
  },
])
</script>

<template>
  <section class="mt-[var(--space-section)]" aria-labelledby="how-h">
    <h2 id="how-h" class="text-title">از آگهی پراکنده تا یک قیمت قابل دفاع</h2>
    <p class="mt-2 max-w-[58ch] text-lead text-ink-2">
      چهار مرحله. هیچ‌کدام جادو نیست و هر کدام یک تصمیم دارد که می‌شود بازرسی‌اش کرد.
    </p>

    <ol class="mt-6 grid gap-3 sm:grid-cols-2">
      <li v-for="(s, i) in steps" :key="s.tag" class="glass rounded-2xl p-5">
        <div class="flex items-center gap-2.5">
          <span class="flex size-7 shrink-0 items-center justify-center rounded-full border border-accent/30 bg-accent/[.12] text-[.78rem] font-bold text-accent">
            {{ f.year(i + 1) }}
          </span>
          <span class="font-bold text-ink">{{ s.title }}</span>
          <span class="font-mono text-[.68rem] tracking-wider text-ink-3" dir="ltr">{{ s.tag }}</span>
        </div>
        <p class="mt-3 text-[.9rem] leading-8 text-ink-2">{{ s.body }}</p>
        <p class="mt-2 border-t border-white/[.07] pt-2 text-cap text-ink-3">{{ s.note }}</p>
      </li>
    </ol>
  </section>
</template>
