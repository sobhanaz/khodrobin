<script setup lang="ts">
const model = defineModel<string>({ required: true })
const tabs = RANKING_MODES

const wrap = ref<HTMLElement | null>(null)
const pill = reactive({ width: 0, right: 0 })

/**
 * The indicator is positioned from the *right* edge because the document is
 * RTL: `offsetLeft` is still measured from the left, so the right inset is
 * container width minus left minus width.
 */
function sync() {
  const el = wrap.value?.querySelector<HTMLElement>('[aria-selected="true"]')
  if (!el || !wrap.value) return
  pill.width = el.offsetWidth
  pill.right = wrap.value.offsetWidth - el.offsetLeft - el.offsetWidth
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
  <div
    ref="wrap"
    role="tablist"
    class="glass relative mx-auto mt-6 flex w-max max-w-full gap-1 overflow-x-auto rounded-full p-1.5"
  >
    <span
      class="absolute inset-y-1.5 z-0 rounded-full bg-accent transition-all duration-[380ms]"
      :style="{ width: `${pill.width}px`, right: `${pill.right}px` }"
      :class="{ 'opacity-0': !pill.width }"
    />
    <button
      v-for="t in tabs"
      :key="t.id"
      type="button"
      role="tab"
      :aria-selected="model === t.id"
      class="relative z-10 whitespace-nowrap rounded-full px-4.5 py-2 text-[.84rem] transition-colors duration-200"
      :class="model === t.id ? 'text-white' : 'text-ink-3 hover:text-ink-2'"
      @click="model = t.id"
    >
      {{ t.label }}
    </button>
  </div>
</template>
