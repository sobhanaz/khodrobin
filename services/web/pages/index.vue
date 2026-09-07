<script setup lang="ts">
const { query, mode, data, pending, failed, run } = useSearch()
const f = useFormat()

// Server-rendered first paint: the page arrives with results, not a spinner.
await useAsyncData('initial', () => run())
</script>

<template>
  <main class="mx-auto max-w-[1080px] px-5">
    <section class="py-16 text-center">
      <span class="inline-flex items-center gap-2 rounded-full border border-accent/30 bg-accent/[.12] px-3 py-1 font-mono text-[.7rem] tracking-wider text-accent">
        crawl → normalize → rank → explain
      </span>
      <h1 class="mx-auto mt-5 text-[clamp(2rem,5.5vw,3.4rem)] font-black leading-[1.22] tracking-tight">
        یک خودرو، <em class="not-italic text-accent">همه‌ی آگهی‌ها</em>،<br>یک قیمت واقعی.
      </h1>
      <p class="mx-auto mt-3 max-w-[52ch] text-[1.02rem] text-ink-2">
        آگهی‌های دیوار، باما، همراه‌مکانیک و خودرو۴۵ را کنار هم می‌گذاریم، واحدها و مدل‌ها را یکسان می‌کنیم، و میانه‌ی واقعی بازار را نشان می‌دهیم.
      </p>

      <div class="mx-auto mt-8 max-w-[720px]">
        <SearchBox v-model="query" @submit="run()" />

        <div class="mt-3.5 flex flex-wrap justify-center gap-2">
          <button
            v-for="ex in EXAMPLE_QUERIES"
            :key="ex"
            type="button"
            class="glass rounded-full px-3.5 py-1.5 text-[.8rem] text-ink-2 transition hover:-translate-y-0.5 hover:text-ink"
            @click="run(ex)"
          >{{ ex }}</button>
        </div>

        <IntentChips :intent="data?.result.intent ?? null" />
      </div>

      <ModeTabs v-model="mode" />
    </section>

    <section>
      <div class="mb-3.5 mt-8 flex flex-wrap items-center justify-between gap-3">
        <p class="text-[.88rem] text-ink-2">
          <template v-if="data?.result.total">
            <b class="text-ink">{{ f.fa(data.result.total) }}</b> خودرو پیدا شد
          </template>
          <template v-else-if="!pending">چیزی پیدا نشد</template>
        </p>
        <p v-if="data" class="font-mono text-[.7rem] text-ink-3" dir="ltr">
          parse {{ data.timing.parse_ms.toFixed(2) }}ms · rank {{ data.timing.rank_ms.toFixed(2) }}ms · {{ data.result.mode }}
        </p>
      </div>

      <div class="grid gap-3">
        <template v-if="pending && !data">
          <div v-for="i in 3" :key="i" class="skeleton h-[104px] rounded-2xl" />
        </template>
        <p v-else-if="failed" class="py-14 text-center text-ink-3">ارتباط با سرور برقرار نشد.</p>
        <p v-else-if="!data?.result.specs.length" class="py-14 text-center text-ink-3">
          برای این جست‌وجو آگهی‌ای نداریم. یکی از نمونه‌های بالا را امتحان کن.
        </p>
        <SpecCard
          v-for="(s, i) in data?.result.specs ?? []"
          :key="s.key"
          :spec="s"
          :index="i"
        />
      </div>
    </section>

  </main>
</template>

<style scoped>
.skeleton {
  background: linear-gradient(90deg, var(--color-surface), var(--color-surface-2), var(--color-surface));
  background-size: 200% 100%;
  animation: sheen 1.3s linear infinite;
}
@keyframes sheen { to { background-position: -200% 0 } }
</style>
