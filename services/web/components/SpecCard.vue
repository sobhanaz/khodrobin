<script setup lang="ts">
import type { Spec } from '~/types'

const props = defineProps<{ spec: Spec, index: number }>()
const f = useFormat()

const open = ref<'offers' | 'backstage' | null>(null)
function toggle(panel: 'offers' | 'backstage') {
  open.value = open.value === panel ? null : panel
}

const delay = computed(() => Math.min(props.index * 45, 400))

const ALL_SOURCES = ['دیوار', 'باما', 'همراه‌مکانیک', 'خودرو۴۵']
const present = computed(() => new Set(props.spec.offers.map(o => o.source_fa)))

const heading = computed(() => {
  const s = props.spec
  const name = [s.brand_fa, s.model_fa].filter(Boolean).join(' ')
  const trim = s.trim ? ` · ${s.trim}` : ''
  const gearbox = s.gearbox_fa ? ` · ${s.gearbox_fa}` : ''
  return `${name}${trim} — مدل ${f.year(s.year)}${gearbox}`
})

const mileageBand = computed(() => {
  const b = props.spec.km_bucket
  if (b == null) return null
  return `${f.money(b * 25)}–${f.money((b + 1) * 25)} هزار کیلومتر`
})
</script>

<template>
  <article
    class="card group overflow-hidden rounded-2xl border border-white/[.07] bg-surface transition
           hover:-translate-y-0.5 hover:border-white/[.12] hover:shadow-[0_18px_44px_-28px_#000]"
    :style="{ animationDelay: `${delay}ms` }"
  >
    <div class="flex items-start gap-4 p-4 sm:p-5">
      <CarImage :src="spec.image ?? null" :alt="heading" />

      <div class="min-w-0 flex-1">
        <h3 class="text-[1.06rem] font-bold tracking-tight">{{ heading }}</h3>

        <div class="mt-2 flex flex-wrap gap-1.5">
          <span class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 text-[.74rem] text-ink-2">
            {{ f.fa(spec.offer_count) }} آگهی
          </span>
          <span class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 text-[.74rem] text-ink-2">
            {{ f.fa(spec.source_count) }} منبع
          </span>
          <span v-if="mileageBand" class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 text-[.74rem] text-ink-2">
            {{ mileageBand }}
          </span>
          <span v-if="spec.flag_count" class="rounded-md border border-warn/25 bg-warn/[.12] px-2 py-0.5 text-[.74rem] text-warn">
            {{ f.fa(spec.flag_count) }} تناقض
          </span>
        </div>

        <div class="mt-2 flex flex-wrap gap-1.5">
          <span
            v-for="s in ALL_SOURCES"
            :key="s"
            class="rounded-full border px-2.5 py-0.5 text-[.7rem]"
            :class="present.has(s)
              ? 'border-good/30 bg-good/[.12] text-good'
              : 'border-white/[.12] bg-surface-2 text-ink-3'"
          >{{ s }}</span>
        </div>
      </div>

      <div class="shrink-0 text-left" dir="ltr">
        <div class="text-right font-sans text-[.68rem] text-ink-3" dir="rtl">میانه‌ی بازار</div>
        <div>
          <span class="font-mono text-[1.34rem] font-bold tabular-nums tracking-tight">{{ f.money(spec.median_price) }}</span>
          <span class="ms-1 font-sans text-[.72rem] text-ink-3">تومان</span>
        </div>
      </div>
    </div>

    <PriceSpread :spec="spec" :delay="delay" />

    <div class="flex flex-wrap gap-2 px-5 pb-4">
      <button
        type="button"
        class="rounded-full border px-3.5 py-1.5 text-[.78rem] transition"
        :class="open === 'offers'
          ? 'border-accent/35 bg-accent/[.12] text-accent'
          : 'border-white/[.07] bg-surface-2 text-ink-2 hover:text-ink hover:border-white/[.12]'"
        :aria-expanded="open === 'offers'"
        @click="toggle('offers')"
      >{{ f.fa(spec.offer_count) }} آگهی</button>
      <button
        type="button"
        class="rounded-full border px-3.5 py-1.5 text-[.78rem] transition"
        :class="open === 'backstage'
          ? 'border-accent/35 bg-accent/[.12] text-accent'
          : 'border-white/[.07] bg-surface-2 text-ink-2 hover:text-ink hover:border-white/[.12]'"
        :aria-expanded="open === 'backstage'"
        @click="toggle('backstage')"
      >پشت صحنه</button>
    </div>

    <!-- grid-template-rows animates to a natural height; max-height guessing
         either clips long content or eases against a number nobody can see. -->
    <div class="grid transition-[grid-template-rows] duration-[420ms]" :class="open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'">
      <div class="overflow-hidden">
        <div class="border-t border-white/[.07] bg-bg-2 px-5 py-4">
          <template v-if="open === 'offers'">
            <OfferRow v-for="(o, i) in spec.offers" :key="i" :offer="o" />
          </template>

          <template v-else-if="open === 'backstage'">
            <p class="mb-2 font-mono text-[.68rem] uppercase tracking-wider text-ink-3">
              امتیاز رتبه‌بندی — مجموع {{ spec.score.toFixed(3) }}
            </p>
            <ScoreBars :spec="spec" />

            <p class="mb-2 mt-4 font-mono text-[.68rem] uppercase tracking-wider text-ink-3">کلید یکسان‌سازی</p>
            <pre class="overflow-x-auto rounded-lg border border-white/[.07] bg-surface-2 px-3 py-2.5 text-left font-mono text-[.72rem] text-ink-3" dir="ltr">{{ spec.key }}</pre>

            <p class="mb-2 mt-4 font-mono text-[.68rem] uppercase tracking-wider text-ink-3">آنچه هر منبع گفته بود</p>
            <pre class="overflow-x-auto rounded-lg border border-white/[.07] bg-surface-2 px-3 py-2.5 text-left font-mono text-[.72rem] text-ink-3" dir="ltr">{{ spec.offers.map(o => `${o.source.padEnd(9)} brand=${o.raw_brand ?? '-'} model=${o.raw_model ?? '-'} trim=${o.raw_trim ?? '-'}`).join('\n') }}</pre>
          </template>
        </div>
      </div>
    </div>
  </article>
</template>

<style scoped>
.card { animation: rise .5s var(--ease-out-quint) both; }
@keyframes rise { from { opacity: 0; transform: translateY(14px) } to { opacity: 1; transform: none } }
</style>
