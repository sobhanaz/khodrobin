<script setup lang="ts">
import type { FilterKey, FilterPatch, Intent } from '~/types'

const props = defineProps<{
  intent: Intent | null
  /** Fields the URL explicitly cleared («?price_max=»), so they can come back. */
  cleared?: FilterKey[]
}>()
const emit = defineEmits<{ patch: [p: FilterPatch] }>()
const f = useFormat()

/**
 * What the machine understood, shown back to the user — and now something to
 * hold. Every chip renders from the MERGED intent the API returned, so it is
 * the effective filter state, not the raw parse. Removing one never edits the
 * typed query; it writes an explicit empty override into the URL and lets the
 * API's merge suppress the parsed value. The text stays exactly as typed,
 * which is what makes the removal undoable.
 */
interface Chip {
  id: string
  k: string
  v: string
  /** URL params an × clears. An exact year is one chip but two params. */
  params: FilterKey[]
  price?: boolean
}

const chips = computed<Chip[]>(() => {
  const i = props.intent
  if (!i) return []
  const out: Chip[] = []
  if (i.brand_fa) out.push({ id: 'brand', k: 'برند', v: i.brand_fa, params: ['brand'] })
  if (i.model_fa) out.push({ id: 'model', k: 'مدل', v: i.model_fa, params: ['model'] })
  // «مدل ۹۸» parses to min == max. That is one fact to the user, so it is one
  // chip, and its × must clear both bounds or removal flips it into «تا سال».
  const exactYear = !!i.year_min && i.year_min === i.year_max
  if (i.year_min) {
    out.push({
      id: 'year_min', k: exactYear ? 'سال' : 'از سال', v: f.year(i.year_min),
      params: exactYear ? ['year_min', 'year_max'] : ['year_min'],
    })
  }
  if (i.year_max && !exactYear) out.push({ id: 'year_max', k: 'تا سال', v: f.year(i.year_max), params: ['year_max'] })
  if (i.price_max) out.push({ id: 'price_max', k: 'حداکثر', v: f.short(i.price_max), params: ['price_max'], price: true })
  if (i.price_min) out.push({ id: 'price_min', k: 'حداقل', v: f.short(i.price_min), params: ['price_min'], price: true })
  if (i.gearbox) out.push({ id: 'gearbox', k: 'گیربکس', v: i.gearbox === 'at' ? 'اتوماتیک' : 'دنده‌ای', params: ['gearbox'] })
  return out
})

function remove(c: Chip) {
  if (c.price) editing.value = false
  const p: FilterPatch = {}
  for (const k of c.params) p[k] = ''
  emit('patch', p)
}

/*
 * An accidental × is one tap to undo. A cleared field shows a quiet
 * «بازگرداندن» chip; restoring deletes the override from the URL entirely, so
 * the field falls back to whatever the text still says. Bounds that were
 * cleared together collapse into one entry, mirroring how they were removed.
 */
const RESTORE_FA: Partial<Record<FilterKey, string>> = {
  brand: 'برند', model: 'مدل', gearbox: 'گیربکس',
  price_min: 'حداقل قیمت', price_max: 'حداکثر قیمت',
  year_min: 'از سال', year_max: 'تا سال',
}

const restorable = computed(() => {
  const left = new Set(props.cleared ?? [])
  const out: { id: string, label: string, params: FilterKey[] }[] = []
  const pair = (a: FilterKey, b: FilterKey, label: string) => {
    if (left.has(a) && left.has(b)) {
      left.delete(a); left.delete(b)
      out.push({ id: `${a}+${b}`, label, params: [a, b] })
    }
  }
  pair('year_min', 'year_max', 'سال')
  pair('price_min', 'price_max', 'قیمت')
  for (const k of left) {
    const label = RESTORE_FA[k]
    if (label) out.push({ id: k, label, params: [k] })
  }
  return out
})

function restore(r: { params: FilterKey[] }) {
  const p: FilterPatch = {}
  for (const k of r.params) p[k] = null
  emit('patch', p)
}

/* Tapping a price chip opens a small inline editor instead of only offering
   removal, because "wrong budget" is far more common than "no budget". */
const editing = ref(false)
const minRaw = ref('')
const maxRaw = ref('')
const minEl = ref<HTMLInputElement | null>(null)

function openEditor() {
  minRaw.value = props.intent?.price_min ? String(props.intent.price_min) : ''
  maxRaw.value = props.intent?.price_max ? String(props.intent.price_max) : ''
  editing.value = true
  void nextTick(() => minEl.value?.focus())
}

const minParsed = computed(() => parseTomans(minRaw.value))
const maxParsed = computed(() => parseTomans(maxRaw.value))

// Assembled in script because the template condenses the whitespace between
// adjacent v-if blocks, which glued «میلیون» to «تومان» on a live preview.
const previewText = computed(() => {
  const parts: string[] = []
  if (minParsed.value) parts.push(`از ${f.short(minParsed.value)}`)
  if (maxParsed.value) parts.push(`تا ${f.short(maxParsed.value)}`)
  return parts.length ? `${parts.join(' ')} تومان` : null
})

function applyPrice() {
  // An emptied field only needs an override if the text asserts a value to
  // suppress; otherwise leaving the param out keeps the URL clean.
  emit('patch', {
    price_min: minParsed.value ? String(minParsed.value) : props.intent?.price_min ? '' : null,
    price_max: maxParsed.value ? String(maxParsed.value) : props.intent?.price_max ? '' : null,
  })
  editing.value = false
}
</script>

<template>
  <div>
    <div class="mx-auto mt-4 flex min-h-8 max-w-[820px] flex-wrap items-center justify-center gap-2" aria-live="polite">
      <TransitionGroup name="chip">
        <span
          v-for="(c, n) in chips"
          :key="c.id"
          :style="{ transitionDelay: `${n * 45}ms` }"
          class="inline-flex min-h-11 items-center rounded-full border border-good/30 bg-good/[.12] ps-3 pe-1 text-[.8rem] text-good"
        >
          <button
            v-if="c.price"
            type="button"
            class="flex min-h-11 items-center gap-1.5 rounded-full focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            :aria-label="`${c.k} قیمت ${c.v}، ویرایش بازه قیمت`"
            :aria-expanded="editing"
            @click="editing ? editing = false : openEditor()"
          >
            <span class="font-mono text-[.66rem] text-ink-3">{{ c.k }}</span>
            <span>{{ c.v }}</span>
            <svg viewBox="0 0 24 24" class="size-3 opacity-60" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path d="M12 20h9M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4L16.5 3.5z" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </button>
          <template v-else>
            <span class="me-1.5 font-mono text-[.66rem] text-ink-3">{{ c.k }}</span>
            <span>{{ c.v }}</span>
          </template>
          <button
            type="button"
            class="ms-0.5 grid size-11 shrink-0 place-items-center rounded-full text-ink-2 transition hover:text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-4px] focus-visible:outline-focus"
            :aria-label="`حذف فیلتر ${c.k} ${c.v}`"
            @click="remove(c)"
          >
            <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
              <path d="M6 6l12 12M18 6L6 18" stroke-linecap="round" />
            </svg>
          </button>
        </span>

        <button
          v-for="r in restorable"
          :key="`restore:${r.id}`"
          type="button"
          class="inline-flex min-h-11 items-center gap-1.5 rounded-full border border-dashed border-white/[.14] px-3.5
                 text-[.76rem] text-ink-3 transition hover:border-white/[.3] hover:text-ink-2
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          @click="restore(r)"
        >
          <svg viewBox="0 0 24 24" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M3 12a9 9 0 109-9 9.75 9.75 0 00-6.74 2.74L3 8" stroke-linecap="round" stroke-linejoin="round" />
            <path d="M3 3v5h5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          بازگرداندن {{ r.label }}
        </button>
      </TransitionGroup>
    </div>

    <form
      v-if="editing"
      class="glass mx-auto mt-3 w-full max-w-[420px] rounded-2xl p-4 text-right"
      aria-label="ویرایش بازه قیمت"
      @submit.prevent="applyPrice"
      @keydown.esc.stop="editing = false"
    >
      <div class="grid grid-cols-2 gap-3">
        <label class="block">
          <span class="mb-1 block text-[.72rem] text-ink-2">حداقل (تومان)</span>
          <input
            ref="minEl"
            v-model="minRaw"
            type="text"
            inputmode="numeric"
            class="mono-nums min-h-11 w-full rounded-lg border border-white/[.12] bg-surface-2 px-3 text-[.85rem] text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >
        </label>
        <label class="block">
          <span class="mb-1 block text-[.72rem] text-ink-2">حداکثر (تومان)</span>
          <input
            v-model="maxRaw"
            type="text"
            inputmode="numeric"
            class="mono-nums min-h-11 w-full rounded-lg border border-white/[.12] bg-surface-2 px-3 text-[.85rem] text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >
        </label>
      </div>

      <!-- The shorthand is read back before it is applied: type «۵۰۰» and this
           line says ۵۰۰ میلیون تومان, so a wrong scale never filters silently. -->
      <p class="mt-2 text-[.74rem] text-ink-3" aria-live="polite">
        {{ previewText ?? 'عدد به تومان؛ «۵۰۰» یعنی ۵۰۰ میلیون تومان' }}
      </p>

      <div class="mt-3 flex justify-end gap-2">
        <button
          type="button"
          class="min-h-11 rounded-lg px-4 text-[.8rem] text-ink-2 transition hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          @click="editing = false"
        >بستن</button>
        <button
          type="submit"
          class="min-h-11 rounded-lg border border-good/40 bg-good/[.14] px-5 text-[.8rem] font-bold text-good transition hover:bg-good/25
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >اعمال بازه</button>
      </div>
    </form>
  </div>
</template>

<style scoped>
/* Named properties, not `all`: `all` also watches layout properties that
   change for reasons unrelated to this transition, and animates those too. */
.chip-enter-active, .chip-leave-active {
  transition: opacity .38s var(--ease-out-quint), transform .38s var(--ease-out-quint);
}
.chip-enter-from, .chip-leave-to { opacity: 0; transform: translateY(6px) scale(.94); }
</style>
