<script setup lang="ts">
import type { FilterKey, FilterOverrides, FilterPatch, Intent } from '~/types'

const props = defineProps<{
  /** Merged intent from the last response: the effective filter state. */
  intent: Intent | null
  /** Overrides as they stand in the URL; source of truth for rail-only fields. */
  filters: FilterOverrides
  /** Marketplace slugs from /api/v1/stats; empty while stats are unavailable. */
  sources: string[]
  activeCount: number
}>()
const emit = defineEmits<{ patch: [p: FilterPatch], clear: [] }>()
const f = useFormat()

/*
 * The rail owns no filter state. Every control reads either the merged intent
 * (price, year, gearbox — fields the text can also assert) or the URL
 * (sources, multi_only, unflagged — fields only the rail can set), and every
 * change is emitted as a URL patch. The chips above and this rail are two
 * views of the same address bar, which is the only reason they cannot drift.
 */

const priceMin = ref('')
const priceMax = ref('')
const yearMin = ref('')
const yearMax = ref('')

// Inputs re-seed from each response rather than being two-way bound: the URL
// round-trip normalizes what was typed («98» becomes 1398), and the field
// should show what is actually filtering, not what was keyed in.
watch(() => props.intent, (i) => {
  priceMin.value = i?.price_min ? String(i.price_min) : ''
  priceMax.value = i?.price_max ? String(i.price_max) : ''
  yearMin.value = i?.year_min ? String(i.year_min) : ''
  yearMax.value = i?.year_max ? String(i.year_max) : ''
}, { immediate: true })

function overrideFor(k: FilterKey, parsed: number | null): string | null {
  // An emptied field needs an explicit clear only when the text would
  // otherwise reassert a value; otherwise the param can simply go away.
  const has = props.intent?.[k as 'price_min' | 'price_max' | 'year_min' | 'year_max']
  return parsed ? String(parsed) : has ? '' : null
}

function applyPrices() {
  emit('patch', {
    price_min: overrideFor('price_min', parseTomans(priceMin.value)),
    price_max: overrideFor('price_max', parseTomans(priceMax.value)),
  })
}
function applyYears() {
  emit('patch', {
    year_min: overrideFor('year_min', parseJalaliYear(yearMin.value)),
    year_max: overrideFor('year_max', parseJalaliYear(yearMax.value)),
  })
}

const pricePreview = computed(() => {
  const lo = parseTomans(priceMin.value)
  const hi = parseTomans(priceMax.value)
  if (!lo && !hi) return null
  return [lo ? `از ${f.short(lo)}` : '', hi ? `تا ${f.short(hi)}` : ''].filter(Boolean).join(' ') + ' تومان'
})

const gearbox = computed(() => props.intent?.gearbox ?? '')
function setGearbox(v: string) {
  // «همه» writes an explicit clear, because the text may still say اتوماتیک
  // and the merge must be told to ignore it, not just left to reassert it.
  emit('patch', { gearbox: v })
}

/* Sources: absent from the URL means "all of them", which is also what a full
   selection collapses back to, so the param only exists while it excludes. */
const checkedSources = computed<Set<string>>(() => {
  const raw = props.filters.sources
  if (raw === undefined || raw === '') return new Set(props.sources)
  return new Set(raw.split(',').filter(Boolean))
})
function toggleSource(slug: string) {
  const next = new Set(checkedSources.value)
  if (next.has(slug)) next.delete(slug)
  else next.add(slug)
  // Unchecking the last box would ask for cars from nowhere; treat it as
  // "start over from everywhere" rather than rendering a guaranteed zero.
  if (next.size === 0 || next.size === props.sources.length) emit('patch', { sources: null })
  else emit('patch', { sources: props.sources.filter(s => next.has(s)).join(',') })
}

const multiOnly = computed(() => props.filters.multi_only === '1')
const unflagged = computed(() => props.filters.unflagged === '1')

/* ---- the mobile bottom sheet ---- */

const open = ref(false)
const panel = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)

function openSheet() {
  open.value = true
  void nextTick(() => focusables()[0]?.focus())
}
function close() {
  if (!open.value) return
  open.value = false
  trigger.value?.focus()
}

function focusables(): HTMLElement[] {
  return Array.from(
    panel.value?.querySelectorAll<HTMLElement>('button, input, select, [href], [tabindex]:not([tabindex="-1"])') ?? [],
  ).filter(el => !el.hasAttribute('disabled') && el.offsetParent !== null)
}

// A sheet that scrolls the page behind it, or lets Tab wander into it, is a
// sheet in name only. Focus cycles inside; Escape and the backdrop dismiss.
function onKeydown(e: KeyboardEvent) {
  if (!open.value) return
  if (e.key === 'Escape') { e.preventDefault(); close(); return }
  if (e.key !== 'Tab') return
  const els = focusables()
  if (!els.length) return
  const first = els[0]!
  const last = els[els.length - 1]!
  const active = document.activeElement
  if (e.shiftKey && (active === first || !panel.value?.contains(active))) {
    e.preventDefault(); last.focus()
  } else if (!e.shiftKey && active === last) {
    e.preventDefault(); first.focus()
  }
}

watch(open, (v) => {
  if (import.meta.client) document.documentElement.style.overflow = v ? 'hidden' : ''
})
onBeforeUnmount(() => {
  if (import.meta.client) document.documentElement.style.overflow = ''
})
// Rotating past the breakpoint while the sheet is open would leave the page
// scroll-locked behind a rail that no longer looks like a dialog.
onMounted(() => {
  const mq = window.matchMedia('(min-width: 1024px)')
  const onChange = (e: MediaQueryListEvent) => { if (e.matches) close() }
  mq.addEventListener('change', onChange)
  onBeforeUnmount(() => mq.removeEventListener('change', onChange))
})

const inputClass = 'mono-nums min-h-11 w-full rounded-lg border border-white/[.12] bg-surface-2 px-3 text-[.85rem] text-ink '
  + 'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
const checkClass = 'size-4 shrink-0 accent-good focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
</script>

<template>
  <div>
    <!-- On a phone the rail collapses into one button; the badge says how many
         filters are already narrowing the list before the sheet is opened. -->
    <button
      ref="trigger"
      type="button"
      class="glass inline-flex min-h-11 items-center gap-2 rounded-full px-5 text-[.85rem] text-ink transition hover:text-ink
             focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus lg:hidden"
      :aria-expanded="open"
      @click="open ? close() : openSheet()"
    >
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M3 5h18M6 12h12M10 19h4" stroke-linecap="round" />
      </svg>
      فیلترها
      <span
        v-if="activeCount"
        class="grid min-w-5 place-items-center rounded-full border border-good/40 bg-good/[.15] px-1 text-[.72rem] text-good"
      >{{ f.fa(activeCount) }}</span>
    </button>

    <Transition name="fade">
      <div v-if="open" class="fixed inset-0 z-40 bg-bg/80 lg:hidden" aria-hidden="true" @click="close" />
    </Transition>

    <section
      ref="panel"
      :role="open ? 'dialog' : undefined"
      :aria-modal="open ? 'true' : undefined"
      aria-label="فیلترها"
      class="lg:sticky lg:top-24 lg:block lg:max-h-[calc(100dvh-7rem)] lg:overflow-y-auto lg:rounded-2xl lg:border lg:border-white/[.08] lg:bg-surface/60 lg:p-4"
      :class="open
        ? 'sheet-in max-lg:fixed max-lg:inset-x-0 max-lg:bottom-0 max-lg:z-50 max-lg:max-h-[85dvh] max-lg:overflow-y-auto max-lg:rounded-t-2xl max-lg:border-t max-lg:border-white/[.12] max-lg:bg-bg-2 max-lg:p-5 max-lg:pb-8'
        : 'max-lg:hidden'"
      @keydown="onKeydown"
    >
      <div class="mb-3 flex items-center justify-between">
        <h3 class="text-[.95rem] font-bold text-ink">فیلترها</h3>
        <button
          type="button"
          class="grid size-11 place-items-center rounded-full text-ink-2 transition hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-4px] focus-visible:outline-focus lg:hidden"
          aria-label="بستن فیلترها"
          @click="close"
        >
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
            <path d="M6 6l12 12M18 6L6 18" stroke-linecap="round" />
          </svg>
        </button>
      </div>

      <div class="grid gap-5">
        <fieldset>
          <legend class="mb-2 text-[.78rem] font-bold text-ink-2">قیمت (تومان)</legend>
          <div class="grid grid-cols-2 gap-2">
            <label class="block">
              <span class="mb-1 block text-[.7rem] text-ink-3">از</span>
              <input v-model="priceMin" type="text" inputmode="numeric" :class="inputClass" @change="applyPrices">
            </label>
            <label class="block">
              <span class="mb-1 block text-[.7rem] text-ink-3">تا</span>
              <input v-model="priceMax" type="text" inputmode="numeric" :class="inputClass" @change="applyPrices">
            </label>
          </div>
          <p class="mt-1.5 min-h-4 text-[.72rem] text-ink-3">
            <template v-if="pricePreview">{{ pricePreview }}</template>
            <template v-else>عدد کوچک یعنی میلیون؛ «۵۰۰» یعنی ۵۰۰ میلیون</template>
          </p>
        </fieldset>

        <fieldset>
          <legend class="mb-2 text-[.78rem] font-bold text-ink-2">سال ساخت</legend>
          <div class="grid grid-cols-2 gap-2">
            <label class="block">
              <span class="mb-1 block text-[.7rem] text-ink-3">از</span>
              <input v-model="yearMin" type="text" inputmode="numeric" :class="inputClass" @change="applyYears">
            </label>
            <label class="block">
              <span class="mb-1 block text-[.7rem] text-ink-3">تا</span>
              <input v-model="yearMax" type="text" inputmode="numeric" :class="inputClass" @change="applyYears">
            </label>
          </div>
        </fieldset>

        <fieldset>
          <legend class="mb-1 text-[.78rem] font-bold text-ink-2">گیربکس</legend>
          <label v-for="g in [{ v: '', l: 'فرقی ندارد' }, { v: 'at', l: 'اتوماتیک' }, { v: 'mt', l: 'دنده‌ای' }]" :key="g.v"
                 class="flex min-h-11 cursor-pointer items-center gap-2.5 text-[.85rem] text-ink-2 has-checked:text-ink">
            <input
              type="radio" name="gearbox" :value="g.v" :checked="gearbox === g.v"
              :class="checkClass" @change="setGearbox(g.v)"
            >
            {{ g.l }}
          </label>
        </fieldset>

        <!-- Stats down means no slugs to offer; the section disappears rather
             than rendering boxes that could not do anything. -->
        <fieldset v-if="sources.length">
          <legend class="mb-1 text-[.78rem] font-bold text-ink-2">منبع آگهی</legend>
          <label v-for="s in sources" :key="s"
                 class="flex min-h-11 cursor-pointer items-center gap-2.5 text-[.85rem] text-ink-2 has-checked:text-ink">
            <input
              type="checkbox" :checked="checkedSources.has(s)"
              :class="checkClass" @change="toggleSource(s)"
            >
            {{ SOURCE_FA[s] ?? s }}
          </label>
        </fieldset>

        <fieldset>
          <legend class="mb-1 text-[.78rem] font-bold text-ink-2">اعتبار</legend>
          <label class="flex min-h-11 cursor-pointer items-center gap-2.5 text-[.85rem] text-ink-2 has-checked:text-ink">
            <input
              type="checkbox" :checked="multiOnly"
              :class="checkClass" @change="emit('patch', { multi_only: multiOnly ? null : '1' })"
            >
            فقط چندمنبعی
          </label>
          <label class="flex min-h-11 cursor-pointer items-center gap-2.5 text-[.85rem] text-ink-2 has-checked:text-ink">
            <input
              type="checkbox" :checked="unflagged"
              :class="checkClass" @change="emit('patch', { unflagged: unflagged ? null : '1' })"
            >
            بدون پرچم
          </label>
        </fieldset>

        <button
          v-if="activeCount"
          type="button"
          class="min-h-11 rounded-lg border border-white/[.12] text-[.8rem] text-ink-2 transition hover:border-white/[.3] hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          @click="emit('clear'); close()"
        >حذف همه فیلترها</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity .22s var(--ease-out-quint); }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* Enter-only: the closed state is display:none, which nothing can animate to. */
@media (max-width: 1023.98px) {
  .sheet-in { animation: sheet-in .3s var(--ease-out-quint); }
}
@keyframes sheet-in {
  from { transform: translateY(24px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}
</style>
