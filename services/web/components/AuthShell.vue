<script setup lang="ts">
defineProps<{ title: string, subtitle?: string }>()
</script>

<template>
  <main class="mx-auto flex min-h-[70vh] max-w-[420px] flex-col justify-center px-5 py-16">
    <h1 class="text-[1.6rem] font-black tracking-tight">{{ title }}</h1>
    <p v-if="subtitle" class="mt-2 text-[.9rem] leading-8 text-ink-2">{{ subtitle }}</p>
    <div class="mt-7"><slot /></div>
  </main>
</template>
