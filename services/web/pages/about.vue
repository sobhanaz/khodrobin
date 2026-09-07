<script setup lang="ts">
useSeoMeta({
  title: 'درباره‌ی خودروبین',
  description: 'چرا خودروبین ساخته شد، چطور کار می‌کند، و چه چیزهایی را عمداً انجام نمی‌دهد.',
})
const { data: stats } = useStats()
const f = useFormat()

const pipeline = [
  { step: '۱', title: 'جمع‌آوری', body: 'آگهی‌های عمومی چهار سایت را هر سه ساعت می‌خوانیم. هیچ‌کدام نیاز به مرورگر خودکار ندارند؛ هر چهار سایت داده‌ی ساختاریافته منتشر می‌کنند.' },
  { step: '۲', title: 'یکسان‌سازی', body: 'دیوار قیمت را به ریال می‌دهد و بقیه به تومان. هر چهار سایت سال شمسی و میلادی را قاطی می‌کنند. همه را به یک شکل واحد در می‌آوریم.' },
  { step: '۳', title: 'گروه‌بندی', body: 'خودروهایی با برند، مدل، تیپ، گیربکس، سال و بازه‌ی کارکرد یکسان زیر یک کارت جمع می‌شوند — همان کاری که ترب با کالا می‌کند.' },
  { step: '۴', title: 'رتبه‌بندی', body: 'با یک امتیاز وزنیِ قابل‌بازرسی، نه با یک مدل. می‌توانی وزن هر عامل را در «پشت صحنه» ببینی.' },
  { step: '۵', title: 'توضیح', body: 'یک مدل زبانی در دو جمله می‌گوید چرا این گزینه بهتر است — و هر عددش دوباره با داده‌ی ورودی سنجیده می‌شود.' },
]

const principles = [
  { t: 'عدد اشتباه را درست نمی‌کنیم، نشان می‌دهیم', b: 'اگر آگهی‌ای با خودش نخواند، پرچم می‌خورد. عددی که ما «اصلاح» کنیم، دروغی است با ظاهر بهتر.' },
  { t: 'مدل فقط جایی که لازم است', b: 'از ۹۲ جست‌وجوی آزمون، ۹۰ تا را قواعد قطعی جواب می‌دهند. مدل برای دو موردی است که قاعده از پسش برنمی‌آید.' },
  { t: 'هر ادعا قابل بازرسی است', b: 'هر کارت باز می‌شود و منبع، تصمیم‌های یکسان‌سازی، وزن رتبه‌بندی و خود متن مدل را نشان می‌دهد.' },
  { t: 'جست‌وجو پشت ورود نمی‌رود', b: 'حساب کاربری فقط برای ذخیره‌ی جست‌وجو و هشدار قیمت است. بدون آن همه‌چیز کار می‌کند.' },
]
</script>

<template>
  <main class="mx-auto max-w-[820px] px-5 py-16">
    <span class="inline-flex items-center rounded-full border border-accent/30 bg-accent/[.12] px-3 py-1 font-mono text-[.7rem] tracking-wider text-accent">
      about
    </span>
    <h1 class="mt-5 text-[clamp(1.8rem,4.5vw,2.6rem)] font-black leading-tight tracking-tight">
      قیمت واقعی یک خودرو، نه قیمت یک آگهی.
    </h1>
    <p class="mt-4 text-[1.02rem] leading-9 text-ink-2">
      یک پژو ۲۰۷ مدل ۱۴۰۴ هم‌زمان در چهار سایت آگهی می‌شود، با قیمت‌هایی که تا ۵۰٪ با هم فرق دارند.
      برخلاف کالای فروشگاهی، خودروی دست‌دوم کد کالا ندارد؛ هویتش باید از روی یک عنوان آزاد فارسی
      استنباط شود. خودروبین دقیقاً همین کار را می‌کند: آگهی‌های پراکنده را به یک خودرو تبدیل می‌کند
      و می‌گوید بازار واقعاً چقدر می‌ارزد.
    </p>

    <div v-if="stats" class="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-4">
      <div v-for="s in [
        { n: f.money(stats.stats.listings), l: 'آگهی خوانده‌شده' },
        { n: f.money(stats.stats.specs), l: 'خودروی یکتا' },
        { n: f.money(stats.stats.multi_source_specs), l: 'چندمنبعی' },
        { n: f.fa(stats.stats.sources.length), l: 'منبع' },
      ]" :key="s.l" class="rounded-xl border border-white/[.07] bg-surface p-4">
        <div class="font-mono text-[1.2rem] font-bold" dir="ltr">{{ s.n }}</div>
        <div class="mt-1 text-[.75rem] text-ink-3">{{ s.l }}</div>
      </div>
    </div>

    <h2 class="mt-14 text-[1.3rem] font-black">چطور کار می‌کند</h2>
    <ol class="mt-5 grid gap-3">
      <li v-for="p in pipeline" :key="p.step"
          class="flex gap-4 rounded-2xl border border-white/[.07] bg-surface p-5">
        <span class="flex size-8 shrink-0 items-center justify-center rounded-full border border-accent/30 bg-accent/[.12] text-[.8rem] font-bold text-accent">
          {{ p.step }}
        </span>
        <div>
          <div class="font-bold">{{ p.title }}</div>
          <p class="mt-1 text-[.9rem] leading-8 text-ink-2">{{ p.body }}</p>
        </div>
      </li>
    </ol>

    <h2 class="mt-14 text-[1.3rem] font-black">چیزهایی که عمداً انجام می‌دهیم یا نمی‌دهیم</h2>
    <div class="mt-5 grid gap-3 sm:grid-cols-2">
      <div v-for="p in principles" :key="p.t" class="rounded-2xl border border-white/[.07] bg-surface p-5">
        <div class="font-bold text-ink">{{ p.t }}</div>
        <p class="mt-2 text-[.88rem] leading-8 text-ink-2">{{ p.b }}</p>
      </div>
    </div>

    <h2 class="mt-14 text-[1.3rem] font-black">سازنده</h2>
    <p class="mt-4 text-[.95rem] leading-9 text-ink-2">
      خودروبین را سبحان عظیم‌زاده ساخته — مهندس نرم‌افزار فول‌استک، با Go، Nuxt و زیرساخت.
      این پروژه برای چالش <span class="text-ink">AI Product Engineer</span> ترب نوشته شده است:
      «ترب ___ رو بساز».
      کل کد، تصمیم‌ها و اعدادش عمومی است.
    </p>
    <div class="mt-5 flex flex-wrap gap-2">
      <a href="https://github.com/sobhanaz/khodrobin" target="_blank" rel="noopener noreferrer"
         class="rounded-full border border-white/[.12] px-4 py-2 text-[.85rem] text-ink-2 hover:text-ink">
        کد پروژه ↗
      </a>
      <a href="https://github.com/sobhanaz" target="_blank" rel="noopener noreferrer"
         class="rounded-full border border-white/[.12] px-4 py-2 text-[.85rem] text-ink-2 hover:text-ink">
        گیت‌هاب سازنده ↗
      </a>
      <NuxtLink to="/contact" class="rounded-full bg-accent px-4 py-2 text-[.85rem] font-bold text-white hover:brightness-110">
        تماس
      </NuxtLink>
    </div>
  </main>
</template>
