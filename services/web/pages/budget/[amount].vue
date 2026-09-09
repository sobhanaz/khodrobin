<script setup lang="ts">
import type { SearchResponse, Spec } from '~/types'

/*
 * Budget-first entry: «۸۵۰ میلیون دارم، چی می‌تونم بخرم؟»
 *
 * Half the people in this market do not start from a model, they start from
 * money. Every other way into the product assumes you can already name the car
 * you want, which is exactly the assumption a first-time buyer cannot meet.
 *
 * The amount in the URL is WHOLE MILLIONS of toman: /budget/850 means
 * 850,000,000. That is how the number is said out loud here — nobody reads
 * «هشتصد و پنجاه میلیون» off nine digits — and it keeps the URL short enough to
 * type from memory or read out over the phone.
 */
const route = useRoute()
const f = useFormat()

// The accepted range is [1, 100000] million, inclusive at BOTH ends.
//
// This comment used to say both ends were rejected, which they are not: /budget/1
// and /budget/100000 each return a real page. The bound exists to stop the page
// asking the API a question whose answer could not be meaningful, and an in-range
// number that happens to match nothing renders the empty state, which tells a
// reader more than a validation error would.
const MAX_MILLIONS = 100_000

const millions = computed<number | null>(() => {
  // Persian digits survive a copy-paste from a chat message, so accept them.
  const raw = toEnDigits(String(route.params.amount ?? '')).trim()
  if (!/^\d{1,6}$/.test(raw)) return null
  const n = Number(raw)
  return n >= 1 && n <= MAX_MILLIONS ? n : null
})

const budget = computed(() => (millions.value ?? 0) * 1_000_000)

/**
 * How the amount is said. «۱٫۲ میلیارد» is what a person says for 1200, but
 * f.short rounds to one decimal, so 1250 would come back as «۱٫۳ میلیارد» —
 * a headline 4% richer than the filter that produced the page. Anything that
 * does not divide cleanly stays in millions, where it is exact.
 */
function say(m: number) {
  return m >= 1000 && m % 100 === 0 ? f.short(m * 1_000_000) : `${f.fa(m)} میلیون`
}
/** The «۱۰٪ بیشتر» amount, in millions, so it can also be a link. */
const stretchMillions = computed(() => Math.round((millions.value ?? 0) * 1.1))

/*
 * All three queries run on the server so the page arrives with its answer
 * already in the HTML. People land here from a search engine and from a link
 * someone sent them; a budget page that renders empty and fills in afterwards
 * is a budget page that looks like it found nothing.
 *
 * The two stretch queries catch their own failures instead of taking the page
 * down with them. They are an extra thought, not the answer: if they fail, that
 * row hides and the visitor still gets everything they came for.
 *
 * The amount is IN THE KEY, and that is not decoration. With a fixed key and a
 * watcher instead, tapping a preset from one budget page to another re-rendered
 * the new headline over the previous budget's cards — Nuxt found data already
 * cached under that key and never asked again. Keying by amount means a budget
 * this page has not fetched cannot resolve to another budget's answer.
 */
const { data, pending, error, refresh } = await useAsyncData(() => `budget:${millions.value}`, async () => {
  if (millions.value === null) return null
  const url = apiUrl('/api/v1/search')
  const stretchMax = stretchMillions.value * 1_000_000
  const [fits, more, wider] = await Promise.all([
    $fetch<SearchResponse>(url, {
      params: { q: '', mode: 'value', limit: 60, price_max: budget.value },
    }),
    // price_min keeps specs whose dearest offer reaches past the budget, which
    // is a superset of the ones this budget cannot afford — no candidate is
    // lost, and the pool shrinks enough that the best-value twelve are mostly
    // genuinely new. Without it the twelve best in the whole 0..+10% range were
    // cars the visitor could already afford, and /budget/200 and /budget/2000
    // hid the row entirely while more cars really had appeared.
    $fetch<SearchResponse>(url, {
      params: { q: '', mode: 'value', limit: 12, price_min: budget.value, price_max: stretchMax },
    }).catch(() => null),
    // Counting only. The honest «how many more» needs the unnarrowed total, and
    // a limit of one buys it for a few kilobytes.
    $fetch<SearchResponse>(url, {
      params: { q: '', mode: 'value', limit: 1, price_max: stretchMax },
    }).catch(() => null),
  ])
  return { fits, more, wider }
})

const specs = computed<Spec[]>(() => data.value?.fits.result.specs ?? [])
const total = computed(() => data.value?.fits.result.total ?? 0)

/*
 * Grouped by brand, not by price band.
 *
 * Price bands were the other candidate and they lose on this page: everything
 * here is under one ceiling by construction, so bands mostly re-state the
 * filter that built the list. Looking at the live answer for ۸۵۰ settled it —
 * ten brands, headed by تیبا and پراید — and «چه ماشینی؟» is the question a
 * budget is really asking. Groups are ordered by size so the shape of the
 * market at this budget is the first thing you see; inside a group the API's
 * value ranking is left exactly as it came.
 */
const groups = computed(() => {
  const map = new Map<string, Spec[]>()
  for (const s of specs.value) {
    const list = map.get(s.brand_fa) ?? []
    list.push(s)
    map.set(s.brand_fa, list)
  }
  return [...map.entries()]
    .map(([label, list]) => ({ label, slug: list[0]!.brand, specs: list }))
    .sort((a, b) => b.specs.length - a.specs.length)
})

/*
 * What ۱۰٪ more money would add.
 *
 * A spec counts as new when its cheapest offer sits above the current budget:
 * that is exactly the test the first query applied, so this needs no set
 * difference against a list that was truncated at 60 anyway.
 */
const stretchNew = computed<Spec[]>(() =>
  (data.value?.more?.result.specs ?? []).filter(s => s.min_price > budget.value))

/** Live totals, subtracted. Missing or not larger means the row has no number. */
const stretchExtra = computed(() => {
  const t = data.value?.wider?.result.total
  return t && t > total.value ? t - total.value : 0
})

useSeoMeta({
  title: () => millions.value
    ? `با ${say(millions.value)} تومان چه خودرویی می‌شود خرید؟ | خودروبین`
    : 'خرید خودرو با بودجه | خودروبین',
  description: () => millions.value
    ? `خودروهایی که با ${say(millions.value)} تومان دست‌کم یک آگهی زیر بودجه دارند، `
      + 'بر اساس قیمت واقعی چند بازار خودرو.'
    : 'بودجه‌ات را بنویس و ببین با آن چه خودروهایی در بازار هست.',
  // /budget/851 and /budget/852 are the same page twice. Only the round
  // amounts people actually say out loud are worth an index entry; the rest
  // stay crawlable but out of the index so they cannot eat the crawl budget
  // the car pages need.
  robots: () => (millions.value && millions.value % 50 === 0 ? 'index, follow' : 'noindex, follow'),
})

useHead(() => ({
  link: millions.value
    ? [{ rel: 'canonical', href: `https://khodrobin.noxioai.com/budget/${millions.value}` }]
    : [],
}))
</script>

<template>
  <main class="mx-auto max-w-[1080px] px-5 pb-24 pt-10">
    <!-- A route param is user input. Anything that is not a real amount lands
         here with a way to type a real one — never on a crash, and never on an
         empty grid that reads as «no such car exists». -->
    <section v-if="millions === null" class="mx-auto max-w-[640px] py-10">
      <h1 class="text-title text-center">عدد بودجه را نفهمیدیم</h1>
      <p class="mt-3 text-center text-lead text-ink-2">
        بودجه را به میلیون تومان بنویس. برای ۸۵۰ میلیون تومان، آدرس صفحه می‌شود
        <span class="mono-nums text-ink">/budget/850</span>.
      </p>
      <div class="mt-8">
        <BudgetEntry />
      </div>
    </section>

    <template v-else>
      <section class="mx-auto max-w-[760px] text-center">
        <h1 class="text-title">
          با {{ say(millions) }} تومان چه می‌شود خرید؟
        </h1>

        <p class="mt-3 text-lead text-ink-2" aria-live="polite">
          <template v-if="pending">در حال گرفتن نتیجه‌ها…</template>
          <template v-else-if="error">ارتباط با سرور برقرار نشد.</template>
          <template v-else-if="total">
            <b class="text-ink">{{ f.fa(total) }} خودرو</b> دست‌کم یک آگهی زیر این بودجه دارند.
          </template>
          <template v-else>با این بودجه هیچ آگهی‌ای پیدا نکردیم.</template>
        </p>

        <!-- The API filters on the CHEAPEST offer of each car, so a card can
             carry a median above the budget. Saying that once, here, is the
             difference between a page that is useful and a page that looks
             like it ignored the number you typed. -->
        <p v-if="!pending && !error && total" class="mt-2 text-cap text-ink-3">
          بودجه با ارزان‌ترین آگهی هر خودرو سنجیده می‌شود؛ عددی که روی هر کارت می‌بینی
          میانه‌ی بازار همان خودروست، نه قیمت آن آگهی.
        </p>
      </section>

      <div v-if="error" class="py-14 text-center">
        <button
          type="button"
          class="min-h-11 rounded-full border border-white/[.14] px-6 text-[.85rem] text-ink transition
                 hover:border-white/[.3] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          @click="refresh()"
        >دوباره تلاش کن</button>
      </div>

      <!-- Stale cards under a fresh headline would be a lie, so a re-query
           clears the grid rather than leaving the previous budget's answer up. -->
      <div v-else-if="pending" class="mt-10 grid gap-3">
        <div v-for="i in 4" :key="i" class="h-[104px] animate-pulse rounded-2xl bg-surface" />
      </div>

      <template v-else-if="total">
        <p class="mt-10 text-cap text-ink-3">
          {{ f.fa(specs.length) }} خودروی برتر از نظر ارزش، دسته‌بندی‌شده بر اساس برند.
        </p>

        <section v-for="g in groups" :key="g.label" class="mt-8" :aria-label="g.label">
          <div class="mb-3 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
            <h2 class="text-head">
              {{ g.label }}
              <span class="ms-2 text-[.8rem] font-normal text-ink-3">{{ f.fa(g.specs.length) }} خودرو</span>
            </h2>
            <NuxtLink
              :to="{ path: '/search', query: { brand: g.slug, price_max: String(budget) } }"
              class="text-[.82rem] text-ink-2 underline decoration-white/20 underline-offset-4 transition
                     hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            >همه‌ی {{ g.label }} با این بودجه</NuxtLink>
          </div>

          <div class="grid gap-3">
            <SpecCard v-for="(s, i) in g.specs" :key="s.key" :spec="s" :index="i" />
          </div>
        </section>
      </template>

      <!-- Nothing fits. The honest move is to say so and hand over the entry
           field, not to widen the search behind the visitor's back. -->
      <section v-else class="py-14">
        <p class="text-center text-ink-2">
          هیچ خودرویی با آگهی زیر {{ say(millions) }} تومان در فهرست ما نیست.
        </p>
        <div class="mt-8">
          <BudgetEntry :specs="stretchNew" />
        </div>
      </section>

      <!-- «با ۱۰٪ بیشتر» appears when the COUNT says more cars fit, not when a
           sample happens to contain one.
           This was gated on stretchNew.length, a twelve-item slice ranked by
           value rather than by price, so at /budget/3000 the row vanished while
           133 cars genuinely became affordable: none of the top twelve by value
           happened to sit above the budget. Visibility was a lottery over a
           ranking with nothing to do with price, under a comment claiming the
           row appeared whenever it "genuinely adds cars". -->
      <section v-if="!pending && stretchExtra" class="mt-[var(--space-section)]" aria-labelledby="stretch-h">
        <h2 id="stretch-h" class="text-head">
          با ۱۰٪ بیشتر، یعنی {{ say(stretchMillions) }}
        </h2>
        <p class="mt-1 text-cap text-ink-2">
          {{ f.fa(stretchExtra) }} خودروی دیگر هم به فهرست اضافه می‌شود.
          <template v-if="stretchNew.length">چند نمونه:</template>
        </p>

        <!-- The examples are a bonus, not the reason the row exists. The count
             above is the claim; these are whichever of the sample happen to sit
             above the budget, and there may be none. -->
        <div v-if="stretchNew.length" class="mt-4 grid gap-3">
          <SpecCard v-for="(s, i) in stretchNew.slice(0, 3)" :key="s.key" :spec="s" :index="i" />
        </div>

        <NuxtLink
          :to="`/budget/${stretchMillions}`"
          class="mt-4 inline-flex min-h-11 items-center rounded-full border border-white/[.14] px-5 text-[.85rem] text-ink transition
                 hover:border-white/[.3] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >دیدن همه با {{ say(stretchMillions) }}</NuxtLink>
      </section>

      <!-- BudgetEntry carries its own heading, so this section adds none. -->
      <section v-if="!pending && !error && total" class="mt-[var(--space-section)]">
        <BudgetEntry :specs="specs" />
      </section>
    </template>
  </main>
</template>
