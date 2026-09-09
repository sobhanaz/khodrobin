<script setup lang="ts">
import type { Spec } from '~/types'
import type { SavedSpec } from '~/composables/useSaved'

const { list, unsave, clear } = useSaved()
const { isLoggedIn } = useAuth()
const f = useFormat()

useSeoMeta({
  title: 'خودروهای ذخیره‌شده | خودروبین',
  description: 'خودروهایی که ستاره زده‌ای، با قیمت امروزشان کنار قیمتی که موقع ذخیره داشتند.',
  // Nothing here is the same for two visitors and none of it is server
  // rendered, so an index entry would point at an empty page.
  robots: 'noindex, follow',
})

/**
 * 'gone' and 'unreachable' are deliberately different verdicts.
 *
 * A 404 means the index no longer holds this spec — every offer for it expired
 * — and that is information the buyer wants. Any other failure is our problem,
 * not the car's, and printing «دیگر وجود ندارد» because a fetch timed out would
 * be the product asserting something it does not know.
 */
type State = 'live' | 'gone' | 'unreachable'
const today = ref(new Map<string, { spec: Spec | null, state: State }>())
const loading = ref(true)

/**
 * Re-fetched live, never read back from the snapshot. The snapshot is the
 * baseline, and a page that showed it as today's price would quietly become a
 * museum of prices that have all since moved.
 *
 * One request per saved car, up to the 60-entry cap. No batching endpoint
 * exists and none is worth adding: each read is an in-memory map lookup on the
 * Go side, and the browser already caps itself at ~6 connections per origin.
 */
// skeletons=false keeps the rows on screen while refetching.
//
// «تلاش دوباره» used to blank every row, including the saved prices that were
// still perfectly readable, and replace them with skeletons for the duration of
// the fetch. The one case it exists for is a slow or failing API, which is
// exactly the case where that fetch takes longest, so the button destroyed what
// it was meant to recover.
async function load(skeletons = true) {
  loading.value = skeletons
  const saved = [...list.value]
  const settled = await Promise.allSettled(
    saved.map(s => $fetch<Spec>(apiUrl(`/api/v1/specs/${s.key}`))),
  )
  const next = new Map<string, { spec: Spec | null, state: State }>()
  saved.forEach((s, i) => {
    const r = settled[i]!
    if (r.status === 'fulfilled') {
      next.set(s.key, { spec: r.value, state: 'live' })
      return
    }
    const status = (r.reason as { status?: number, statusCode?: number })?.status
      ?? (r.reason as { statusCode?: number })?.statusCode
    next.set(s.key, { spec: null, state: status === 404 ? 'gone' : 'unreachable' })
  })
  today.value = next
  loading.value = false
}

// useSaved() above registers its own onMounted, and Vue fires hooks in
// registration order, so list.value is already hydrated out of localStorage by
// the time this one runs. That ordering is the whole reason a page whose only
// data source is localStorage can fetch on mount without waiting a tick first.
onMounted(load)

interface Row { saved: SavedSpec, spec: Spec | null, state: State }

// Derived from the list rather than copied out of it, so removing a car takes
// its row with it without a second bookkeeping pass.
const rows = computed<Row[]>(() =>
  list.value.map(s => ({
    saved: s,
    ...(today.value.get(s.key) ?? { spec: null, state: 'unreachable' as State }),
  })),
)

const unreachable = computed(() => rows.value.filter(r => r.state === 'unreachable').length)

/** Today's median, but only when the index says it is worth quoting. */
function median(spec: Spec | null): number | null {
  const s = spec as (Spec & { median_reliable?: boolean }) | null
  if (!s) return null
  return s.median_reliable === false ? null : s.median_price
}

/**
 * What moved, in tomans and percent.
 *
 * Anything under a million is reported as no movement. f.short rounds to whole
 * millions, so a 200,000 toman drift renders as «۰ میلیون ارزان‌تر» — a
 * sentence that is both nonsense and, at that size, noise in a market where
 * asking prices are quoted in tens of millions.
 */
function delta(r: Row): { diff: number, pct: number } | null {
  const now = median(r.spec)
  const then = r.saved.median
  if (now == null || then == null || then <= 0) return null
  const diff = now - then
  if (Math.abs(diff) < 1_000_000) return null
  return { diff, pct: (diff / then) * 100 }
}

const rtf = new Intl.RelativeTimeFormat('fa-IR', { numeric: 'auto' })
function ago(iso: string): string {
  const t = Date.parse(iso)
  if (!Number.isFinite(t)) return ''
  return rtf.format(Math.round((t - Date.now()) / 86_400_000), 'day')
}

function confirmClear() {
  // Native confirm on purpose. The alternative is a modal, a focus trap and an
  // escape handler, all to guard a list the user can rebuild in three clicks.
  if (window.confirm('همه‌ی خودروهای ذخیره‌شده پاک شوند؟')) clear()
}

const ring = 'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
const pill = `inline-flex min-h-[44px] items-center rounded-full border border-white/[.07] bg-surface-2 px-3.5 `
  + `text-[.78rem] text-ink-2 transition hover:border-white/[.12] hover:text-ink ${ring}`
</script>

<template>
  <main class="mx-auto max-w-[900px] px-5 pb-20 pt-12">
    <h1 class="text-title">ذخیره‌شده‌ها</h1>
    <p class="mt-2 max-w-[58ch] text-cap text-ink-2">
      قیمت‌ها همین حالا دوباره از ایندکس خوانده شدند. چیزی که می‌بینی قیمت امروز است، نه قیمت روزی که ذخیره کردی.
    </p>

    <div v-if="loading" class="mt-8 grid gap-3" aria-live="polite" aria-busy="true">
      <div v-for="i in 3" :key="i" class="skeleton h-[132px] rounded-2xl" />
      <span class="sr-only">در حال گرفتن قیمت امروز</span>
    </div>

    <template v-else>
      <!-- Empty is the state most visitors see first, so it has to teach the
           feature rather than apologise for having no data. -->
      <div v-if="!rows.length" class="mt-8 rounded-2xl border border-white/[.07] bg-surface p-8 text-center">
        <p class="text-[1rem] font-bold">هنوز خودرویی ذخیره نکرده‌ای.</p>
        <p class="mx-auto mt-3 max-w-[48ch] text-[.88rem] leading-8 text-ink-2">
          روی هر کارت نتیجه دکمه‌ی «ذخیره» هست. وقتی بزنی، میانه‌ی قیمت همان لحظه هم ثبت می‌شود و
          دفعه‌ی بعد که برگردی، اینجا می‌بینی از آن روز تا امروز چقدر بالا یا پایین رفته.
        </p>
        <NuxtLink to="/search" :class="`mt-6 ${pill}`">رفتن به جست‌وجو</NuxtLink>
      </div>

      <template v-else>
        <div class="mt-8 flex flex-wrap items-center justify-between gap-3">
          <p class="text-[.88rem] text-ink-2" aria-live="polite">
            <b class="text-ink">{{ f.fa(rows.length) }}</b> خودرو در این مرورگر
          </p>
          <button type="button" :class="pill" @click="confirmClear">پاک کردن همه</button>
        </div>

        <!-- Partial failure gets its own banner because the rows underneath it
             still show a real saved price, and without this line they read as
             cars whose price simply did not move. -->
        <div
          v-if="unreachable"
          class="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-warn/25 bg-warn/[.12] p-4 text-[.85rem] leading-8 text-warn"
        >
          <span>قیمت امروزِ {{ f.fa(unreachable) }} خودرو را از سرور نگرفتیم.</span>
          <button
            type="button"
            :class="`min-h-[44px] rounded-full border border-warn/35 px-4 text-[.8rem] ${ring}`"
            @click="load(false)"
          >تلاش دوباره</button>
        </div>

        <div class="mt-4 grid gap-3">
          <article
            v-for="r in rows"
            :key="r.saved.key"
            class="rounded-2xl border border-white/[.07] bg-surface p-4 sm:p-5"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-5">
              <div class="min-w-0">
                <h2 class="text-[1rem] font-bold">
                  <!-- The title is the row's way into the car, and on a phone
                       it is the biggest thing to tap, so it gets the full 44px
                       target. The negative margin gives that height back to the
                       layout: the hit area grows, the rhythm does not move. -->
                  <NuxtLink
                    v-if="r.state === 'live'"
                    :to="`/car/${r.saved.key}`"
                    :class="`-my-2.5 inline-flex min-h-[44px] items-center transition hover:text-accent ${ring}`"
                  >{{ r.saved.name }}</NuxtLink>
                  <span v-else>{{ r.saved.name }}</span>
                </h2>
                <p v-if="r.saved.at" class="mt-1 text-[.76rem] text-ink-3">ذخیره‌شده {{ ago(r.saved.at) }}</p>
              </div>

              <div v-if="median(r.spec) !== null" class="shrink-0 sm:text-left">
                <div class="text-[.68rem] text-ink-3">میانه‌ی امروز</div>
                <div dir="ltr" class="text-right sm:text-left">
                  <span class="font-mono text-[1.15rem] font-bold tabular-nums">{{ f.money(median(r.spec)!) }}</span>
                  <span class="ms-1 font-sans text-[.72rem] text-ink-3">تومان</span>
                </div>
              </div>
            </div>

            <div class="mt-3 border-t border-white/[.07] pt-3 text-[.83rem] leading-8">
              <!-- The car left the index. Shown, never silently dropped: a car
                   disappearing from five marketplaces at once is itself a fact
                   about the market, and the price it was last seen at is the
                   only record of it anyone has. -->
              <p v-if="r.state === 'gone'" class="text-ink-2">
                این خودرو دیگر در ایندکس نیست؛ آگهی‌هایش منقضی شده‌اند.
                <template v-if="r.saved.median != null">
                  آخرین چیزی که ازش داریم همان
                  <span class="mono-nums text-ink">{{ f.money(r.saved.median) }}</span>
                  تومانی است که موقع ذخیره ثبت شد.
                </template>
              </p>

              <p v-else-if="r.state === 'unreachable'" class="text-ink-2">
                قیمت امروزش را نگرفتیم.
                <template v-if="r.saved.median != null">
                  وقتی ذخیره کردی
                  <span class="mono-nums text-ink">{{ f.money(r.saved.median) }}</span>
                  تومان بود.
                </template>
              </p>

              <!-- Live, but today's median is not quotable. Saying so beats
                   printing a number the rest of the site refuses to print. -->
              <p v-else-if="median(r.spec) === null" class="text-ink-2">
                امروز کمتر از سه آگهی برای این خودرو داریم، پس میانه‌اش را قیمت بازار حساب نمی‌کنیم.
              </p>

              <!-- Saved while its median was unreliable: there is no baseline,
                   and a ۰٪ would be an invented one. -->
              <p v-else-if="r.saved.median == null" class="text-ink-2">
                وقتی ذخیره کردی، میانه‌ی این خودرو هنوز قابل اتکا نبود، پس مبنایی برای مقایسه نداریم.
                از حالا به بعد قیمت امروزش را داری.
              </p>

              <p v-else class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                <span class="text-ink-3">وقتی ذخیره کردی</span>
                <span class="mono-nums text-ink-2">{{ f.money(r.saved.median) }}</span>
                <span class="text-ink-3">تومان.</span>

                <!-- The sentence the whole feature exists for. -->
                <template v-if="delta(r)">
                  <span :class="delta(r)!.diff < 0 ? 'font-bold text-good' : 'font-bold text-warn'">
                    {{ f.short(Math.abs(delta(r)!.diff)) }}
                    {{ delta(r)!.diff < 0 ? 'ارزان‌تر' : 'گران‌تر' }} از وقتی ذخیره کردی
                  </span>
                  <span class="text-ink-3">({{ f.fa(Math.abs(Math.round(delta(r)!.pct * 10) / 10)) }}٪)</span>
                </template>
                <span v-else class="text-ink-2">از آن روز تکان نخورده.</span>
              </p>
            </div>

            <div class="mt-3">
              <button
                type="button"
                :class="pill"
                :aria-label="`حذف ${r.saved.name} از ذخیره‌ها`"
                @click="unsave(r.saved.key)"
              >حذف</button>
            </div>
          </article>
        </div>
      </template>

      <!--
        The honest pitch. Signing in does NOT sync this list — nothing in the
        backend stores per-user saved specs, and saying otherwise would be a
        promise the product cannot keep the first time someone opens their
        phone. What an account really adds is the alert watcher, which ships
        and runs every three hours.
      -->
      <section class="mt-12 rounded-2xl border border-white/[.07] bg-surface p-5" aria-labelledby="upgrade-h">
        <h2 id="upgrade-h" class="text-[1.05rem] font-bold">این فهرست فقط در همین مرورگر است</h2>
        <p class="mt-2 max-w-[62ch] text-[.86rem] leading-8 text-ink-2">
          ذخیره‌ها در حافظه‌ی همین مرورگر می‌مانند؛ روی گوشی‌ات نیستند و با پاک کردن داده‌های مرورگر می‌روند.
          ورود به حساب هم آن‌ها را همگام نمی‌کند، چون ما چیزی از این فهرست روی سرور نگه نمی‌داریم.
        </p>
        <p class="mt-2 max-w-[62ch] text-[.86rem] leading-8 text-ink-2">
          کاری که حساب واقعاً اضافه می‌کند این است: روی یک جست‌وجو هشدار قیمت بگذاری. هر ۳ ساعت همان
          جست‌وجو دوباره اجرا می‌شود و اگر میانه‌ی قیمت بیشتر از آستانه‌ات جابه‌جا شد، ایمیل می‌گیری.
        </p>
        <NuxtLink :to="isLoggedIn ? '/account' : '/login?next=/account'" :class="`mt-4 ${pill}`">
          {{ isLoggedIn ? 'ساختن هشدار قیمت' : 'ورود و ساختن هشدار قیمت' }}
        </NuxtLink>
      </section>
    </template>
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
