<script setup lang="ts">
withDefaults(defineProps<{
  id: string
  label: string
  autocomplete: 'current-password' | 'new-password'
  hint?: string
  error?: string | null
}>(), {})

const model = defineModel<string>({ required: true })

// A visibility toggle instead of a confirm-password field. Retyping a password
// you cannot see causes more mistakes than it prevents; being able to read what
// you typed prevents them at the source.
const revealed = ref(false)
</script>

<template>
  <FormField
    :id="id"
    v-model="model"
    :label="label"
    :type="revealed ? 'text' : 'password'"
    :autocomplete="autocomplete"
    :hint="hint"
    :error="error"
    required
    dir="ltr"
  >
    <template #adornment="{ focus }">
      <!-- Toggling puts focus on the button, which drops the caret out of the
           field someone is mid-way through typing into. Handing it straight
           back means reading what you typed costs nothing. -->
      <button
        type="button"
        tabindex="-1"
        class="absolute inset-y-0 left-0 flex w-12 items-center justify-center rounded-r-xl text-ink-3
               transition hover:text-ink-2
               focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent"
        :aria-label="revealed ? 'پنهان کردن رمز' : 'نمایش رمز'"
        :aria-pressed="revealed"
        @click="revealed = !revealed; focus()"
      >
        <svg v-if="!revealed" viewBox="0 0 24 24" class="size-[18px]" fill="none" stroke="currentColor" stroke-width="1.8">
          <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" stroke-linecap="round" stroke-linejoin="round" />
          <circle cx="12" cy="12" r="3" />
        </svg>
        <svg v-else viewBox="0 0 24 24" class="size-[18px]" fill="none" stroke="currentColor" stroke-width="1.8">
          <path d="M3 3l18 18M10.6 10.6a3 3 0 0 0 4.2 4.2M9.9 5.2A9.6 9.6 0 0 1 12 5c6.5 0 10 7 10 7a17 17 0 0 1-3.2 4.1M6.3 6.4A17 17 0 0 0 2 12s3.5 7 10 7c1 0 1.9-.1 2.7-.4"
                stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
    </template>
  </FormField>
</template>
