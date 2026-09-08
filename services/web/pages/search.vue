<script setup lang="ts">
import type { FilterKey, FilterPatch } from '~/types'

const route = useRoute()
const router = useRouter()
const { query, mode, data, pending, failed, filters, run } = useSearch()
const { data: stats } = useStats()
const f = useFormat()

/*
 * The URL is the query — all of it.
 *
 * A link to /search?q=…&price_max=… has to arrive with its FILTERED results
 * already in the HTML, so q and every override are read from the route before
 * the first fetch and that fetch runs on the server. The overrides live only
 * in the URL (useSearch reads route.query directly), which is what makes a
 * shared link arrive meaning what it meant to the person who shared it.
 */
query.value = typeof route.query.q === 'string' ? route.query.q : ''

/*
 * The handler hands the result back so it lands in the payload. Returning
 * nothing meant the browser hydrated against an empty ref, Vue threw the
 * server-rendered cards away as a mismatch, and every visit paid for the same
 * search twice. The failure flag travels with it for the same reason: without
 * it a search that failed on the server hydrated as «nothing found», which is
 * a different and much more misleading statement. Keyed by fullPath because
 * the same q under different overrides is a different result.
 */
const { data: firstPaint } = await useAsyncData(`search:${route.fullPath}`, async () => {
  await run()
  return { result: data.value, failed: failed.value }
})
if (!data.value) {
  data.value = firstPaint.value?.result ?? null
  failed.value = firstPaint.value?.failed ?? false
}

/*
 * One writer, one trigger. The chips and the rail only emit patches; this is
 * the single place that touches the router, and the fullPath watcher below is
 * the single place that re-runs the search. Navigation causing the fetch
 * (instead of racing beside it) is what keeps back/forward, chip removal and
 * rail edits all behaving identically.
 */
function apply(patch: FilterPatch) {
  const next: Record<string, string> = {}
  for (const [k, v] of Object.entries(route.query)) {
    next[k] = typeof v === 'string' ? v : v == null ? '' : String(v[0] ?? '')
  }
  for (const [k, v] of Object.entries(patch)) {
    if (v === null || v === undefined) delete next[k]
    else next[k] = v
  }
  void router.push({ path: '/search', query: next })
}

function clearAllFilters() {
  const q = query.value.trim()
  void router.push({ path: '/search', query: q ? { q } : {} })
}

/*
 * «فیلترها را شل کن» has to actually loosen something. Overrides in the URL
 * are dropped, and constraints the text itself asserts are suppressed with
 * explicit clears — «پژو زیر ۳۰۰» relaxes to «پژو», not to an empty page.
 * Brand and model survive on purpose: they are what the search is about.
 */
/*
 * Whether relaxAll would actually change the URL.
 *
 * activeCount includes brand and model, which relaxAll deliberately keeps, so
 * a query parsing to a brand alone («بنز») with no results offered a
 * «حذف محدودیت‌ها» button that pushed the identical URL: the route watcher
 * never fired and the click did nothing. An empty state must offer only
 * affordances that lead somewhere.
 */
const relaxable = computed(() => {
  const i = data.value?.result.intent
  const fs = filters.value
  return Boolean(
    i?.price_min || i?.price_max || i?.year_min || i?.year_max || i?.gearbox ||
    fs.sources || fs.multi_only === '1' || fs.unflagged === '1',
  )
})

function relaxAll() {
  const q = query.value.trim()
  const next: Record<string, string> = q ? { q } : {}
  const i = data.value?.result.intent
  if (i?.price_min || i?.price_max) { next.price_min = ''; next.price_max = '' }
  if (i?.year_min || i?.year_max) { next.year_min = ''; next.year_max = '' }
  if (i?.gearbox) next.gearbox = ''
  // The rail-only filters are overrides, so dropping the params is the relax.
  void router.push({ path: '/search', query: next })
}

function submit(q?: string) {
  if (q !== undefined) query.value = q
  const next = query.value.trim()
  // push, not replace: a search is a place, and back should return to the
  // previous one rather than to whatever page came before the whole session.
  // A fresh sentence is a fresh parse, so overrides do not survive it: keeping
  // «price_max=» from the last query would silently mute the new text.
  const target = { path: '/search', query: next ? { q: next } : {} }
  if (router.resolve(target).fullPath === route.fullPath) {
    void run(next)
    return
  }
  void router.push(target)
}

// Back and forward change the address without remounting this page, so the
// results follow the address bar rather than the other way round. Every push
// above lands here exactly once; the path guard keeps the fetch from firing
// while the router is already leaving for another page.
watch(() => route.fullPath, () => {
  if (route.path !== '/search') return
  query.value = typeof route.query.q === 'string' ? route.query.q : ''
  void run()
})

/** Overrides the URL holds as explicitly cleared, for the undo affordance. */
const cleared = computed(() =>
  Object.entries(filters.value).filter(([, v]) => v === '').map(([k]) => k as FilterKey),
)

/**
 * How many distinct things are narrowing the list right now. Counted off the
 * merged intent (plus the rail-only params), so a filter counts the same
 * whether the text asserted it or the rail did — it is the effective state
 * the badge, the count line and the relax affordance all describe.
 */
const activeCount = computed(() => {
  const i = data.value?.result.intent
  const fs = filters.value
  let n = 0
  if (i?.brand) n++
  if (i?.model) n++
  if (i?.gearbox) n++
  if (i?.price_min || i?.price_max) n++
  if (i?.year_min || i?.year_max) n++
  if (fs.sources) n++
  if (fs.multi_only === '1') n++
  if (fs.unflagged === '1') n++
  return n
})

useSeoMeta({
  title: () => (query.value ? `${query.value} · جست‌وجو در خودروبین` : 'جست‌وجوی خودروی دست‌دوم | خودروبین'),
  description: 'یک جمله‌ی فارسی معمولی بنویس. آگهی‌های چند بازار زیر یک کارت جمع می‌شوند و میانه‌ی واقعی قیمت را می‌بینی.',
  // A results page for an arbitrary query is not worth an index entry, and a
  // few thousand of them would spend the crawl budget the car pages need.
  robots: () => (query.value ? 'noindex, follow' : 'index, follow'),
})
</script>

<template>
  <main class="mx-auto max-w-[1080px] px-5 pb-20 pt-10">
    <div class="mx-auto max-w-[760px] text-center">
      <h1 class="text-title">جست‌وجوی خودرو</h1>
      <p class="mx-auto mt-2 max-w-[52ch] text-cap text-ink-2">
        یک جمله‌ی معمولی بنویس. برند، مدل، سال و بودجه را از خود جمله در می‌آوریم و نشانت می‌دهیم چه فهمیدیم.
      </p>

      <div class="mt-6">
        <SearchBox v-model="query" @submit="submit()" />

        <div class="mt-3.5 flex flex-wrap justify-center gap-2">
          <button
            v-for="ex in EXAMPLE_QUERIES"
            :key="ex"
            type="button"
            class="glass min-h-[44px] rounded-full px-4 py-2 text-[.8rem] text-ink-2 transition
                   hover:-translate-y-0.5 hover:text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            @click="submit(ex)"
          >{{ ex }}</button>
        </div>

        <IntentChips
          :intent="data?.result.intent ?? null"
          :cleared="cleared"
          @patch="apply"
        />
      </div>
    </div>

    <section class="mt-10" aria-labelledby="results-h">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="results-h" class="text-head">نتیجه‌ها</h2>
          <p class="mt-1 text-[.88rem] text-ink-2" aria-live="polite">
            <template v-if="data?.result.total">
              <b class="text-ink">{{ f.fa(data.result.total) }}</b>
              <template v-if="activeCount"> خودرو با این فیلترها</template>
              <template v-else> خودرو پیدا شد</template>
            </template>
            <template v-else-if="!pending">چیزی پیدا نشد</template>
          </p>
        </div>
        <p v-if="data" class="mono-nums text-[.7rem] text-ink-3">
          parse {{ data.timing.parse_ms.toFixed(2) }}ms · rank {{ data.timing.rank_ms.toFixed(2) }}ms · {{ data.result.mode }}
        </p>
      </div>

      <!-- In RTL the first grid column sits on the right, which is where a
           Persian reader expects the filters: before the results, not after. -->
      <div class="mt-3.5 lg:grid lg:grid-cols-[264px_minmax(0,1fr)] lg:items-start lg:gap-6">
        <FilterRail
          class="mb-3.5 lg:mb-0"
          :intent="data?.result.intent ?? null"
          :filters="filters"
          :sources="stats?.stats.sources ?? []"
          :active-count="activeCount"
          @patch="apply"
          @clear="clearAllFilters"
        />

        <div>
          <ModeTabs v-model="mode" />

          <div class="mt-3.5 grid gap-3">
            <template v-if="pending && !data">
              <div v-for="i in 3" :key="i" class="skeleton h-[104px] rounded-2xl" />
            </template>
            <p v-else-if="failed" class="py-14 text-center text-ink-3">ارتباط با سرور برقرار نشد.</p>
            <div v-else-if="!data?.result.specs.length" class="py-14 text-center">
              <template v-if="relaxable">
                <p class="text-ink-2">هیچ آگهی‌ای از این فیلترها رد نشد. فیلترها را شل کن.</p>
                <button
                  type="button"
                  class="mt-4 min-h-11 rounded-full border border-white/[.14] px-6 text-[.85rem] text-ink transition
                         hover:border-white/[.3] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
                  @click="relaxAll"
                >حذف محدودیت‌ها</button>
              </template>
              <p v-else class="text-ink-3">
                برای این جست‌وجو آگهی‌ای نداریم. یکی از نمونه‌های بالا را امتحان کن.
              </p>
            </div>
            <SpecCard
              v-for="(s, i) in data?.result.specs ?? []"
              :key="s.key"
              :spec="s"
              :index="i"
            />
          </div>
        </div>
      </div>
    </section>

    <!-- Someone who read a page of results is the person most likely to want
         told when one of these prices moves. -->
    <section class="mt-16" aria-labelledby="alert-h">
      <h2 id="alert-h" class="mb-3 text-head">وقتی قیمت تکان خورد، خبردار شو</h2>
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
