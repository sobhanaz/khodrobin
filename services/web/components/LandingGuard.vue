<script setup lang="ts">
// Five axes because the model went wrong in five different ways, and three of
// these exist only because a real answer got past the checks that came before.
const AXES = [
  { en: 'numbers', fa: 'عدد', catches: 'قیمتی یا کارکردی که در ورودی نبوده.' },
  { en: 'percentages', fa: 'درصد', catches: 'فاصله‌ای تا میانه که هیچ آگهی‌ای پشتش نیست.' },
  { en: 'topics', fa: 'موضوع', catches: 'موضوعی که ورودی اصلاً نداشته: گارانتی، رنگ، آپشن.' },
  { en: 'sources', fa: 'منبع', catches: 'نسبت‌دادن آگهی به سایتی که اینجا آگهی‌ای از آن نیست.' },
  { en: 'coherence', fa: 'انسجام', catches: 'جمله‌ای که تک‌تک عددهایش درست است و در کل بی‌معنی.' },
]
</script>

<template>
  <section class="mt-[var(--space-section)]" aria-labelledby="guard-h">
    <div class="glass rounded-3xl p-6 sm:p-8">
      <span class="inline-flex items-center rounded-full border border-good/30 bg-good/[.12] px-3 py-1 font-mono text-[.7rem] tracking-wider text-good">
        guard
      </span>
      <h2 id="guard-h" class="mt-4 text-title">
        هر توضیح، قبل از آنکه ببینی‌اش، رد شده است یا از پنج آزمون گذشته
      </h2>
      <p class="mt-3 max-w-[62ch] text-lead text-ink-2">
        نکته‌ی جالب لایه‌ی هوش مصنوعی، پرامپت نیست. هر توضیحی که مدل می‌نویسد دوباره روی پنج محور
        مستقل با همان داده‌ی ورودی سنجیده می‌شود، چون مدل به پنج شکل متفاوت خطا می‌کند.
      </p>

      <ul class="mt-6 grid gap-2">
        <li
          v-for="a in AXES"
          :key="a.en"
          class="flex flex-col gap-1 rounded-xl bg-white/[.03] p-3.5 sm:flex-row sm:items-baseline sm:gap-4"
        >
          <span class="flex shrink-0 items-baseline gap-2 sm:w-40">
            <span class="font-bold text-ink">{{ a.fa }}</span>
            <span class="font-mono text-[.68rem] text-ink-3" dir="ltr">{{ a.en }}</span>
          </span>
          <span class="text-[.88rem] leading-7 text-ink-2">{{ a.catches }}</span>
        </li>
      </ul>

      <!-- The fifth axis is the one worth telling: four factual checks passed a
           sentence that was true and meaningless, and only a check for sense
           could catch it. -->
      <figure class="mt-6 rounded-2xl border-s-2 border-warn/60 bg-warn/[.07] p-5">
        <blockquote class="text-[1rem] font-bold leading-9 text-ink">
          «کارکرد صفر کیلومتر را از دست می‌دهی.»
        </blockquote>
        <figcaption class="mt-3 text-[.88rem] leading-8 text-ink-2">
          این جمله روی یک کارت واقعی نشست. تک‌تک عددهایش درست بود، پس هر چهار محور واقعیت‌سنج
          تأییدش کردند، و باز هم مزخرف است: ارزان‌ترین گزینه هم‌زمان کم‌کارکردترین بود و
          معامله‌ای در کار نبود که نامش را ببرد. ۱۰٪ از توضیح‌های زنده همین کار را می‌کردند.
          محور پنجم برای همین اضافه شد.
        </figcaption>
      </figure>

      <p class="mt-5 text-[.92rem] leading-8 text-ink-2">
        <b class="font-bold text-ink">پرامپت یک درخواست است؛ نگهبان یک تضمین.</b>
        وقتی نگهبان جلوی یک توضیح را می‌گیرد، کارت همین را می‌گوید و نشان می‌دهد چه چیزی رد شده،
        بعد به جمله‌ای برمی‌گردد که از خود داده ساخته شده: همیشه در دسترس، همیشه درست.
      </p>
    </div>
  </section>
</template>
