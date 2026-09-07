<script setup lang="ts">
import type { Offer } from '~/types'

const props = defineProps<{
  /** Persian display name of the marketplace, e.g. «دیوار». */
  label: string
  /** Every offer this spec has from this source, cheapest first. Empty = absent. */
  offers: Offer[]
}>()

const f = useFormat()

/**
 * The cheapest offer is what the badge links to.
 *
 * A source that lists the same spec five times should still be one badge — the
 * badge answers "is it here, and for how much", and the cheapest answer is the
 * one a buyer acts on. The rest stay visible in the offers drawer.
 */
const best = computed(() => props.offers[0] ?? null)
const present = computed(() => best.value !== null)

const open = ref(false)
// Touch devices have no hover. Tapping the badge should follow the link rather
// than reveal a preview the user then has to dismiss, so the preview is bound
// to pointer and keyboard focus only.
const hoverable = ref(false)
onMounted(() => {
  hoverable.value = window.matchMedia('(hover: hover) and (pointer: fine)').matches
})

const anchor = ref<HTMLElement | null>(null)
const pos = ref({ bottom: 0, left: 0 })

const PEEK_W = 288 // must match the w-72 below
const GUTTER = 12

/**
 * Positioned in the viewport, not in the card.
 *
 * As an absolutely-positioned child anchored `right-0`, this panel hung 288px
 * to the LEFT of its badge. For any badge near the left of the screen — which
 * in an RTL layout is most of them — that ran off the edge, and a live card
 * showed the price of a Peugeot 207 as «۰,۰۰۰ تومان». On a price-comparison
 * product a clipped price is not a layout bug, it is a wrong number.
 *
 * `fixed` also escapes the `overflow-hidden` on the card, which was clipping
 * whatever the viewport had not already cut off.
 */
function place() {
  const el = anchor.value
  if (!el) return
  const r = el.getBoundingClientRect()
  // Prefer alignment with the badge's right edge, then clamp into the viewport
  // so the panel is always fully readable, wherever the badge happens to sit.
  const wanted = r.right - PEEK_W
  const maxLeft = window.innerWidth - PEEK_W - GUTTER
  pos.value = {
    // Anchored by its bottom edge, so the panel sits above the badge without
    // needing a translate of its own. The transition owns `transform`; sharing
    // it would make the panel jump on every open.
    bottom: Math.round(window.innerHeight - r.top + 8),
    left: Math.round(Math.min(Math.max(GUTTER, wanted), maxLeft)),
  }
}

function show() {
  if (!hoverable.value) return
  place()
  open.value = true
}
function hide() { open.value = false }

// Scrolling with the pointer still down would otherwise leave the panel behind,
// pointing at nothing. Passive listeners on capture catch scrolls in any
// ancestor, not just the window.
onMounted(() => {
  window.addEventListener('scroll', hide, { passive: true, capture: true })
  window.addEventListener('resize', hide, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', hide, true)
  window.removeEventListener('resize', hide)
})

const freshness = computed(() => {
  const seen = best.value?.seen_at
  if (!seen) return null
  const hours = (Date.now() - new Date(seen).getTime()) / 36e5
  if (hours < 1) return 'کمتر از یک ساعت پیش'
  if (hours < 24) return `${f.fa(Math.round(hours))} ساعت پیش`
  return `${f.fa(Math.round(hours / 24))} روز پیش`
})
</script>

<template>
  <!-- Absent sources stay as inert text: a badge that looks clickable and is
       not is worse than one that plainly says "not listed here". -->
  <span
    v-if="!present"
    class="rounded-full border border-white/[.12] bg-surface-2 px-2.5 py-0.5 text-[.7rem] text-ink-3"
    :title="`${label}: آگهی‌ای برای این خودرو ندارد`"
  >{{ label }}</span>

  <span v-else ref="anchor" class="relative inline-block" @mouseenter="show" @mouseleave="hide">
    <a
      :href="best!.url"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex items-center gap-1 rounded-full border border-good/30 bg-good/[.12] px-2.5 py-0.5
             text-[.7rem] text-good transition hover:border-good/60 hover:bg-good/20
             focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-good"
      :aria-label="`${label} — ${f.money(best!.price)} تومان، باز کردن آگهی در تب جدید`"
      @focus="show"
      @blur="hide"
    >
      {{ label }}
      <svg viewBox="0 0 24 24" class="size-2.5 opacity-70" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
        <path d="M7 17L17 7M17 7H9M17 7v8" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </a>

    <!-- Teleported to the body, and it has to be.
         `position: fixed` is only relative to the viewport while no ancestor
         carries a transform. SpecCard has hover:-translate-y-0.5, and hovering
         a badge means hovering the card, so the card became the containing
         block at precisely the moment this panel opened: the coordinates were
         computed against the viewport and then resolved against the card,
         putting the panel 588px below the badge and off the screen. Leaving the
         card's subtree also escapes its overflow-hidden for good. -->
    <Teleport to="body">
      <Transition name="peek">
        <span
          v-if="open"
          role="tooltip"
          dir="rtl"
          class="fixed z-50 block w-72 max-w-[calc(100vw-1.5rem)] rounded-xl border border-white/[.12]
                 bg-surface p-3 text-right shadow-[0_24px_60px_-24px_#000]"
          :style="{ bottom: `${pos.bottom}px`, left: `${pos.left}px` }"
        >
          <span class="flex gap-3">
            <CarImage :src="best!.image ?? null" :alt="best!.title || label" class="!w-20 shrink-0" />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[.8rem] text-ink">{{ best!.title || label }}</span>
              <span class="mt-1 block font-mono text-[1rem] font-bold text-ink" dir="ltr">
                {{ f.money(best!.price) }}
                <span class="font-sans text-[.66rem] text-ink-3">تومان</span>
              </span>
              <span
                class="mt-1 inline-block rounded px-1.5 py-0.5 font-mono text-[.66rem]"
                dir="ltr"
                :class="best!.vs_median_pct < 0 ? 'bg-good/[.12] text-good' : 'bg-accent/[.12] text-accent'"
              >{{ best!.vs_median_pct > 0 ? '+' : '' }}{{ best!.vs_median_pct }}%</span>
            </span>
          </span>

          <span class="mt-2 block text-[.7rem] text-ink-3">
            <template v-if="best!.mileage_km != null">{{ f.money(best!.mileage_km) }} کیلومتر</template>
            <template v-if="best!.city"> · {{ best!.city }}</template>
            <template v-if="offers.length > 1"> · {{ f.fa(offers.length) }} آگهی در {{ label }}</template>
          </span>

          <!-- Freshness is stated, not implied. The crawler runs every three
               hours, so a price here can be hours old and the reader should know
               before they act on it. -->
          <span v-if="freshness" class="mt-1 block text-[.66rem] text-ink-3 opacity-75">
            آخرین بررسی: {{ freshness }}
          </span>

          <span class="mt-2 block text-[.66rem] text-accent">باز کردن در {{ label }} ↗</span>
        </span>
      </Transition>
    </Teleport>
  </span>
</template>

<style scoped>
.peek-enter-active, .peek-leave-active { transition: opacity .18s var(--ease-out-quint), transform .18s var(--ease-out-quint); }
.peek-enter-from, .peek-leave-to { opacity: 0; transform: translateY(4px) scale(.97); }
</style>
