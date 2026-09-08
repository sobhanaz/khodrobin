<script setup lang="ts">
import type { Spec } from '~/types'

const route = useRoute()
const f = useFormat()

// A spec key contains slashes (peugeot/207/base/mt/1404/0), so the route is a
// catch-all and the segments are rejoined rather than treated as a hierarchy.
const key = computed(() =>
  Array.isArray(route.params.key) ? route.params.key.join('/') : String(route.params.key))

const { data: spec } = await useFetch<Spec>(() => apiUrl(`/api/v1/specs/${key.value}`), {
  key: () => `spec:${key.value}`,
})

if (!spec.value) {
  throw createError({ statusCode: 404, statusMessage: 'این خودرو پیدا نشد', fatal: true })
}

const name = computed(() => [spec.value!.brand_fa, spec.value!.model_fa].filter(Boolean).join(' '))
const heading = computed(() => {
  const s = spec.value!
  const trim = s.trim ? ` ${s.trim}` : ''
  const gb = s.gearbox_fa ? ` ${s.gearbox_fa}` : ''
  return `${name.value}${trim} مدل ${f.year(s.year)}${gb}`
})

/**
 * Server-rendered per-car pages exist because this is a search product, and a
 * search product search engines cannot read is competing with one hand tied.
 *
 * The structured data below is the same schema.org markup Divar publishes —
 * which is exactly what made them readable when this project crawled them.
 */
useSeoMeta({
  title: () => `قیمت ${heading.value} | خودروبین`,
  description: () => spec.value
    ? `میانه‌ی قیمت ${heading.value} بر اساس ${f.fa(spec.value.offer_count)} آگهی از `
      + `${f.fa(spec.value.source_count)} منبع: ${f.money(spec.value.median_price)} تومان. `
      + `ارزان‌ترین ${f.money(spec.value.min_price)} تومان.`
    : '',
  ogType: 'website',
  ogImage: () => spec.value?.image ?? undefined,
})

useHead(() => ({
  link: [{ rel: 'canonical', href: `https://khodrobin.noxioai.com/car/${key.value}` }],
  script: spec.value
    ? [{
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Car',
          name: heading.value,
          brand: { '@type': 'Brand', name: spec.value.brand_fa },
          model: spec.value.model_fa || spec.value.model,
          vehicleTransmission: spec.value.gearbox_fa ?? undefined,
          productionDate: String(spec.value.year),
          image: spec.value.image ?? undefined,
          offers: {
            '@type': 'AggregateOffer',
            // schema.org has no toman. Stating IRR while publishing toman
            // figures would understate every price tenfold, so the numbers are
            // converted to the unit the currency code actually names.
            priceCurrency: 'IRR',
            lowPrice: spec.value.min_price * 10,
            highPrice: spec.value.max_price * 10,
            offerCount: spec.value.offer_count,
          },
        }),
      }]
    : [],
}))

// شیپور was crawled after this list was first written; without it, sheypoor
// offers appeared in the drawer but the badge row silently denied they exist.
const ALL_SOURCES = ['دیوار', 'باما', 'همراه‌مکانیک', 'خودرو۴۵', 'شیپور']
const bySource = computed(() => {
  const map = new Map<string, Spec['offers']>()
  for (const o of [...(spec.value?.offers ?? [])].sort((a, b) => a.price - b.price)) {
    const list = map.get(o.source_fa) ?? []
    list.push(o)
    map.set(o.source_fa, list)
  }
  return map
})
</script>

<template>
  <main v-if="spec" class="mx-auto max-w-[900px] px-5 py-12">
    <nav class="mb-6 flex flex-wrap items-center gap-2 text-[.8rem] text-ink-3" aria-label="مسیر">
      <NuxtLink to="/" class="hover:text-ink-2">خودروبین</NuxtLink>
      <span aria-hidden="true">›</span>
      <span class="text-ink-2">{{ spec.brand_fa }}</span>
      <span aria-hidden="true">›</span>
      <span class="text-ink">{{ heading }}</span>
    </nav>

    <div class="flex flex-col gap-5 sm:flex-row sm:items-start">
      <CarImage :src="spec.image ?? null" :alt="heading" class="!w-full sm:!w-56" />

      <div class="min-w-0 flex-1">
        <!-- No tracking-tight here: the heading is Persian, and Persian
             letters join; negative tracking pulls the strokes into each other. -->
        <h1 class="text-[clamp(1.4rem,3.5vw,2rem)] font-black">{{ heading }}</h1>

        <div class="mt-3 flex flex-wrap gap-1.5">
          <span class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 text-[.78rem] text-ink-2">
            {{ f.fa(spec.offer_count) }} آگهی
          </span>
          <span class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 text-[.78rem] text-ink-2">
            {{ f.fa(spec.source_count) }} منبع
          </span>
          <span
            v-if="spec.km_bucket != null"
            class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 text-[.78rem] text-ink-2"
          >
            {{ f.money(spec.km_bucket * 25) }}–{{ f.money((spec.km_bucket + 1) * 25) }} هزار کیلومتر
          </span>
          <span
            v-if="spec.flag_count"
            class="rounded-md border border-warn/25 bg-warn/[.12] px-2 py-0.5 text-[.78rem] text-warn"
          >
            {{ f.fa(spec.flag_count) }} تناقض
          </span>
        </div>

        <div class="mt-4 flex flex-wrap gap-1.5">
          <SourceBadge v-for="s in ALL_SOURCES" :key="s" :label="s" :offers="bySource.get(s) ?? []" />
        </div>

        <div class="mt-6 rounded-2xl border border-white/[.07] bg-surface p-5">
          <div class="text-[.75rem] text-ink-3">میانه‌ی بازار</div>
          <div dir="ltr" class="text-right sm:text-left">
            <span class="font-mono text-[1.6rem] font-bold tabular-nums">{{ f.money(spec.median_price) }}</span>
            <span class="ms-1 font-sans text-[.8rem] text-ink-3">تومان</span>
          </div>
          <PriceSpread :spec="spec" :delay="0" />
        </div>
      </div>
    </div>

    <!-- Reading order is the buyer's, not the system's: what it costs and
         where it sits → where the price has been → why this one → the raw
         offers → what to ask on the phone → what else to look at. -->
    <section class="mt-10">
      <h2 class="mb-3 text-[1.05rem] font-bold">نمودار قیمت</h2>
      <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
        <PriceHistory :spec-key="spec.key" />
      </div>
    </section>

    <section class="mt-10">
      <h2 class="mb-3 text-[1.05rem] font-bold">چرا این گزینه؟</h2>
      <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
        <WhyThis :spec-key="spec.key" />
      </div>
    </section>

    <section class="mt-10">
      <h2 class="mb-3 text-[1.05rem] font-bold">همه‌ی آگهی‌ها</h2>
      <div class="rounded-2xl border border-white/[.07] bg-surface px-5 py-2">
        <OfferRow v-for="(o, i) in spec.offers" :key="i" :offer="o" />
      </div>
      <p class="mt-3 text-[.78rem] leading-7 text-ink-3">
        آگهی‌ها متعلق به همان سایت‌هاست. برای تماس با فروشنده روی نام منبع بزن تا آگهی اصلی باز شود.
      </p>
    </section>

    <section class="mt-10">
      <h2 class="mb-3 text-[1.05rem] font-bold">قبل از تماس بپرس</h2>
      <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
        <CallChecklist :offers="spec.offers" />
      </div>
    </section>

    <!-- Renders its own section so an empty result removes the heading too. -->
    <SimilarCars :spec="spec" />

    <NuxtLink to="/" class="mt-10 inline-block text-[.88rem] text-accent hover:underline">
      ← جست‌وجوی خودروی دیگر
    </NuxtLink>
  </main>
</template>
