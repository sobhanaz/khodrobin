<script setup lang="ts">
const props = defineProps<{
  /**
   * Share of captured rows that were the same ad crawled again, from the live
   * index. Null when stats have not loaded — the sentence then says the thing
   * without the figure rather than shipping a remembered «۴۰٪» that the index
   * may no longer agree with.
   */
  collapsedPct: number | null
}>()

const f = useFormat()

const steps = computed(() => [
  {
    tag: 'crawl',
    title: 'جمع‌آوری',
    body: 'هر سه ساعت بازارهای آگهی را می‌خوانیم، بدون مرورگر خودکار — همه‌شان داده‌ی ساختاریافته منتشر می‌کنند.',
    // The soft block is the honest headline of this step: a 200 that means no.
    note: 'دیوار وقتی تصمیم بگیرد تند می‌خوانی، JSON-LD را از پاسخ برمی‌دارد و همچنان ۲۰۰ برمی‌گرداند. پس خروجی را می‌سنجیم، نه کد وضعیت را.',
  },
  {
    tag: 'normalize',
    title: 'یکسان‌سازی',
    body: 'دیوار قیمت را به ریال می‌دهد و سه تای دیگر به تومان؛ هر چهار سایت سال شمسی و میلادی را داخل یک فید قاطی می‌کنند.',
    note: props.collapsedPct === null
      ? 'بخش بزرگی از ردیف‌های خوانده‌شده همان آگهی است که دوباره خوانده شده. قبل از آنکه چیزی شمرده شود حذفشان می‌کنیم.'
      : `${f.fa(props.collapsedPct)}٪ از ردیف‌های خوانده‌شده همان آگهی است که دوباره خوانده شده. قبل از آنکه چیزی شمرده شود حذفشان می‌کنیم — وگرنه میانه‌ی قیمت را جابه‌جا می‌کنند.`,
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
  <section class="mt-20" aria-labelledby="how-h">
    <h2 id="how-h" class="text-[clamp(1.4rem,3.6vw,2rem)] font-black tracking-tight">
      از آگهی پراکنده تا یک قیمت قابل دفاع
    </h2>
    <p class="mt-2 max-w-[58ch] text-[.95rem] leading-8 text-ink-2">
      چهار مرحله. هیچ‌کدام جادو نیست و هر کدام یک تصمیم دارد که می‌شود بازرسی‌اش کرد.
    </p>

    <ol class="mt-6 grid gap-3 sm:grid-cols-2">
      <li
        v-for="(s, i) in steps"
        :key="s.tag"
        class="glass rounded-2xl p-5"
      >
        <div class="flex items-center gap-2.5">
          <span class="flex size-7 shrink-0 items-center justify-center rounded-full border border-accent/30 bg-accent/[.12] text-[.78rem] font-bold text-accent">
            {{ f.year(i + 1) }}
          </span>
          <span class="font-bold text-ink">{{ s.title }}</span>
          <span class="font-mono text-[.68rem] tracking-wider text-ink-3" dir="ltr">{{ s.tag }}</span>
        </div>
        <p class="mt-3 text-[.9rem] leading-8 text-ink-2">{{ s.body }}</p>
        <p class="mt-2 border-t border-white/[.07] pt-2 text-[.82rem] leading-7 text-ink-3">{{ s.note }}</p>
      </li>
    </ol>
  </section>
</template>
