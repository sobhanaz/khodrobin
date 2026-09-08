<script setup lang="ts">
const model = defineModel<string>({ required: true })
const tabs = RANKING_MODES

const wrap = ref<HTMLElement | null>(null)
const pill = reactive({ width: 0, left: 0 })

/**
 * The indicator is placed with `left`, straight from `offsetLeft`.
 *
 * It used to convert that into a `right` inset, reasoning that the document is
 * RTL. But both `offsetLeft` and CSS `left` are physical, measured from the same
 * left edge, so the conversion was never needed and it introduced a bug: it
 * subtracted from `offsetWidth`, the container's VISIBLE width, while
 * `offsetLeft` is measured across the full scrollable content. Once four tabs
 * stopped fitting on a phone the two disagreed, and «بهترین» sat under a pill
 * that had drifted off the end of the row.
 */
function sync() {
  const el = wrap.value?.querySelector<HTMLElement>('[aria-selected="true"]')
  if (!el) return
  pill.width = el.offsetWidth
  pill.left = el.offsetLeft
  // A mode the user cannot see is a mode they do not know exists. When the row
  // scrolls, bring the selected one into view rather than leaving it past an
  // edge with nothing to suggest there is more.
  el.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}

onMounted(() => {
  sync()
  window.addEventListener('resize', sync)
  // Web fonts land after first paint and change tab widths.
  document.fonts?.ready.then(sync)
})
onBeforeUnmount(() => window.removeEventListener('resize', sync))
watch(model, () => nextTick(sync))
</script>

<template>
  <!-- `w-max` lets the row size to its content and scroll when that overflows,
       which is what should happen at 390px. -->
  <div
    ref="wrap"
    role="tablist"
    class="glass relative mx-auto mt-6 flex w-max max-w-full gap-1 overflow-x-auto
           rounded-full p-1.5 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
  >
    <span
      class="absolute inset-y-1.5 z-0 rounded-full bg-accent duration-[380ms]
             [transition-property:left,width]"
      :style="{ width: `${pill.width}px`, left: `${pill.left}px` }"
      :class="{ 'opacity-0': !pill.width }"
    />
    <button
      v-for="t in tabs"
      :key="t.id"
      type="button"
      role="tab"
      :aria-selected="model === t.id"
      class="relative z-10 min-h-[44px] whitespace-nowrap rounded-full px-4.5 py-2 text-[.84rem]
             transition-colors duration-200
             focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus"
      :class="model === t.id ? 'text-white' : 'text-ink-3 hover:text-ink-2'"
      @click="model = t.id"
    >
      {{ t.label }}
    </button>
  </div>
</template>
