<script setup lang="ts">
const model = defineModel<string>({ required: true })
const emit = defineEmits<{ submit: [] }>()
</script>

<template>
  <form
    class="flex items-center gap-3 rounded-2xl border border-white/[.12] bg-surface p-1.5
           transition-all duration-250 focus-within:-translate-y-px
           focus-within:border-accent/55 focus-within:shadow-[0_0_0_4px_rgba(255,46,77,.12),0_16px_44px_-20px_rgba(0,0,0,.9)]"
    role="search"
    @submit.prevent="emit('submit')"
  >
    <input
      v-model="model"
      type="search"
      autocomplete="off"
      aria-label="جست‌وجوی خودرو"
      placeholder="مثلاً: ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵ زیر ۵۰۰ میلیون"
      class="min-w-0 flex-1 bg-transparent px-3.5 py-3 text-base text-ink outline-none placeholder:text-ink-3"
    >
    <button
      type="submit"
      class="shrink-0 rounded-[10px] bg-accent px-5 py-3 font-bold text-white
             transition active:scale-[.97] hover:brightness-110"
    >
      جست‌وجو
    </button>
  </form>
</template>
