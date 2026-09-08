<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const { query, mode, data, pending, failed, run } = useSearch()
const f = useFormat()

/*
 * The URL is the query.
 *
 * A link to /search?q=… has to arrive with its results already in the HTML, so
 * the parameter is read before the first fetch and that fetch runs on the
 * server. The old combined home page never read ?q= at all: every shared
 * search opened on an empty box and the visitor had to type it again.
 */
query.value = typeof route.query.q === 'string' ? route.query.q : ''

/*
 * The handler hands the result back so it lands in the payload. Returning
 * nothing meant the browser hydrated against an empty ref, Vue threw the
 * server-rendered cards away as a mismatch, and every visit paid for the same
 * search twice. The failure flag travels with it for the same reason: without
 * it a search that failed on the server hydrated as «nothing found», which is
 * a different and much more misleading statement.
 */
const { data: firstPaint } = await useAsyncData(`search:${query.value}`, async () => {
  await run()
  return { result: data.value, failed: failed.value }
})
if (!data.value) {
  data.value = firstPaint.value?.result ?? null
  failed.value = firstPaint.value?.failed ?? false
}

function submit(q?: string) {
  if (q !== undefined) query.value = q
  const next = query.value.trim()
  // push, not replace: a search is a place, and back should return to the
  // previous one rather than to whatever page came before the whole session.
  void router.push({ path: '/search', query: next ? { q: next } : {} })
  void run(next)
}

// Back and forward change the address without remounting this page, so the
// results follow the address bar rather than the other way round. The equality
// check is what stops a submit running its own search twice.
watch(() => route.query.q, (v) => {
  const next = typeof v === 'string' ? v : ''
  if (next !== query.value) void run(next)
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

        <IntentChips :intent="data?.result.intent ?? null" />
      </div>
    </div>

    <section class="mt-10" aria-labelledby="results-h">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="results-h" class="text-head">نتیجه‌ها</h2>
          <p class="mt-1 text-[.88rem] text-ink-2" aria-live="polite">
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
