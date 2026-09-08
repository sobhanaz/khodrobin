<script setup lang="ts">
/**
 * A one-of-many question as chips.
 *
 * Free text here would produce a thousand spellings of «گوگل» and answer
 * nothing, so the answers are a fixed set and the fixed value is what gets
 * sent. «other» is the honest escape hatch: a list that cannot be wrong is a
 * list people lie to.
 *
 * Native radios under the chips rather than buttons with role="radio": arrow
 * key navigation, group semantics and the checked state all come from the
 * platform, and there is nothing left to get wrong.
 */
const props = defineProps<{
  name: string
  legend: string
  options: { value: string, label: string }[]
  otherLabel: string
  otherFieldLabel: string
  otherPlaceholder?: string
  error?: string | null
}>()

const model = defineModel<string>({ required: true })
const other = defineModel<string>('other', { required: true })

const all = computed(() => [...props.options, { value: 'other', label: props.otherLabel }])
const errorId = computed(() => (props.error ? `${props.name}-error` : undefined))
</script>

<template>
  <fieldset>
    <legend class="mb-2.5 text-[.85rem] text-ink-2">{{ legend }}</legend>

    <div class="flex flex-wrap gap-2">
      <label v-for="o in all" :key="o.value" class="relative cursor-pointer">
        <input
          v-model="model"
          type="radio"
          :name="name"
          :value="o.value"
          :aria-describedby="errorId"
          :aria-invalid="error ? 'true' : undefined"
          class="peer sr-only"
        >
        <!-- 44px tall so it is a real touch target, not an 18px word. -->
        <span
          class="inline-flex min-h-[44px] items-center rounded-full border px-4 text-[.83rem] leading-6 transition
                 border-white/[.12] text-ink-2
                 hover:border-white/25 hover:text-ink
                 peer-checked:border-accent/60 peer-checked:bg-accent/[.12] peer-checked:text-ink
                 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2
                 peer-focus-visible:outline-focus"
        >{{ o.label }}</span>
      </label>
    </div>

    <p
      v-if="error"
      :id="errorId"
      role="alert"
      class="mt-2 flex items-start gap-1.5 text-[.8rem] leading-6 text-accent"
    >
      <span aria-hidden="true">⚠</span><span>{{ error }}</span>
    </p>

    <!-- Revealed only when it is needed, and it reuses the shared field so the
         label, the error wiring and the focus colour are the same ones every
         other input on the page uses. -->
    <FormField
      v-if="model === 'other'"
      :id="`${name}-other`"
      v-model="other"
      :label="otherFieldLabel"
      :placeholder="otherPlaceholder"
      class="mt-3"
      required
    />
  </fieldset>
</template>
