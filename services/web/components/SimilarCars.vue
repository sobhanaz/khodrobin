<script setup lang="ts">
import type { Spec, SearchResponse } from '~/types'

const props = defineProps<{ spec: Spec }>()
const f = useFormat()

// image rides on the API's spec payload but not on the Spec type this run;
// read it through a narrowing instead of touching types.ts, which is owned
// by another change in flight.
const img = (s: Spec) => (s as Spec & { image?: string | null }).image ?? null

const similar = ref<{ spec: Spec, why: 'brand' | 'price' }[]>([])
const loaded = ref(false)

/**
 * Two searches, merged: same brand, and same money (median ±15%), both ranked
 * by value mode. The filter params are new on the API, and an older deploy
 * simply ignores unknown query strings and returns the full ranking — so every
 * returned spec is re-checked against the constraint CLIENT-SIDE. Against an
 * old API that check turns "everything" back into "same brand" / "same
 * budget"; against a broken one it leaves nothing, and the section hides
 * itself rather than showing unfiltered nonsense.
 */
onMounted(async () => {
  const pMin = Math.round(props.spec.median_price * 0.85)
  const pMax = Math.round(props.spec.median_price * 1.15)
  const base = { q: '', mode: 'value', limit: 12 }
  const [byBrand, byPrice] = await Promise.allSettled([
    $fetch<SearchResponse>(apiUrl('/api/v1/search'), { params: { ...base, brand: props.spec.brand } }),
    $fetch<SearchResponse>(apiUrl('/api/v1/search'), { params: { ...base, price_min: pMin, price_max: pMax } }),
  ])

  const seen = new Set([props.spec.key])
  const out: { spec: Spec, why: 'brand' | 'price' }[] = []
  const take = (r: PromiseSettledResult<SearchResponse>, why: 'brand' | 'price', ok: (s: Spec) => boolean) => {
    if (r.status !== 'fulfilled') return
    for (const s of r.value.result?.specs ?? []) {
      if (!seen.has(s.key) && ok(s)) { seen.add(s.key); out.push({ spec: s, why }) }
    }
  }
  take(byBrand, 'brand', s => s.brand === props.spec.brand)
  take(byPrice, 'price', s => s.median_price >= pMin && s.median_price <= pMax)

  similar.value = out.slice(0, 4)
  loaded.value = true
})

const name = (s: Spec) => [s.brand_fa, s.model_fa].filter(Boolean).join(' ')
</script>

<template>
  <!-- Rendered only once there is something true to show: while loading, and
       when both queries came back empty or failed, the section does not exist.
       A heading over an apology is worse than no heading. -->
  <section v-if="loaded && similar.length" class="mt-10" aria-labelledby="similar-h">
    <h2 id="similar-h" class="mb-1 text-[1.05rem] font-bold">مشابه‌ها</h2>
    <p class="mb-3 text-[.78rem] text-ink-3">همین برند، یا در پنجره‌ی ۱۵٪ بالا و پایین میانه‌ی همین خودرو؛ مرتب بر اساس بهترین ارزش.</p>

    <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <NuxtLink
        v-for="it in similar"
        :key="it.spec.key"
        :to="`/car/${it.spec.key}`"
        class="group flex flex-col gap-2.5 rounded-2xl border border-white/[.07] bg-surface p-3 transition
               hover:-translate-y-0.5 hover:border-white/[.12]
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      >
        <CarImage :src="img(it.spec)" :alt="name(it.spec)" class="!w-full" />
        <div class="min-w-0">
          <div class="truncate text-[.86rem] font-bold">{{ name(it.spec) }}</div>
          <div class="mt-0.5 text-[.72rem] text-ink-3">
            مدل {{ f.year(it.spec.year) }} · {{ f.fa(it.spec.offer_count) }} آگهی
          </div>
          <div dir="ltr" class="mt-1.5 text-right">
            <span class="font-mono text-[.9rem] font-bold tabular-nums">{{ f.money(it.spec.median_price) }}</span>
            <span class="ms-1 font-sans text-[.66rem] text-ink-3">تومان</span>
          </div>
          <span class="mt-1.5 inline-block rounded-md border border-white/[.07] bg-surface-2 px-1.5 py-0.5 text-[.66rem] text-ink-3">
            {{ it.why === 'brand' ? 'همین برند' : 'همین بودجه' }}
          </span>
        </div>
      </NuxtLink>
    </div>
  </section>
</template>
