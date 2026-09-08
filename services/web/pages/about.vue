<script setup lang="ts">
useSeoMeta({
  title: 'درباره‌ی خودروبین',
  description: 'چرا خودروی دست‌دوم سخت‌ترین نسخه‌ی مسئله‌ی ترب است، خودروبین چطور حلش می‌کند، و چه چیزهایی را عمداً انجام نمی‌دهد.',
})

const { data: stats } = useStats()
const f = useFormat()

/**
 * Every figure on this page is the live index or is absent.
 *
 * The prose used to say «چهار سایت» in two places. There are five adapters
 * now, there were three before that, and a sentence that has to be edited when
 * the crawler changes is a sentence that will be wrong for a while first.
 */
const s = computed(() => stats.value?.stats ?? null)

const pipeline = [
  {
    title: 'جمع‌آوری',
    body: 'آگهی‌های عمومی هر منبع را هر سه ساعت می‌خوانیم. هیچ‌کدام به مرورگر خودکار نیاز ندارند: هر سایت داده‌ی ساختاریافته منتشر می‌کند، چون خودش هم می‌خواهد موتورهای جست‌وجو بخوانندش، و یک کلاینت HTTP ساده کافی است.',
  },
  {
    title: 'یکسان‌سازی',
    body: 'بعضی منبع‌ها قیمت را به ریال می‌دهند و بعضی به تومان. همه‌شان سال شمسی و میلادی را داخل یک فید قاطی می‌کنند، بسته به اینکه خودرو داخلی باشد یا وارداتی. واحدها با مقایسه‌ی یک خودروی مشخص بین منبع‌ها سنجیده شده‌اند، نه با اعتماد به اسم فیلدها.',
  },
  {
    title: 'گروه‌بندی',
    body: 'آگهی‌هایی که برند، مدل، تیپ، گیربکس، سال و بازه‌ی کارکرد یکسان دارند زیر یک کارت جمع می‌شوند، با توزیع قیمت واقعی زیرشان. این همان کاری است که کارت ترب با یک کالا می‌کند.',
  },
  {
    title: 'رتبه‌بندی',
    body: 'با یک امتیاز وزنیِ قطعی که در Go نوشته شده و تست واحد دارد، نه با یک مدل. هر کارت وزن هر عامل را در «پشت صحنه» نشان می‌دهد، چون رتبه‌ای که نتوانی بازش کنی، رتبه‌ای است که باید به آن ایمان بیاوری.',
  },
  {
    title: 'توضیح',
    body: 'یک مدل زبانی در دو جمله می‌گوید چرا این گزینه بهتر است. بعد یک نگهبان، متن را روی پنج محور با داده‌ی ورودی می‌سنجد و اگر رد شود، جمله‌ای که از خود داده ساخته شده جایش می‌نشیند.',
  },
]

const built = [
  { name: 'crawler', lang: 'Python', body: 'خواندن منبع‌ها، یکسان‌سازی واحدها، پرچم تناقض، ساخت نمایه.' },
  { name: 'api', lang: 'Go', body: 'جست‌وجو، تجزیه‌ی پرسش فارسی، رتبه‌بندی قطعی، کش و متریک.' },
  { name: 'ai', lang: 'FastAPI', body: 'تشخیص نیت، نوشتن توضیح، نگهبان پنج‌محوره و هارنس ارزیابی.' },
  { name: 'auth', lang: 'Go', body: 'حساب، نشست، جست‌وجوی ذخیره‌شده و هشدار قیمت.' },
  { name: 'web', lang: 'Nuxt', body: 'همین رابط، رندرشده در سرور، فارسی و راست‌به‌چپ.' },
]

const principles = [
  {
    t: 'عدد اشتباه را درست نمی‌کنیم، نشانش می‌دهیم',
    b: 'آگهی‌ای که با خودش نخواند پرچم می‌خورد و همان‌طور می‌ماند. عددی که ما «اصلاح» کنیم، دروغی است با ظاهر بهتر.',
  },
  {
    t: 'میانه‌ای که پشتش نایستیم اعلام نمی‌شود',
    b: 'زیر سه آگهی، میانه فقط میانگین دو قیمت پیشنهادی است که هیچ‌کس آن را نمی‌فروشد. کارت به‌جای عدد، همین را می‌گوید.',
  },
  {
    t: 'مدل فقط جایی که قاعده کم می‌آورد',
    b: 'بیشتر عنوان‌های فارسی خودرو قالب‌دار هستند و با قاعده تجزیه می‌شوند. مدل برای جایی است که قاعده واقعاً از پسش برنمی‌آید، و همان‌جا هم اندازه گرفته می‌شود.',
  },
  {
    t: 'جست‌وجو پشت ورود نمی‌رود',
    b: 'حساب کاربری فقط برای ذخیره‌ی جست‌وجو و هشدار قیمت است. بدون آن، جست‌وجو، رتبه‌بندی، توضیح و صفحه‌ی هر خودرو کامل کار می‌کنند.',
  },
]
</script>

<template>
  <main class="mx-auto max-w-[760px] px-5 py-16">
    <h1 class="text-title">قیمت واقعی یک خودرو، نه قیمت یک آگهی.</h1>

    <p class="mt-5 max-w-[45ch] text-lead text-ink-2">
      یک پژو ۲۰۷ هم‌زمان در چند سایت آگهی می‌شود، با قیمت‌هایی که تا چند برابر با هم فرق دارند.
      خودروبین آن آگهی‌های پراکنده را به یک خودرو تبدیل می‌کند و می‌گوید بازار واقعاً چقدر می‌ارزد.
    </p>

    <!-- The state of the index as a sentence rather than a row of tiles: the
         landing page already shows these as figures, and repeating the same
         component here would make the two pages one page with two headings. -->
    <p v-if="s" class="mt-6 max-w-[45ch] border-s-2 border-accent/40 ps-4 text-[.92rem] leading-8 text-ink-2">
      همین حالا <b class="font-bold text-ink">{{ f.fa(s.indexed) }}</b> آگهی از
      <b class="font-bold text-ink">{{ f.fa(s.sources.length) }}</b> منبع در نمایه است، روی
      <b class="font-bold text-ink">{{ f.fa(s.specs) }}</b> خودروی یکتا، که
      <b class="font-bold text-ink">{{ f.fa(s.multi_source_specs) }}</b> تای آن‌ها بیش از یک منبع دارند.
    </p>

    <section class="mt-[var(--space-section)]">
      <h2 class="text-head">چرا خودروی دست‌دوم</h2>
      <div class="mt-4 max-w-[45ch] space-y-4 text-[.95rem] leading-8 text-ink-2">
        <p>
          سخت‌ترین مسئله‌ای که ترب برای خودش نوشته این است: تشخیص اینکه یک کالا در دو فروشگاه،
          همان کالاست. برای کالای فروشگاهی معمولاً یک کد کالا این کار را می‌کند و مسئله به یک
          اتصال ساده تبدیل می‌شود.
        </p>
        <p>
          خودروی دست‌دوم کد کالا ندارد. هویتش باید از روی یک عنوان آزاد فارسی استنباط شود:
          «پژو ۲۰۶ تیپ۵ مدل۹۰ فول» قبل از هر مقایسه‌ای باید به برند، مدل، تیپ و سال تبدیل شود،
          آن هم وقتی هر سایت واژگان خودش را دارد. همین یک تفاوت، مسئله را از یک اتصال به یک
          استنباط با نرخ خطای واقعی تبدیل می‌کند، و نرخ خطای واقعی یعنی چیزی که باید اندازه‌اش
          گرفت و نشانش داد.
        </p>
        <p>
          نقشه‌ی اول این بود که بفهمیم سه آگهی در سه سایت، یک خودروی فیزیکی هستند. قبل از
          ساختنش داده را سنجیدیم و این‌طور نبود: برخوردهای بین‌منبعی عملاً خودروهای متفاوتی با
          قیمت نزدیک بودند. پس واحد مقایسه «مشخصات» شد، نه یک خودروی مشخص. این عقب‌نشینی نیست؛
          ادعای قابل‌دفاع‌تری است: این مشخصات میانه‌ای دارد که چند آگهی از چند منبع پشتش است.
        </p>
      </div>
    </section>

    <section class="mt-[var(--space-section)]">
      <h2 class="text-head">مسیر یک آگهی تا کارت</h2>
      <!-- A rule with markers, not five cards. The landing page is built out of
           card grids; a page that is mostly prose should not borrow its chrome. -->
      <ol class="mt-6 max-w-[45ch] border-s border-white/[.1] ps-6">
        <li v-for="(p, i) in pipeline" :key="p.title" :class="i ? 'mt-7' : ''" class="relative">
          <span
            class="absolute -start-[1.78rem] top-1.5 size-2.5 rounded-full bg-accent ring-4 ring-bg"
            aria-hidden="true"
          />
          <h3 class="font-bold text-ink">{{ p.title }}</h3>
          <p class="mt-1.5 text-[.92rem] leading-8 text-ink-2">{{ p.body }}</p>
        </li>
      </ol>
    </section>

    <section class="mt-[var(--space-section)]">
      <h2 class="text-head">چه چیزی ساخته شد</h2>
      <p class="mt-3 max-w-[45ch] text-[.92rem] leading-8 text-ink-2">
        {{ f.fa(built.length) }} سرویس روی یک سرور، که همه‌شان از گیت‌هاب می‌روند بالا.
      </p>
      <dl class="mt-5 max-w-[45ch] divide-y divide-white/[.07] border-y border-white/[.07]">
        <div v-for="b in built" :key="b.name" class="flex flex-wrap items-baseline gap-x-4 gap-y-1 py-3.5">
          <dt class="flex w-[9.5rem] shrink-0 items-baseline gap-2">
            <span class="font-mono text-[.85rem] font-bold text-ink" dir="ltr">{{ b.name }}</span>
            <span class="font-mono text-[.7rem] text-ink-2" dir="ltr">{{ b.lang }}</span>
          </dt>
          <dd class="min-w-[16ch] flex-1 text-[.88rem] leading-7 text-ink-2">{{ b.body }}</dd>
        </div>
      </dl>
    </section>

    <section class="mt-[var(--space-section)]">
      <h2 class="text-head">چیزهایی که عمداً می‌کنیم یا نمی‌کنیم</h2>
      <div class="mt-5 grid gap-x-10 gap-y-7 sm:grid-cols-2">
        <div v-for="p in principles" :key="p.t">
          <h3 class="text-[.95rem] font-bold text-ink">{{ p.t }}</h3>
          <p class="mt-1.5 max-w-[42ch] text-[.88rem] leading-8 text-ink-2">{{ p.b }}</p>
        </div>
      </div>
    </section>

    <section class="mt-[var(--space-section)] border-t border-white/[.07] pt-8">
      <h2 class="text-head">سازنده</h2>
      <p class="mt-3 max-w-[45ch] text-[.95rem] leading-8 text-ink-2">
        خودروبین را سبحان عظیم‌زاده نوشته؛ عمق کارش Go، Vue/Nuxt و زیرساخت است.
        این پروژه برای چالش <span class="text-ink">AI Product Engineer</span> ترب ساخته شده،
        «ترب ___ رو بساز»، و کد، تصمیم‌ها و اعدادش عمومی است.
      </p>
      <div class="mt-5 flex flex-wrap gap-2">
        <a
          href="https://github.com/sobhanaz/khodrobin"
          target="_blank"
          rel="noopener noreferrer"
          class="flex min-h-[44px] items-center rounded-full border border-white/[.14] px-4 text-[.85rem] text-ink-2 transition hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >کد پروژه ↗</a>
        <NuxtLink
          to="/faq"
          class="flex min-h-[44px] items-center rounded-full border border-white/[.14] px-4 text-[.85rem] text-ink-2 transition hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >پرسش‌های پرتکرار</NuxtLink>
        <NuxtLink
          to="/contact"
          class="flex min-h-[44px] items-center rounded-full bg-accent px-5 text-[.85rem] font-bold text-white transition hover:brightness-110
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >تماس</NuxtLink>
      </div>
    </section>
  </main>
</template>
