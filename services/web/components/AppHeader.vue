<script setup lang="ts">
const { data: stats } = useStats()
const f = useFormat()
</script>

<template>
  <header
    class="sticky top-0 z-50 border-b border-white/[.07]"
    style="background: rgba(7,8,11,.72); backdrop-filter: blur(14px) saturate(140%)"
  >
    <div class="mx-auto flex h-[62px] max-w-[1080px] items-center gap-3.5 px-5">
      <div class="flex items-center gap-2.5 text-[1.06rem] font-black tracking-tight">
        <span class="ping size-2.5 rounded-full bg-accent" />
        خودروبین
      </div>
      <span class="rounded-full border border-white/[.07] px-2 py-[3px] font-mono text-[.66rem] tracking-wide text-ink-3">
        torob for used cars
      </span>
      <span class="flex-1" />
      <span
        v-if="stats"
        class="hidden truncate font-mono text-[.72rem] text-ink-3 md:inline"
      >
        <b class="font-medium text-good">{{ f.money(stats.stats.indexed) }}</b> آگهی ·
        <b class="font-medium text-good">{{ f.money(stats.stats.specs) }}</b> خودرو ·
        <b class="font-medium text-good">{{ f.money(stats.stats.multi_source_specs) }}</b> چندمنبعی
      </span>
    </div>
  </header>
</template>

<style scoped>
.ping { box-shadow: 0 0 0 4px rgba(255,46,77,.12); animation: ping 2.6s var(--ease-out-quint) infinite; }
@keyframes ping { 0%,100% { box-shadow: 0 0 0 4px rgba(255,46,77,.12) } 50% { box-shadow: 0 0 0 9px transparent } }
</style>
