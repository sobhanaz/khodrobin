<script setup lang="ts">
const { query, mode, data, pending, failed, run } = useSearch()
const { data: stats } = useStats()
const f = useFormat()

/*
 * Server-rendered first paint: the page arrives with results, not a spinner.
 *
 * The handler has to hand the result back so it lands in the payload. It used
 * to return nothing, which meant the browser hydrated against an empty ref,
 * Vue threw the server-rendered cards away as a mismatch, and every visit paid
 * for the same search twice.
 */
const { data: firstPaint } = await useAsyncData('initial', async () => {
  await run()
  return data.value
})
if (!data.value) data.value = firstPaint.value

/**
 * The dedup rate, computed from the live index rather than remembered.
 *
 * It was 40% on the day it was written down; the next crawl decides what it is
 * today. Null keeps the sentence honest until the figure actually arrives.
 */
const collapsedPct = computed(() => {
  const s = stats.value?.stats
  if (!s?.listings_captured || !s.duplicates_collapsed) return null
  return Math.round((s.duplicates_collapsed / s.listings_captured) * 100)
})

function scrollToResults() {
  const el = document.getElementById('results')
  if (!el) return
  // Someone who turned motion off asked not to be flung down a page; they get
  // the same destination without the ride.
  const still = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  el.scrollIntoView({ behavior: still ? 'auto' : 'smooth', block: 'start' })
}

/**
 * The marketing sections now sit between the search box and the results, so a
 * submit that only swapped the results out would look like nothing happened.
 *
 * Scrolled twice on purpose: the first call starts moving immediately, and the
 * second re-aims once the new list has changed the page height underneath the
 * animation — a smooth scroll keeps the pixel it was given, so without it the
 * page lands hundreds of pixels short of the results it was sent to.
 */
async function search(q?: string) {
  scrollToResults()
  await run(q)
  scrollToResults()
}
</script>

<template>
  <main class="mx-auto max-w-[1080px] px-5">
    <section class="pb-8 pt-14 text-center sm:pt-16">
      <span class="inline-flex items-center gap-2 rounded-full border border-accent/30 bg-accent/[.12] px-3 py-1 font-mono text-[.7rem] tracking-wider text-accent">
        crawl → normalize → rank → explain
      </span>
      <h1 class="mx-auto mt-5 text-[clamp(2rem,5.5vw,3.4rem)] font-black leading-[1.22] tracking-tight">
        یک خودرو، <em class="not-italic text-accent">همه‌ی آگهی‌ها</em>،<br>یک قیمت واقعی.
      </h1>
      <p class="mx-auto mt-3 max-w-[52ch] text-[1.02rem] text-ink-2">
        آگهی‌های دیوار، باما، همراه‌مکانیک و خودرو۴۵ را کنار هم می‌گذاریم، واحدها و مدل‌ها را یکسان می‌کنیم، و میانه‌ی واقعی بازار را نشان می‌دهیم.
      </p>

      <div class="mx-auto mt-8 max-w-[720px]">
        <SearchBox v-model="query" @submit="search()" />

        <div class="mt-3.5 flex flex-wrap justify-center gap-2">
          <button
            v-for="ex in EXAMPLE_QUERIES"
            :key="ex"
            type="button"
            class="glass min-h-[44px] rounded-full px-4 py-2 text-[.8rem] text-ink-2 transition hover:-translate-y-0.5 hover:text-ink"
            @click="search(ex)"
          >{{ ex }}</button>
        </div>

        <IntentChips :intent="data?.result.intent ?? null" />
      </div>
    </section>

    <LandingProof :stats="stats" />

    <LandingHowItWorks :collapsed-pct="collapsedPct" />

    <LandingSources :sources="stats?.stats.sources ?? null" />

    <LandingGuard />

    <section class="mt-20" aria-labelledby="cta-h">
      <h2 id="cta-h" class="sr-only">خبرنامه</h2>
      <LandingSubscribe />
    </section>

    <!-- The search results the hero box drives. Anchored, and offset for the
         sticky header so the first card is not hidden under it. -->
    <section id="results" class="mt-20 scroll-mt-[78px]" aria-labelledby="results-h">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="results-h" class="text-[clamp(1.4rem,3.6vw,2rem)] font-black tracking-tight">نتیجه‌ها</h2>
          <p class="mt-1 text-[.88rem] text-ink-2">
            <template v-if="data?.result.total">
              <b class="text-ink">{{ f.fa(data.result.total) }}</b> خودرو پیدا شد
            </template>
            <template v-else-if="!pending">چیزی پیدا نشد</template>
          </p>
        </div>
        <p v-if="data" class="mono-nums text-[.7rem] text-ink-3">
          parse {{ data.timing.parse_ms.toFixed(2) }}ms · rank {{ data.timing.rank_ms.toFixed(2) }}ms · {{ data.result.mode }}
        </p>
      </div>

      <ModeTabs v-model="mode" />

      <div class="mt-3.5 grid gap-3">
        <template v-if="pending && !data">
          <div v-for="i in 3" :key="i" class="skeleton h-[104px] rounded-2xl" />
        </template>
        <p v-else-if="failed" class="py-14 text-center text-ink-3">ارتباط با سرور برقرار نشد.</p>
        <p v-else-if="!data?.result.specs.length" class="py-14 text-center text-ink-3">
          برای این جست‌وجو آگهی‌ای نداریم. یکی از نمونه‌های بالا را امتحان کن.
        </p>
        <SpecCard
          v-for="(s, i) in data?.result.specs ?? []"
          :key="s.key"
          :spec="s"
          :index="i"
        />
      </div>
    </section>

    <!-- Same component, footer register: someone who scrolled the whole page
         and read the results should not have to scroll back up to subscribe. -->
    <section class="mt-16" aria-label="عضویت در خبرنامه">
      <LandingSubscribe compact />
    </section>
  </main>
</template>

<style scoped>
.skeleton {
  background: linear-gradient(90deg, var(--color-surface), var(--color-surface-2), var(--color-surface));
  background-size: 200% 100%;
  animation: sheen 1.3s linear infinite;
}
@keyframes sheen { to { background-position: -200% 0 } }
</style>
