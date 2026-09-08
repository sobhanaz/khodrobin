<script setup lang="ts">
const model = defineModel<string>({ required: true })
const emit = defineEmits<{ submit: [] }>()
</script>

<template>
  <!-- Stacked on a phone, side by side from `sm` up.
       Inline at 390px the button took about 100px of a 350px form, and the
       placeholder rendered «مثلاً: ارزون‌ترین پژو ۲۰۶ بالای» and stopped
       mid-sentence. The primary interaction of the whole product was unusable
       on the device most of its users are holding.
       The placeholder is short now because the example chips below already do
       the job of showing how complex a query is allowed to be; a truncated
       example teaches nothing and looks broken. -->
  <form
    class="flex flex-col gap-2 rounded-2xl border border-white/[.12] bg-surface p-1.5
           transition-all duration-250
           focus-within:-translate-y-px focus-within:border-focus/55
           focus-within:shadow-[0_0_0_4px_rgba(107,116,230,.16),0_16px_44px_-20px_rgba(0,0,0,.9)]
           sm:flex-row sm:items-center sm:gap-3"
    role="search"
    @submit.prevent="emit('submit')"
  >
    <input
      v-model="model"
      type="search"
      autocomplete="off"
      enterkeyhint="search"
      aria-label="جست‌وجوی خودرو"
      placeholder="مثلاً: پژو ۲۰۶ زیر ۵۰۰ میلیون"
      class="min-w-0 flex-1 bg-transparent px-3.5 py-3 text-base text-ink outline-none placeholder:text-ink-3"
    >
    <button
      type="submit"
      class="shrink-0 rounded-[10px] bg-accent px-5 py-3 font-bold text-white transition
             active:scale-[.97] hover:brightness-110
             focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
    >
      جست‌وجو
    </button>
  </form>
</template>
