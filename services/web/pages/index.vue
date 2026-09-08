<script setup lang="ts">
import type { SearchResponse, Spec } from '~/types'

/*
 * The landing page. Search lives at /search.
 *
 * These were one page: a hero, a search box and a results grid stacked on top
 * of each other, which served neither. A first-time visitor had to read past
 * the product to find out what it was, and a search submitted from the hero
 * had to scroll the visitor down past the explanation to reach its own answer.
 */
const router = useRouter()
const { data: stats } = useStats()
const f = useFormat()

const q = ref('')

function go(text?: string) {
  const next = (text ?? q.value).trim()
  router.push({ path: '/search', query: next ? { q: next } : {} })
}

/**
 * One live search is this page's evidence.
 *
 * The specimen car, its photographs, its per-source prices and the unit
 * argument all come from this one response, ranked exactly as the results page
 * ranks it. Nothing on this page is a screenshot of a good day.
 */
const { data: sample } = await useFetch<SearchResponse>(() => apiUrl('/api/v1/search'), {
  key: 'landing-sample',
  params: { q: '', mode: 'relevant', limit: 24 },
})

type Specimen = Spec & { image?: string | null, median_reliable?: boolean }

/**
 * The car the page argues with: best corroborated, photographed, uncontested.
 * No candidate means the section hides rather than falling back to a
 * remembered example. Whether that car can also carry the rial/toman argument
 * is LandingUnits' own question, and it hides itself when the answer is no.
 */
const specimen = computed<Specimen | null>(() => {
  const usable = ((sample.value?.result.specs ?? []) as Specimen[]).filter(s =>
    !!s.image
    // A flagged offer makes the price spread arguable, and the spread is the
    // headline. Argue it on a car nothing has been flagged on.
    && s.flag_count === 0
    && s.source_count >= 3)

  return usable.sort((a, b) =>
    b.source_count - a.source_count || b.offer_count - a.offer_count)[0] ?? null
})

const sourceCount = computed(() => stats.value?.stats.sources.length ?? null)

/**
 * The dedup rate, computed from the live index rather than remembered. It was
 * 40% on the day it was written down; the next crawl decides what it is today.
 */
const collapsedPct = computed(() => {
  const s = stats.value?.stats
  if (!s?.listings_captured || !s.duplicates_collapsed) return null
  return Math.round((s.duplicates_collapsed / s.listings_captured) * 100)
})

useSeoMeta({
  title: 'خودروبین | ترب برای خودروی دست‌دوم',
  description: 'آگهی‌های یک خودرو را از چند بازار زیر یک کارت می‌آوریم، واحدها و مدل‌ها را یکسان می‌کنیم، و میانه‌ی واقعی بازار را نشان می‌دهیم.',
})
</script>

<template>
  <main class="mx-auto max-w-[1080px] px-5 pb-24">
    <!-- Hero: one claim, one entry. Everything below exists to make the claim
         checkable, and the entry is the only interactive thing on the screen. -->
    <section class="pb-10 pt-14 text-center sm:pt-16" aria-labelledby="hero-h">
      <span class="inline-flex items-center gap-2 rounded-full border border-accent/30 bg-accent/[.12] px-3 py-1 font-mono text-[.7rem] tracking-wider text-accent" dir="ltr">
        crawl → normalize → rank → explain
      </span>
      <h1 id="hero-h" class="mx-auto mt-5 text-display">
        یک خودرو، <span class="text-accent">همه‌ی آگهی‌ها</span>،<br>یک قیمت واقعی.
      </h1>
      <p class="mx-auto mt-4 max-w-[54ch] text-lead text-ink-2">
        <template v-if="sourceCount">
          آگهی‌های <span class="mono-nums">{{ f.fa(sourceCount) }}</span> بازار خودرو را کنار هم می‌گذاریم،
        </template>
        <template v-else>
          آگهی‌های بازارهای خودرو را کنار هم می‌گذاریم،
        </template>
        واحدها و مدل‌ها را یکسان می‌کنیم، و میانه‌ی واقعی بازار را نشان می‌دهیم.
      </p>

      <div class="mx-auto mt-8 max-w-[720px]">
        <SearchBox v-model="q" @submit="go()" />

        <div class="mt-3.5 flex flex-wrap justify-center gap-2">
          <button
            v-for="ex in EXAMPLE_QUERIES"
            :key="ex"
            type="button"
            class="glass min-h-[44px] rounded-full px-4 py-2 text-[.8rem] text-ink-2 transition
                   hover:-translate-y-0.5 hover:text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            @click="go(ex)"
          >{{ ex }}</button>
        </div>
      </div>
    </section>

    <LandingProof :stats="stats" />

    <!-- The other way in: not a query, a listing someone already has open. -->
    <LookupBox />

    <LandingSpecimen v-if="specimen" :spec="specimen" />

    <LandingUnits v-if="specimen" :spec="specimen" />

    <LandingHowItWorks :collapsed-pct="collapsedPct" :source-count="sourceCount" />

    <LandingGuard />

    <LandingSources :sources="stats?.stats.sources ?? null" />

    <!-- The way back into the product for someone who read to the bottom. -->
    <section class="mt-[var(--space-section)]" aria-labelledby="cta-h">
      <div class="rounded-3xl border border-accent/25 bg-accent/[.07] px-6 py-10 text-center sm:px-10">
        <h2 id="cta-h" class="text-title">یک جمله بنویس، بقیه‌اش با ما</h2>
        <p class="mx-auto mt-2 max-w-[46ch] text-lead text-ink-2">
          «ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵» به همان اندازه کار می‌کند که یک فرم پر از فیلتر.
        </p>
        <NuxtLink
          to="/search"
          class="mt-6 inline-flex min-h-[48px] items-center rounded-xl bg-accent px-6 font-bold text-white transition
                 hover:brightness-110
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >رفتن به جست‌وجو</NuxtLink>
      </div>
    </section>

    <section class="mt-[var(--space-section)]" aria-labelledby="sub-h">
      <h2 id="sub-h" class="sr-only">خبرنامه</h2>
      <LandingSubscribe />
    </section>
  </main>
</template>
