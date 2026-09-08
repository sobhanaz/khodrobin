<script setup lang="ts">
const { data: stats } = useStats()
const f = useFormat()

/**
 * Persian names for the slugs the index reports, not a list of marketplaces.
 *
 * The number of adapters has changed twice already, and a footer that names
 * four sources beside an index carrying five is a claim the product does not
 * keep. So the *set* comes from the live stats and this map only translates
 * it; an adapter that lands before its name does shows its slug rather than
 * disappearing from the footer.
 */
const SOURCE_FA: Record<string, string> = {
  divar: 'دیوار',
  bama: 'باما',
  hamrah: 'همراه‌مکانیک',
  khodro45: 'خودرو۴۵',
  sheypoor: 'شیپور',
}

const sources = computed(() => stats.value?.stats.sources ?? [])
const sourceNames = computed(() => sources.value.map(s => SOURCE_FA[s] ?? s))

/**
 * The status column is one register, all the way down: Persian labels, Persian
 * figures. It used to read «5,414 offers / 3,126 cars», Latin words set left
 * inside a right-aligned Persian block, which made the one part of the footer
 * carrying live numbers look like a log line that had leaked into the page.
 */
const status = computed(() => {
  const s = stats.value?.stats
  if (!s) return null
  return [
    { label: 'آگهی نمایه‌شده', value: f.fa(s.indexed) },
    { label: 'خودروی یکتا', value: f.fa(s.specs) },
    { label: 'بیش از یک منبع', value: f.fa(s.multi_source_specs) },
    { label: 'منبع', value: f.fa(s.sources.length) },
  ]
})

// Read, not remembered. A hardcoded year is right for twelve months and then
// quietly wrong, which is the same failure as a hardcoded source count.
const year = new Intl.DateTimeFormat('fa-IR-u-ca-persian', { year: 'numeric' }).format(new Date())

const navLink = 'flex min-h-[44px] items-center text-[.84rem] text-ink-2 transition hover:text-ink ' +
  'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
</script>

<template>
  <footer class="mt-20 border-t border-white/[.07]">
    <div class="mx-auto grid max-w-[1080px] gap-8 px-5 py-12 sm:grid-cols-2 lg:grid-cols-4">
      <div>
        <div class="mb-2 flex items-center gap-2 font-black">
          <BrandMark :size="22" class="shrink-0 text-ink" />خودروبین
        </div>
        <p class="max-w-[34ch] text-[.82rem] leading-7 text-ink-2">
          آگهی‌های خودروی دست‌دوم را از سایت‌های آگهی ایرانی کنار هم می‌گذاریم تا قیمت واقعی بازار را ببینی.
        </p>
      </div>

      <nav aria-labelledby="ft-pages">
        <h2 id="ft-pages" class="mb-2 text-[.8rem] font-bold text-ink-2">صفحه‌ها</h2>
        <NuxtLink to="/" :class="navLink">جست‌وجو</NuxtLink>
        <NuxtLink to="/about" :class="navLink">درباره‌ی خودروبین</NuxtLink>
        <NuxtLink to="/faq" :class="navLink">پرسش‌های پرتکرار</NuxtLink>
        <NuxtLink to="/contact" :class="navLink">تماس با ما</NuxtLink>
      </nav>

      <div>
        <h2 class="mb-3 text-[.8rem] font-bold text-ink-2">منبع‌ها</h2>
        <p v-if="sourceNames.length" class="text-[.82rem] leading-7 text-ink-2">
          {{ sourceNames.join(' · ') }}
        </p>
        <p class="mt-2 max-w-[34ch] text-[.76rem] leading-6 text-ink-2">
          آگهی‌ها متعلق به همان سایت‌هاست و هر کارت به آگهی اصلی لینک می‌دهد.
        </p>
      </div>

      <div>
        <h2 class="mb-3 text-[.8rem] font-bold text-ink-2">وضعیت</h2>
        <dl v-if="status" class="max-w-[22ch] text-[.8rem] leading-7">
          <div v-for="row in status" :key="row.label" class="flex items-baseline justify-between gap-3">
            <dt class="text-ink-2">{{ row.label }}</dt>
            <dd class="font-bold tabular-nums text-ink">{{ row.value }}</dd>
          </div>
        </dl>
        <a
          href="https://github.com/sobhanaz/khodrobin"
          target="_blank"
          rel="noopener noreferrer"
          class="mt-3 flex min-h-[44px] items-center font-mono text-[.78rem] text-ink-2 transition hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          dir="ltr"
        >github ↗</a>
      </div>
    </div>

    <div class="border-t border-white/[.07]">
      <div class="mx-auto flex max-w-[1080px] flex-wrap items-center justify-between gap-2 px-5 py-4 text-[.75rem] text-ink-2">
        <span>© {{ year }} خودروبین. ساخته‌شده برای چالش AI Product Engineer ترب.</span>
        <span>قیمت‌ها هر ۳ ساعت به‌روز می‌شوند</span>
      </div>
    </div>
  </footer>
</template>
