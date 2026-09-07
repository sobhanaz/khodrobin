<script setup lang="ts">
const props = defineProps<{ specKey: string }>()

interface ExplainResponse {
  text: string
  source: 'llm' | 'fallback' | 'unavailable' | 'budget_exceeded'
  cached?: boolean
  precomputed?: boolean
  rejected_numbers?: number[]
  usage?: { provider: string, model: string, latency_ms: number, cost_usd: number } | null
  total_ms?: number
}

const data = ref<ExplainResponse | null>(null)
const pending = ref(false)

/**
 * Fetched on demand rather than with the search results.
 *
 * Search answers in about a millisecond and a model answers in seconds, so
 * bundling them would make every result page wait for text most visitors never
 * open — and would pay for explanations nobody reads.
 */
async function load() {
  if (data.value || pending.value) return
  pending.value = true
  try {
    data.value = await $fetch<ExplainResponse>(apiUrl(`/api/v1/explain/${props.specKey}`))
  } catch {
    data.value = { text: '', source: 'unavailable' }
  } finally {
    pending.value = false
  }
}
onMounted(load)

// Being honest about provenance is the point: a templated sentence is true but
// not written by a model, and the reader deserves to know which they are seeing.
const provenance = computed(() => {
  switch (data.value?.source) {
    case 'llm': return { label: 'نوشته‌ی مدل', tone: 'text-good border-good/30 bg-good/[.12]' }
    case 'fallback': return { label: 'متن قالبی — پاسخ مدل رد شد', tone: 'text-warn border-warn/25 bg-warn/[.12]' }
    case 'budget_exceeded': return { label: 'سقف هزینه‌ی روزانه', tone: 'text-warn border-warn/25 bg-warn/[.12]' }
    default: return { label: 'مدل در دسترس نیست', tone: 'text-ink-3 border-white/[.12] bg-surface-2' }
  }
})
</script>

<template>
  <div>
    <p v-if="pending" class="flex items-center gap-2 text-[.86rem] text-ink-3">
      <span class="inline-block size-3 animate-spin rounded-full border-2 border-ink-3/30 border-t-ink-3" aria-hidden="true" />
      در حال نوشتن توضیح… (برای خودروهای کم‌بازدید چند ثانیه طول می‌کشد)
    </p>

    <template v-else-if="data?.text">
      <p class="text-[.92rem] leading-8 text-ink">{{ data.text }}</p>

      <div class="mt-3 flex flex-wrap items-center gap-2 text-[.68rem]">
        <span class="rounded-full border px-2.5 py-0.5" :class="provenance.tone">{{ provenance.label }}</span>
        <span
          v-if="data.precomputed"
          class="rounded-full border border-white/[.12] bg-surface-2 px-2.5 py-0.5 text-ink-3"
        >از پیش آماده‌شده</span>
        <span
          v-else-if="data.cached"
          class="rounded-full border border-white/[.12] bg-surface-2 px-2.5 py-0.5 text-ink-3"
        >از کش</span>
        <span v-if="data.usage" class="font-mono text-ink-3" dir="ltr">
          {{ data.usage.model }} · {{ Math.round(data.usage.latency_ms) }}ms · ${{ data.usage.cost_usd.toFixed(5) }}
        </span>
      </div>

      <!-- When the guard fires, say so and show what was rejected. A product
           whose claim is trustworthy numbers should demonstrate the mechanism,
           not hide it. -->
      <p
        v-if="data.rejected_numbers?.length"
        class="mt-2 rounded-lg border border-warn/20 bg-warn/[.12] px-3 py-2 text-[.78rem] text-warn"
      >
        ⚑ مدل عددی گفت که در داده‌ی ورودی نبود
        (<span class="font-mono" dir="ltr">{{ data.rejected_numbers.join(', ') }}</span>)،
        پس پاسخش رد شد و متن قالبی جایگزین شد.
      </p>
    </template>

    <p v-else class="text-[.86rem] text-ink-3">توضیحی در دسترس نیست.</p>
  </div>
</template>
