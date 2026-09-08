<script setup lang="ts">
useSeoMeta({
  title: 'پرسش‌های پرتکرار | خودروبین',
  description: 'قیمت‌ها از کجا می‌آیند، هر چند وقت به‌روز می‌شوند، چرا دو سایت یک عدد نمی‌گویند، و میانه‌ی بازار کِی اعلام نمی‌شود.',
})

/**
 * Grouped, because nine unlabelled rows is a list to scroll rather than a page
 * to read. The groups are the three questions a buyer actually arrives with:
 * where the number came from, whether to believe it, and what we keep about
 * them.
 *
 * Note there is no list of marketplaces here. The number of sources has
 * changed twice; the footer reads it from the live index on every page, and a
 * sentence that repeats it is a sentence that goes stale on its own.
 */
const groups = [
  {
    title: 'داده از کجا می‌آید',
    items: [
      {
        q: 'قیمت‌ها را از کجا می‌آورید؟',
        a: 'از آگهی‌های عمومی چند سایت آگهی ایرانی. فهرست منبع‌ها در پای همین صفحه از نمایه‌ی زنده خوانده می‌شود، پس همیشه همان چیزی است که همین حالا در داده هست. ما آگهی نمی‌سازیم و قیمت تعیین نمی‌کنیم؛ فقط آنچه را که این سایت‌ها منتشر کرده‌اند کنار هم می‌گذاریم، و هر آگهی به صفحه‌ی اصلی خودش لینک دارد.',
      },
      {
        q: 'هر چند وقت به‌روز می‌شود؟',
        a: 'هر سه ساعت. روی هر آگهی نوشته‌ایم آخرین بار کِی آن را دیده‌ایم، چون قیمتی که چند ساعت قدیمی است باید قبل از تصمیم‌گیری معلوم باشد، نه بعدش.',
      },
      {
        q: 'چرا قیمت یک خودرو در دو سایت فرق دارد؟',
        a: 'چون هر عدد، قیمت پیشنهادی یک فروشنده است، نه قیمتی که معامله شده. دو خودرو با مشخصات یکسان هم می‌توانند از نظر رنگ، آپشن، بیمه و وضعیت بدنه فرق داشته باشند، و بعضی آگهی‌ها هفته‌هاست که مانده‌اند. واحد پول هم یکی نیست: بعضی سایت‌ها ریال می‌نویسند و بعضی تومان، که ما یکسانش می‌کنیم. برای همین به‌جای یک عدد، توزیع قیمت‌ها را نشان می‌دهیم.',
      },
      {
        q: 'چرا بعضی خودروها فقط یک منبع دارند؟',
        a: 'چون آن مشخصات دقیق فقط در یک سایت آگهی دارد. هر چه تعداد منبع بیشتر باشد میانه قابل‌اعتمادتر است، و رتبه‌بندی هم همین را به حساب می‌آورد.',
      },
    ],
  },
  {
    title: 'چقدر می‌شود به عدد اعتماد کرد',
    items: [
      {
        q: '«میانه‌ی بازار» یعنی چه، و چرا گاهی اعلام نمی‌شود؟',
        a: 'برای هر خودرو با مشخصات یکسان (برند، مدل، تیپ، گیربکس، سال و بازه‌ی کارکرد) قیمت وسط همه‌ی آگهی‌ها را حساب می‌کنیم. میانه از میانگین بهتر است چون یک آگهی با قیمت پرت جابه‌جایش نمی‌کند. اگر کمتر از سه آگهی باشد، میانه را به‌عنوان قیمت بازار اعلام نمی‌کنیم: با دو آگهی، میانه فقط میانگین دو قیمت پیشنهادی است که هیچ‌کس آن را نمی‌فروشد. کارت به‌جای عدد، همین را می‌گوید.',
      },
      {
        q: 'پرچم «تناقض» یا «قیمت پرت» روی یک آگهی یعنی چه؟',
        a: 'تناقض یعنی عددهای خود آگهی با هم نمی‌خوانند؛ مثل خودرویی که «صفر کیلومتر» ثبت شده و هم‌زمان «گلگیر تعویض» دارد. قیمت پرت یعنی عدد آگهی به‌طرز غیرعادی از بقیه‌ی هم‌گروه‌هایش دور است، که اغلب نشانه‌ی این است که اصلاً خودرو نیست (مثلاً فروش حواله). هیچ‌کدام را اصلاح نمی‌کنیم و فقط نشان می‌دهیم، ولی پرچم تزئینی هم نیست: آگهی پرچم‌دار ته فهرست همان خودرو می‌نشیند و توضیح خودکار هرگز روی آن نوشته نمی‌شود.',
      },
      {
        q: 'متن «چرا این؟» را چه کسی می‌نویسد؟',
        a: 'یک مدل زبانی که فقط اجازه دارد از عددهای همان کارت استفاده کند. بعد از نوشتن، متن روی پنج محور با داده‌ی ورودی سنجیده می‌شود: عددی که در داده نبوده، درصدی که هیچ آگهی پشتش نیست، موضوعی که اصلاً مطرح نبوده، اسم سایتی که اینجا آگهی ندارد، و جمله‌ای که راست است ولی بی‌معنا (مثل اینکه کم‌کارکردترین گزینه را به‌عنوان چیزی که «از دست می‌دهی» معرفی کند). اگر رد شود، جمله‌ای که مستقیم از خود داده ساخته شده جایش می‌آید و زیر توضیح نوشته‌ایم کدام حالت اتفاق افتاده.',
      },
    ],
  },
  {
    title: 'حساب، حریم خصوصی و تماس',
    items: [
      {
        q: 'حساب کاربری برای چیست؟',
        a: 'فقط برای ذخیره‌ی جست‌وجو و هشدار قیمت. جست‌وجو، رتبه‌بندی، توضیح و صفحه‌ی هر خودرو بدون حساب هم کامل کار می‌کنند و هیچ‌وقت پشت ورود نمی‌روند. حساب برای این است که وقتی قیمت خودروی موردنظرت تکان خورد، خبرت کنیم.',
      },
      {
        q: 'شماره تماس فروشنده را دارید؟',
        a: 'نه، و جمعش هم نمی‌کنیم. برای تماس باید به آگهی اصلی در همان سایت بروی. جمع‌آوری ما عمومی و بدون ورود است؛ نه شماره‌ای برداشته می‌شود، نه حساب کاربری‌ای ساخته می‌شود.',
      },
    ],
  },
]

const flat = groups.flatMap(g => g.items)

// FAQPage structured data: this is the page shape search engines render as a
// rich result, and the answers are already written for a person.
useHead({
  script: [{
    type: 'application/ld+json',
    innerHTML: JSON.stringify({
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      mainEntity: flat.map(item => ({
        '@type': 'Question',
        name: item.q,
        acceptedAnswer: { '@type': 'Answer', text: item.a },
      })),
    }),
  }],
})
</script>

<template>
  <main class="mx-auto max-w-[680px] px-5 py-16">
    <h1 class="text-title">پرسش‌های پرتکرار</h1>
    <p class="mt-4 max-w-[44ch] text-lead text-ink-2">
      هر چیزی که قبل از اعتماد به یک عدد روی این سایت، خوب است بدانی.
    </p>

    <!-- Native <details>. It is keyboard-operable, announced as expanded or
         collapsed, findable by the browser's own in-page search even while
         closed, and needs no state to go wrong. -->
    <div class="mt-12 grid gap-12">
      <section v-for="(g, gi) in groups" :key="g.title" :aria-labelledby="`g-${gi}`">
        <h2 :id="`g-${gi}`" class="text-head text-ink-2">{{ g.title }}</h2>
        <div class="mt-4 divide-y divide-white/[.07] border-y border-white/[.07]">
          <details v-for="item in g.items" :key="item.q" class="group">
            <summary
              class="flex min-h-[56px] cursor-pointer list-none items-center justify-between gap-4 py-4
                     focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            >
              <span class="text-[.98rem] font-bold leading-8 text-ink">{{ item.q }}</span>
              <span
                class="grid size-7 shrink-0 place-items-center rounded-full border border-white/[.14] text-ink-2
                       transition-transform duration-300 group-open:rotate-45 group-open:border-accent/45 group-open:text-accent"
                aria-hidden="true"
              >+</span>
            </summary>
            <p class="max-w-[45ch] pb-5 pe-1 text-[.92rem] leading-9 text-ink-2">{{ item.a }}</p>
          </details>
        </div>
      </section>
    </div>

    <p class="mt-12 text-[.9rem] leading-8 text-ink-2">
      جوابت اینجا نبود؟
      <NuxtLink
        to="/contact"
        class="text-accent hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      >برایمان بنویس</NuxtLink>.
    </p>
  </main>
</template>

<style scoped>
/* Safari still paints its own disclosure triangle unless this is said in the
   vendor form; list-none alone does not remove it. */
summary::-webkit-details-marker { display: none; }
</style>
