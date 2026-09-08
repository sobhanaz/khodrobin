<script setup lang="ts">
const { data: stats } = useStats()
const f = useFormat()
const year = 1405
</script>

<template>
  <footer class="mt-20 border-t border-white/[.07]">
    <div class="mx-auto grid max-w-[1080px] gap-8 px-5 py-12 sm:grid-cols-2 lg:grid-cols-4">
      <div>
        <div class="mb-2 flex items-center gap-2 font-black">
          <BrandMark :size="22" class="shrink-0 text-ink" />خودروبین
        </div>
        <p class="text-[.82rem] leading-7 text-ink-3">
          آگهی‌های خودروی دست‌دوم را از چهار منبع کنار هم می‌گذاریم تا قیمت واقعی بازار را ببینی.
        </p>
      </div>

      <div>
        <div class="mb-3 text-[.8rem] font-bold text-ink-2">صفحه‌ها</div>
        <NuxtLink to="/" class="block py-1 text-[.82rem] text-ink-3 hover:text-ink">جست‌وجو</NuxtLink>
        <NuxtLink to="/about" class="block py-1 text-[.82rem] text-ink-3 hover:text-ink">درباره‌ی ما</NuxtLink>
        <NuxtLink to="/faq" class="block py-1 text-[.82rem] text-ink-3 hover:text-ink">پرسش‌های پرتکرار</NuxtLink>
        <NuxtLink to="/contact" class="block py-1 text-[.82rem] text-ink-3 hover:text-ink">تماس با ما</NuxtLink>
      </div>

      <div>
        <div class="mb-3 text-[.8rem] font-bold text-ink-2">منبع‌ها</div>
        <p class="text-[.8rem] leading-7 text-ink-3">دیوار · باما · همراه‌مکانیک · خودرو۴۵</p>
        <p class="mt-2 text-[.75rem] leading-6 text-ink-3 opacity-75">
          آگهی‌ها متعلق به همان سایت‌هاست و هر کارت به آگهی اصلی لینک می‌دهد.
        </p>
      </div>

      <div>
        <div class="mb-3 text-[.8rem] font-bold text-ink-2">وضعیت</div>
        <p v-if="stats" class="font-mono text-[.76rem] leading-7 text-ink-3" dir="ltr">
          {{ f.money(stats.stats.indexed) }} offers<br>
          {{ f.money(stats.stats.specs) }} cars<br>
          {{ f.money(stats.stats.multi_source_specs) }} multi-source
        </p>
        <a href="https://github.com/sobhanaz/khodrobin" target="_blank" rel="noopener noreferrer"
           class="mt-2 inline-block font-mono text-[.76rem] text-ink-3 hover:text-ink-2">github ↗</a>
      </div>
    </div>

    <div class="border-t border-white/[.07]">
      <div class="mx-auto flex max-w-[1080px] flex-wrap items-center justify-between gap-2 px-5 py-4 text-[.75rem] text-ink-3">
        <span>© {{ year }} خودروبین — ساخته‌شده برای چالش AI Product Engineer ترب</span>
        <span class="opacity-70">قیمت‌ها هر ۳ ساعت به‌روز می‌شوند</span>
      </div>
    </div>
  </footer>
</template>
