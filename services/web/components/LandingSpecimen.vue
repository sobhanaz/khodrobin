<script setup lang="ts">
import type { Spec, Offer } from '~/types'

/**
 * One real car, as every marketplace is advertising it right now.
 *
 * Nothing here is written down: the card, the photographs, the prices and the
 * spread all come from the same live search response the results page uses, so
 * the claim on the landing page and the product behind it cannot drift apart.
 * Every row links to the listing it is quoting, because a proof a reviewer
 * cannot click is a screenshot.
 */
const props = defineProps<{
  // The API sends a photo per spec and per offer; the shared Spec type predates
  // both, so the extra fields are named here rather than guessed at elsewhere.
  spec: Spec & { image?: string | null, median_reliable?: boolean }
}>()

const f = useFormat()

const heading = computed(() => {
  const s = props.spec
  const name = [s.brand_fa, s.model_fa].filter(Boolean).join(' ')
  const trim = s.trim ? ` ${s.trim}` : ''
  const gearbox = s.gearbox_fa ? `، ${s.gearbox_fa}` : ''
  return `${name}${trim}، مدل ${f.year(s.year)}${gearbox}`
})

/** The cheapest offer each marketplace has for this exact car, dearest last. */
const perSource = computed(() => {
  const best = new Map<string, Offer & { image?: string | null }>()
  for (const o of props.spec.offers) {
    const held = best.get(o.source)
    if (!held || o.price < held.price) best.set(o.source, o)
  }
  return [...best.values()].sort((a, b) => a.price - b.price)
})

const spread = computed(() => props.spec.max_price - props.spec.min_price)

/**
 * Photos are hotlinked from each marketplace's own CDN and some of them are
 * that site's own placeholder, published as a path relative to their host, so
 * it resolves against us and 404s. A torn image icon over a price is worse
 * than no image; the tile behind it is already a neutral surface.
 */
function hide(e: Event) { (e.target as HTMLImageElement).hidden = true }
</script>

<template>
  <section class="mt-[var(--space-section)]" aria-labelledby="specimen-h">
    <h2 id="specimen-h" class="text-title">
      یک خودرو، {{ f.fa(perSource.length) }} بازار،
      <span class="text-accent">{{ f.short(spread) }} تومان</span> فاصله.
    </h2>
    <p class="mt-2 max-w-[58ch] text-lead text-ink-2">
      این کارت ساختگی نیست؛ همین حالا از فهرست زنده خوانده شد. هر ردیف عکس و قیمت خودِ آن بازار است و
      به آگهی اصلی لینک می‌دهد.
    </p>

    <!-- Photograph beside evidence, deliberately unequal: the car is the thing
         you recognise, the rail is the thing you check. -->
    <div class="mt-6 grid gap-5 lg:grid-cols-[minmax(0,.82fr)_minmax(0,1fr)] lg:items-start">
      <figure class="overflow-hidden rounded-2xl border border-white/[.07] bg-surface">
        <div class="aspect-4/3 w-full overflow-hidden bg-surface-2">
          <img
            v-if="spec.image"
            :src="spec.image"
            :alt="heading"
            loading="lazy"
            decoding="async"
            referrerpolicy="no-referrer"
            class="size-full object-cover"
            @error="hide"
          >
        </div>
        <figcaption class="p-4">
          <NuxtLink
            :to="`/car/${spec.key}`"
            class="text-head transition hover:text-accent
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >{{ heading }}</NuxtLink>
          <p class="mt-1.5 text-cap text-ink-3">
            {{ f.fa(spec.offer_count) }} آگهی از {{ f.fa(spec.source_count) }} منبع، زیر یک کارت.
          </p>
        </figcaption>
      </figure>

      <div>
        <ol class="grid gap-2">
          <li v-for="o in perSource" :key="o.source">
            <a
              :href="o.url"
              target="_blank"
              rel="noopener noreferrer"
              :aria-label="`آگهی ${o.source_fa} با قیمت ${f.money(o.price)} تومان، در سایت خودش باز می‌شود`"
              class="flex min-h-[64px] items-center gap-3 rounded-xl border border-white/[.07] bg-surface p-2.5 transition
                     hover:-translate-y-px hover:border-white/[.16]
                     focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            >
              <span class="size-12 shrink-0 overflow-hidden rounded-lg bg-surface-2">
                <img
                  v-if="o.image"
                  :src="o.image"
                  alt=""
                  loading="lazy"
                  decoding="async"
                  referrerpolicy="no-referrer"
                  class="size-full object-cover"
                  @error="hide"
                >
              </span>

              <span class="min-w-0 flex-1">
                <span class="block text-[.9rem] font-bold text-ink">{{ o.source_fa }}</span>
                <span class="block truncate text-cap text-ink-3">
                  {{ o.city || 'بدون شهر' }}
                  <template v-if="o.mileage_km != null"> · <span class="mono-nums">{{ f.money(o.mileage_km) }} km</span></template>
                </span>
              </span>

              <span class="shrink-0 text-left">
                <span class="block font-mono text-[1rem] font-bold tabular-nums text-ink" dir="ltr">{{ f.money(o.price) }}</span>
                <span
                  class="mt-0.5 inline-block rounded-md px-1.5 py-0.5 font-mono text-[.7rem]"
                  dir="ltr"
                  :class="o.vs_median_pct < 0 ? 'bg-good/[.14] text-good' : 'bg-accent/[.14] text-accent'"
                >{{ o.vs_median_pct > 0 ? '+' : '' }}{{ o.vs_median_pct }}%</span>
              </span>
            </a>
          </li>
        </ol>

        <div class="mt-3 rounded-xl border border-white/[.07] bg-bg-2 p-4">
          <!-- Below three offers the median is the mean of two asking prices
               nobody is asking, so the card says so instead of quoting it. -->
          <template v-if="spec.median_reliable !== false">
            <div class="flex items-baseline justify-between gap-3">
              <span class="text-cap text-ink-2">میانه‌ی بازار برای همین خودرو</span>
              <span dir="ltr">
                <span class="font-mono text-[1.15rem] font-bold tabular-nums text-ink">{{ f.money(spec.median_price) }}</span>
                <span class="ms-1 font-sans text-[.72rem] text-ink-3">تومان</span>
              </span>
            </div>
          </template>
          <p v-else class="text-cap text-ink-2">
            آگهی‌های این خودرو کمتر از آن است که میانه‌اش قابل دفاع باشد، پس میانه‌ای اعلام نمی‌کنیم.
          </p>

          <p class="mt-2 text-cap leading-7 text-ink-3">
            بین ارزان‌ترین و گران‌ترین آگهیِ همین خودرو
            <span class="text-ink-2">{{ f.short(spread) }} تومان</span>
            فاصله هست. تفاوت را کسی نمی‌بیند که فقط یک سایت را باز کرده باشد.
          </p>

          <NuxtLink
            :to="`/car/${spec.key}`"
            class="mt-3 inline-flex min-h-[44px] items-center text-[.85rem] text-accent transition hover:underline
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >هر {{ f.fa(spec.offer_count) }} آگهی این خودرو</NuxtLink>
        </div>
      </div>
    </div>
  </section>
</template>
