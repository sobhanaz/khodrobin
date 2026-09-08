<script setup lang="ts">
import type { Offer, Spec } from '~/types'

/*
 * The reverse direction, on the landing page.
 *
 * POST /api/v1/lookup shipped without any UI, and the moment it exists for is
 * not a search: it is someone on Divar at night with one listing open, asking
 * «گرونه؟». They paste the link and this box answers from the index, in the
 * endpoint's own vocabulary. Nothing here computes a verdict of its own; every
 * number and every failure message comes from the API, because a verdict this
 * box invented would be exactly the kind of number the product promises not to
 * show.
 */

/** Mirrors lookup.Result in services/api/internal/lookup/lookup.go. */
type LookupResult = {
  spec: Spec & { median_reliable?: boolean }
  offer: Offer
  cheapest_offer?: Offer
  overpay_pct?: number
  other_sources: number
  note?: string
}

const f = useFormat()
// Same reasoning as LandingSubscribe: this component may not be the only form
// on the page, and a literal id would silently orphan the second label.
const uid = useId()

const url = ref('')
const state = ref<'idle' | 'loading' | 'done' | 'error'>('idle')
const result = ref<LookupResult | null>(null)
const errorFa = ref('')
// 404 is an honest empty state («توی نمایه‌ی من نیست»), not a user mistake,
// so it renders neutral while a wrong-site link renders as guidance.
const errorKind = ref<'unknown' | 'missing' | 'net'>('net')

async function submit() {
  const raw = url.value.trim()
  if (!raw || state.value === 'loading') return
  state.value = 'loading'
  result.value = null
  try {
    result.value = await $fetch<LookupResult>(apiUrl('/api/v1/lookup'), {
      method: 'POST',
      body: { url: raw },
    })
    state.value = 'done'
  } catch (e: unknown) {
    // The endpoint already speaks Persian about its own failures, and it
    // distinguishes «سایتش را نمی‌شناسم» from «آگهی توی نمایه نیست» with
    // different advice in each. Rewriting those here would mean two places
    // that must agree on five marketplace names forever.
    const err = e as { data?: { error?: string, message_fa?: string } }
    const code = err?.data?.error
    errorKind.value = code === 'not_in_index' ? 'missing' : code ? 'unknown' : 'net'
    errorFa.value = err?.data?.message_fa
      ?? 'ارتباط برقرار نشد؛ یک لحظه بعد دوباره امتحان کن.'
    state.value = 'error'
  }
}

/**
 * A paste IS the submit. Nobody pastes a Divar URL into this field and then
 * wants to edit it by hand, so the default insert is suppressed, the field
 * takes the clipboard text whole, and the lookup fires — one gesture, one
 * verdict. Typing and pressing Enter still works for the rare hand-typed URL.
 */
function onPaste(e: ClipboardEvent) {
  const text = (e.clipboardData?.getData('text') ?? '').trim()
  if (!text) return
  e.preventDefault()
  url.value = text
  submit()
}

/** Same heading recipe the results page uses, so the car reads identically. */
const heading = computed(() => {
  const s = result.value?.spec
  if (!s) return ''
  const name = [s.brand_fa, s.model_fa].filter(Boolean).join(' ')
  const trim = s.trim ? ` · ${s.trim}` : ''
  const gearbox = s.gearbox_fa ? ` · ${s.gearbox_fa}` : ''
  return `${name}${trim} · مدل ${f.year(s.year)}${gearbox}`
})

/**
 * Position against the median, from the offer's own vs_median_pct. Withheld
 * entirely when the API says the median is not reliable: below three offers
 * the card refuses to quote the median, and a percentage against a number we
 * refuse to quote would be worse than no verdict.
 */
const vsMedian = computed(() => {
  const r = result.value
  if (!r || r.spec.median_reliable === false) return null
  const pct = r.offer.vs_median_pct
  const abs = Math.round(Math.abs(pct))
  if (abs < 1) return { above: false, text: 'تقریباً روی میانه‌ی بازار' }
  return {
    above: pct > 0,
    text: pct > 0
      ? `${f.fa(abs)}٪ بالاتر از میانه‌ی بازار`
      : `${f.fa(abs)}٪ پایین‌تر از میانه‌ی بازار`,
  }
})

/** «N٪ گران‌تر از ارزان‌ترین» — only when it rounds to a visible number. */
const overpay = computed(() => {
  const pct = result.value?.overpay_pct
  if (pct == null) return null
  const n = Math.round(pct)
  return n >= 1 ? `${f.fa(n)}٪ گران‌تر از ارزان‌ترین آگهی سالم همین خودرو` : null
})

/** Cheaper offers for the same spec, flagged ones included with their flags
 *  visible; OfferRow already renders those and hiding them would hide the
 *  reason a too-good price is too good. */
const cheaper = computed(() => {
  const r = result.value
  if (!r) return []
  return r.spec.offers
    .filter(o => o.url !== r.offer.url && o.price < r.offer.price)
    .sort((a, b) => a.price - b.price)
})
const CHEAPER_SHOWN = 4

/** Offers per marketplace, cheapest first, for the same badge row SpecCard shows. */
const bySource = computed(() => {
  const map = new Map<string, Offer[]>()
  const offers = result.value?.spec.offers ?? []
  for (const o of [...offers].sort((a, b) => a.price - b.price)) {
    const list = map.get(o.source_fa) ?? []
    list.push(o)
    map.set(o.source_fa, list)
  }
  return map
})
</script>

<template>
  <section class="mt-12 sm:mt-16" :aria-labelledby="`${uid}-h`">
    <div class="glass rounded-2xl p-5 sm:p-7">
      <h2 :id="`${uid}-h`" class="text-head">آگهی‌ای که باز داری گرونه؟</h2>
      <p class="mt-2 max-w-[52ch] text-[.9rem] leading-8 text-ink-2">
        لینکش را بچسبان؛ همه‌ی آگهی‌های همین خودرو را کنار هم می‌گذاریم و نشان می‌دهیم
        قیمتش کجای بازار ایستاده است.
      </p>

      <form class="mt-4 flex flex-col gap-2 sm:flex-row sm:items-start" novalidate @submit.prevent="submit">
        <div class="flex-1">
          <label :for="uid" class="mb-1.5 block text-[.85rem] text-ink-2">لینک آگهی را اینجا بچسبان</label>
          <input
            :id="uid"
            v-model="url"
            type="url"
            dir="ltr"
            inputmode="url"
            autocomplete="off"
            spellcheck="false"
            placeholder="https://divar.ir/v/…"
            :aria-describedby="`${uid}-hint`"
            class="min-h-[48px] w-full rounded-xl border border-white/[.12] bg-surface px-4 py-3 font-mono text-[.85rem] text-ink outline-none transition
                   placeholder:text-ink-3 focus:border-focus/60 focus:shadow-[0_0_0_4px_rgba(107,116,230,.16)]
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            @paste="onPaste"
          >
          <p :id="`${uid}-hint`" class="mt-1.5 text-[.76rem] leading-7 text-ink-3">
            دیوار، باما، همراه‌مکانیک، خودرو۴۵ و شیپور را می‌شناسیم. لینک همین‌جا می‌ماند و سراغ خود آگهی نمی‌رویم.
          </p>
        </div>

        <button
          type="submit"
          :disabled="!url.trim() || state === 'loading'"
          class="min-h-[48px] shrink-0 rounded-xl bg-accent px-5 font-bold text-white transition
                 enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus
                 sm:mt-[30px]"
        >{{ state === 'loading' ? 'در حال بررسی…' : 'بررسی کن' }}</button>
      </form>

      <!-- Always mounted, so the announcement actually announces; a live
           region born together with its message is a live region nobody hears. -->
      <div aria-live="polite" class="empty:hidden" :aria-busy="state === 'loading'">
        <!-- Loading: the shape of the answer, not a spinner in a void. -->
        <div v-if="state === 'loading'" class="mt-5 space-y-2.5" aria-hidden="true">
          <div class="h-5 w-2/3 animate-pulse rounded-md bg-surface-2" />
          <div class="h-9 w-1/2 animate-pulse rounded-md bg-surface-2" />
          <div class="h-4 w-full animate-pulse rounded-md bg-surface-2" />
        </div>

        <!-- The endpoint's own words for both failure kinds; 404 is a neutral
             empty state, a wrong-site link is guidance the API already wrote. -->
        <p
          v-else-if="state === 'error' && errorKind === 'missing'"
          class="mt-4 rounded-xl border border-white/[.12] bg-surface px-4 py-3 text-[.86rem] leading-8 text-ink-2"
        >{{ errorFa }}</p>
        <p
          v-else-if="state === 'error'"
          class="mt-4 flex items-start gap-2 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.86rem] leading-8 text-accent"
        >
          <span aria-hidden="true">⚠</span><span>{{ errorFa }}</span>
        </p>

        <div v-else-if="state === 'done' && result" class="mt-5 rounded-xl border border-white/[.07] bg-surface p-4 sm:p-5">
          <h3 class="text-[1.02rem] font-bold">
            <NuxtLink :to="`/car/${result.spec.key}`" class="transition hover:text-accent">{{ heading }}</NuxtLink>
          </h3>

          <!-- The verdict. Everything in it is the API's: the price is the
               offer's, the position is vs_median_pct, the overpay is
               overpay_pct against the cheapest clean offer. -->
          <div class="mt-3 flex flex-wrap items-end gap-x-8 gap-y-2">
            <div>
              <div class="font-sans text-[.68rem] text-ink-3">قیمت آگهی تو</div>
              <div dir="ltr" class="text-right">
                <span class="font-mono text-[1.3rem] font-bold tabular-nums tracking-tight">{{ f.money(result.offer.price) }}</span>
                <span class="ms-1 font-sans text-[.72rem] text-ink-3">تومان</span>
              </div>
            </div>
            <!-- The median is quoted only when the API stands behind it; below
                 three offers it is the mean of two asking prices nobody is
                 asking, and the card version refuses it for the same reason. -->
            <div v-if="result.spec.median_reliable !== false">
              <div class="font-sans text-[.68rem] text-ink-3">میانه‌ی بازار</div>
              <div dir="ltr" class="text-right">
                <span class="font-mono text-[1.02rem] font-bold tabular-nums tracking-tight text-ink-2">{{ f.money(result.spec.median_price) }}</span>
                <span class="ms-1 font-sans text-[.72rem] text-ink-3">تومان</span>
              </div>
            </div>
          </div>

          <div class="mt-2.5 flex flex-wrap gap-2">
            <span
              v-if="vsMedian"
              class="rounded-full px-3 py-1.5 text-[.82rem] font-bold"
              :class="vsMedian.above ? 'bg-accent/[.12] text-accent' : 'bg-good/[.12] text-good'"
            >{{ vsMedian.text }}</span>
            <span
              v-else
              class="rounded-full bg-surface-2 px-3 py-1.5 text-[.82rem] text-ink-2"
            >کمتر از سه آگهی داریم؛ میانه‌ی قابل اتکایی نیست.</span>

            <span v-if="overpay" class="rounded-full bg-warn/[.12] px-3 py-1.5 text-[.82rem] text-warn">
              {{ overpay }}
            </span>
            <span
              v-else-if="result.cheapest_offer && result.overpay_pct == null"
              class="rounded-full bg-good/[.12] px-3 py-1.5 text-[.82rem] text-good"
            >هیچ آگهی سالمی از این ارزان‌تر نیست.</span>
          </div>

          <!-- The pasted listing as the results page would show it, flags and
               all: if the ad contradicts itself, this is where the buyer
               finds out. -->
          <OfferRow :offer="result.offer" />

          <!-- The range the market actually spans, same plot as the cards. -->
          <div v-if="result.spec.offer_count > 1" class="-mx-4 mt-1 sm:-mx-5">
            <PriceSpread :spec="result.spec" :delay="0" />
          </div>

          <p class="mt-1 text-[.82rem] leading-7 text-ink-2">
            <template v-if="result.other_sources > 0">
              این خودرو در <span class="font-bold text-ink">{{ f.fa(result.other_sources) }}</span> بازار دیگر هم آگهی شده است.
            </template>
            <template v-else>
              فعلاً فقط همین بازار این خودرو را آگهی کرده است.
            </template>
          </p>

          <div class="mt-2 flex flex-wrap gap-2.5">
            <SourceBadge
              v-for="s in ALL_SOURCES"
              :key="s"
              :label="s"
              :offers="bySource.get(s) ?? []"
            />
          </div>

          <template v-if="cheaper.length">
            <p class="mt-4 font-mono text-[.68rem] uppercase tracking-wider text-ink-3">جاهای ارزان‌تر برای همین خودرو</p>
            <OfferRow v-for="(o, i) in cheaper.slice(0, CHEAPER_SHOWN)" :key="i" :offer="o" />
          </template>
          <p v-else-if="result.spec.offer_count > 1" class="mt-4 text-[.82rem] leading-7 text-ink-2">
            آگهی ارزان‌تری از این پیدا نکردیم.
          </p>

          <p class="mt-4 text-[.82rem]">
            <NuxtLink
              :to="`/car/${result.spec.key}`"
              class="text-accent transition hover:brightness-110
                     focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            >
              <template v-if="cheaper.length > CHEAPER_SHOWN">همه‌ی {{ f.fa(result.spec.offer_count) }} آگهی این خودرو ←</template>
              <template v-else>کارت کامل این خودرو ←</template>
            </NuxtLink>
          </p>
        </div>
      </div>
    </div>
  </section>
</template>
