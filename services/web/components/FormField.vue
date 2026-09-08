<script setup lang="ts">
const props = withDefaults(defineProps<{
  id: string
  label: string
  type?: string
  autocomplete?: string
  placeholder?: string
  hint?: string
  error?: string | null
  required?: boolean
  autofocus?: boolean
  dir?: 'rtl' | 'ltr'
}>(), { type: 'text', required: false, autofocus: false })

const model = defineModel<string>({ required: true })
const input = ref<HTMLInputElement | null>(null)
const slots = useSlots()

// An adornment is absolutely positioned over the field, so the field has to
// give up the space or the text runs underneath it. The password toggle sat on
// top of the first characters of every password typed into it, because the
// input's padding was a fixed px-4 that knew nothing about the button.
const adorned = computed(() => Boolean(slots.adornment))

// Autofocus the first field so a returning user can start typing immediately,
// but never on touch: focusing there opens the keyboard over the page and hides
// the context the person came to read.
onMounted(() => {
  if (!props.autofocus) return
  if (window.matchMedia('(hover: hover) and (pointer: fine)').matches) input.value?.focus()
})

// An email address, a phone number or a one-time code is not prose. Underlining
// it in red teaches nothing, and some platforms send the text away to check it.
const spellcheckable = computed(
  () => !['email', 'tel', 'url', 'password'].includes(props.type) && props.autocomplete !== 'one-time-code')

const describedBy = computed(() => {
  const ids: string[] = []
  if (props.hint) ids.push(`${props.id}-hint`)
  if (props.error) ids.push(`${props.id}-error`)
  return ids.length ? ids.join(' ') : undefined
})
</script>

<template>
  <div>
    <!-- A real label, not a placeholder. Placeholders vanish the moment someone
         starts typing, which is exactly when they might need to re-read it. -->
    <label :for="id" class="mb-1.5 block text-[.85rem] text-ink-2">
      {{ label }}
      <span v-if="!required" class="text-ink-3">(اختیاری)</span>
    </label>

    <div class="relative">
      <input
        :id="id"
        ref="input"
        v-model="model"
        :type="type"
        :dir="dir"
        :required="required"
        :autocomplete="autocomplete"
        :placeholder="placeholder"
        :spellcheck="spellcheckable"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="describedBy"
        class="w-full rounded-xl border bg-surface px-4 py-3 text-ink outline-none transition
               placeholder:text-ink-3
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
        :class="[
          // Red is the error colour, so it cannot also be the focus colour.
          // Before this, an autofocused email field arrived outlined in red on a
          // page nobody had typed into yet, and a genuinely invalid field looked
          // exactly like a merely selected one.
          error
            ? 'border-accent/60 focus:shadow-[0_0_0_4px_rgba(255,46,77,.14)] focus-visible:outline-accent'
            : 'border-white/[.12] focus:border-focus/60 focus:shadow-[0_0_0_4px_rgba(107,116,230,.16)] focus-visible:outline-focus',
          // Physical, not logical: the adornment is pinned to the physical left
          // edge, and this field renders LTR content inside an RTL page — a
          // logical ps-* would follow the text direction and reserve the gap on
          // the wrong side.
          adorned && 'pl-12',
        ]"
      >
      <slot name="adornment" :focus="() => input?.focus()" />
    </div>

    <p v-if="hint && !error" :id="`${id}-hint`" class="mt-1.5 text-[.76rem] leading-6 text-ink-3">
      {{ hint }}
    </p>

    <!-- Icon and text, not colour alone: a red border says nothing to a
         colour-blind reader or a screen reader. role="alert" announces it. -->
    <p
      v-if="error"
      :id="`${id}-error`"
      role="alert"
      class="mt-1.5 flex items-start gap-1.5 text-[.8rem] leading-6 text-accent"
    >
      <span aria-hidden="true">⚠</span><span>{{ error }}</span>
    </p>
  </div>
</template>
