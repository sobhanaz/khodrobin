<script setup lang="ts">
useSeoMeta({
  title: 'پرسش‌های پرتکرار — خودروبین',
  description: 'قیمت‌ها از کجا می‌آیند، هر چند وقت به‌روز می‌شوند، و میانه‌ی بازار یعنی چه.',
})

const faqs = [
  {
    q: 'قیمت‌ها از کجا می‌آید؟',
    a: 'از آگهی‌های عمومی چهار سایت: دیوار، باما، همراه‌مکانیک و خودرو۴۵. ما آگهی نمی‌سازیم و قیمت تعیین نمی‌کنیم؛ فقط آنچه را که این سایت‌ها منتشر کرده‌اند کنار هم می‌گذاریم. هر آگهی به صفحه‌ی اصلی خودش لینک دارد.',
  },
  {
    q: '«میانه‌ی بازار» یعنی چه؟',
    a: 'برای هر خودرو با مشخصات یکسان — برند، مدل، تیپ، گیربکس، سال و بازه‌ی کارکرد — قیمت وسط همه‌ی آگهی‌ها را حساب می‌کنیم. میانه از میانگین بهتر است چون یک آگهی با قیمت پرت، آن را جابه‌جا نمی‌کند. اگر کمتر از سه آگهی باشد، میانه را به‌عنوان قیمت بازار اعلام نمی‌کنیم؛ با دو آگهی، میانه فقط میانگین همان دو قیمت است و کسی آن را نمی‌فروشد.',
  },
  {
    q: 'هر چند وقت به‌روز می‌شود؟',
    a: 'هر سه ساعت. روی هر آگهی نوشته‌ایم آخرین بار کی آن را دیده‌ایم، چون قیمتی که چند ساعت قدیمی است باید قبل از تصمیم‌گیری معلوم باشد.',
  },
  {
    q: 'چرا یک آگهی «تناقض» خورده؟',
    a: 'وقتی عددهای خود آگهی با هم نمی‌خوانند. مثلاً پرایدی مدل ۱۳۸۵ که «صفر کیلومتر» ثبت شده و هم‌زمان «گلگیر تعویض» دارد. ما این‌ها را اصلاح نمی‌کنیم، فقط نشان می‌دهیم — عددی که ما «درست» کنیم، دروغی است با ظاهر بهتر.',
  },
  {
    q: 'متن «چرا این؟» را چه کسی می‌نویسد؟',
    a: 'یک مدل زبانی که فقط اجازه دارد از عددهای همان کارت استفاده کند. بعد از نوشتن، متن را دوباره بررسی می‌کنیم: اگر عددی گفته باشد که در داده نبوده، درصدی که هیچ آگهی پشتش نیست، یا اسم سایتی که اینجا آگهی ندارد، جوابش رد می‌شود و یک متن قالبی جایش می‌آید. زیر هر توضیح نوشته‌ایم کدام حالت اتفاق افتاده.',
  },
  {
    q: 'چرا بعضی خودروها فقط یک منبع دارند؟',
    a: 'چون آن مشخصات دقیق فقط در یک سایت آگهی دارد. هر چه تعداد منبع بیشتر باشد، قیمت میانه قابل‌اعتمادتر است، و در رتبه‌بندی هم همین را در نظر می‌گیریم.',
  },
  {
    q: 'حساب کاربری برای چیست؟',
    a: 'برای ذخیره‌ی جست‌وجو و هشدار قیمت. جست‌وجو بدون حساب هم کامل کار می‌کند و هیچ‌وقت پشت ورود قرار نمی‌گیرد؛ حساب فقط برای این است که وقتی قیمت خودروی موردنظرت تغییر کرد، خبرت کنیم.',
  },
  {
    q: 'شماره تماس فروشنده را دارید؟',
    a: 'نه، و جمع هم نمی‌کنیم. برای تماس باید به آگهی اصلی در همان سایت بروی.',
  },
]

// FAQPage structured data: this is the page shape search engines render as a
// rich result, and the answers are already written for a person.
useHead({
  script: [{
    type: 'application/ld+json',
    innerHTML: JSON.stringify({
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      mainEntity: faqs.map(f => ({
        '@type': 'Question',
        name: f.q,
        acceptedAnswer: { '@type': 'Answer', text: f.a },
      })),
    }),
  }],
})

const openIndex = ref<number | null>(0)
</script>

<template>
  <main class="mx-auto max-w-[760px] px-5 py-16">
    <h1 class="text-[clamp(1.7rem,4vw,2.4rem)] font-black tracking-tight">پرسش‌های پرتکرار</h1>
    <p class="mt-3 text-ink-2">هر چیزی که قبل از اعتماد به یک عدد، باید بدانی.</p>

    <div class="mt-10 grid gap-3">
      <div
        v-for="(f, i) in faqs"
        :key="i"
        class="overflow-hidden rounded-2xl border border-white/[.07] bg-surface"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between gap-4 p-5 text-right"
          :aria-expanded="openIndex === i"
          @click="openIndex = openIndex === i ? null : i"
        >
          <span class="font-bold">{{ f.q }}</span>
          <span
            class="shrink-0 text-ink-3 transition-transform duration-300"
            :class="openIndex === i ? 'rotate-45' : ''"
            aria-hidden="true"
          >+</span>
        </button>
        <div class="grid transition-[grid-template-rows] duration-[380ms]"
             :class="openIndex === i ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'">
          <div class="overflow-hidden">
            <p class="border-t border-white/[.07] bg-bg-2 px-5 py-4 text-[.92rem] leading-9 text-ink-2">
              {{ f.a }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <p class="mt-10 text-[.88rem] text-ink-3">
      جوابت اینجا نبود؟ <NuxtLink to="/contact" class="text-accent hover:underline">برایمان بنویس</NuxtLink>.
    </p>
  </main>
</template>
