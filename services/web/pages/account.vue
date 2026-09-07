<script setup lang="ts">
useSeoMeta({ title: 'حساب من — خودروبین', robots: 'noindex' })

const { user, call, init, isLoggedIn } = useAuth()
const f = useFormat()

interface SavedSearch {
  id: number
  label: string | null
  query: string
  mode: string
  alert_pct: number | null
  created_at: string
  last_median: number | null
}

const searches = ref<SavedSearch[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const form = reactive({ query: '', label: '', alert: false, pct: 5 })

async function load() {
  try {
    const res = await call<{ searches: SavedSearch[] }>('/api/auth/searches')
    searches.value = res.searches ?? []
  } catch (e: any) {
    error.value = e?.data?.error ?? 'بارگذاری نشد.'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await init()
  if (!isLoggedIn.value) return navigateTo('/login?next=/account')
  await load()
})

async function add() {
  if (!form.query.trim()) return
  error.value = null
  try {
    await call('/api/auth/searches', {
      method: 'POST',
      body: {
        query: form.query.trim(),
        label: form.label.trim() || undefined,
        alert_pct: form.alert ? form.pct : undefined,
      },
    })
    form.query = ''; form.label = ''; form.alert = false
    await load()
  } catch (e: any) {
    error.value = e?.data?.error ?? 'ذخیره نشد.'
  }
}

async function remove(id: number) {
  try {
    await call(`/api/auth/searches/${id}`, { method: 'DELETE' })
    searches.value = searches.value.filter(s => s.id !== id)
  } catch (e: any) {
    error.value = e?.data?.error ?? 'حذف نشد.'
  }
}

const field = 'w-full rounded-xl border border-white/[.12] bg-surface px-4 py-3 text-ink outline-none ' +
  'transition placeholder:text-ink-3 focus:border-accent/55 focus:shadow-[0_0_0_4px_rgba(255,46,77,.12)]'
</script>

<template>
  <main class="mx-auto max-w-[760px] px-5 py-14">
    <h1 class="text-[1.6rem] font-black tracking-tight">حساب من</h1>
    <p v-if="user" class="mt-2 font-mono text-[.85rem] text-ink-3" dir="ltr">{{ user.email }}</p>

    <div
      v-if="user && !user.verified"
      class="mt-5 rounded-2xl border border-warn/25 bg-warn/[.12] p-4 text-[.88rem] leading-8 text-warn"
    >
      ایمیلت هنوز تأیید نشده. برای ساختن هشدار قیمت باید تأییدش کنی — لینک تأیید در ایمیلت است.
    </div>

    <section class="mt-9">
      <h2 class="mb-3 text-[1.05rem] font-bold">جست‌وجوی جدید</h2>
      <form class="grid gap-3 rounded-2xl border border-white/[.07] bg-surface p-5" @submit.prevent="add">
        <input v-model="form.query" :class="field" placeholder="مثلاً: ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵" required>
        <input v-model="form.label" :class="field" placeholder="یک نام برایش بگذار (اختیاری)">

        <label class="flex items-center gap-2.5 text-[.88rem] text-ink-2">
          <input v-model="form.alert" type="checkbox" :disabled="!user?.verified" class="size-4 accent-[var(--color-accent)]">
          وقتی میانه‌ی قیمت تغییر کرد، ایمیل بزن
        </label>

        <div v-if="form.alert" class="flex items-center gap-3">
          <input v-model.number="form.pct" type="range" min="1" max="50" class="flex-1 accent-[var(--color-accent)]">
          <span class="font-mono text-[.85rem] text-ink-2" dir="ltr">{{ form.pct }}%</span>
        </div>

        <button type="submit" class="justify-self-start rounded-xl bg-accent px-5 py-2.5 font-bold text-white hover:brightness-110">
          ذخیره
        </button>
      </form>
    </section>

    <p v-if="error" class="mt-4 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] text-accent">
      {{ error }}
    </p>

    <section class="mt-10">
      <h2 class="mb-3 text-[1.05rem] font-bold">جست‌وجوهای ذخیره‌شده</h2>

      <p v-if="loading" class="text-ink-3">در حال بارگذاری…</p>
      <p v-else-if="!searches.length" class="rounded-2xl border border-white/[.07] bg-surface p-6 text-center text-ink-3">
        هنوز چیزی ذخیره نکرده‌ای.
      </p>

      <div v-else class="grid gap-3">
        <article
          v-for="s in searches"
          :key="s.id"
          class="flex items-start justify-between gap-4 rounded-2xl border border-white/[.07] bg-surface p-5"
        >
          <div class="min-w-0">
            <div class="font-bold">{{ s.label || s.query }}</div>
            <div v-if="s.label" class="mt-1 text-[.85rem] text-ink-3">{{ s.query }}</div>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <span
                v-if="s.alert_pct"
                class="rounded-md border border-good/30 bg-good/[.12] px-2 py-0.5 text-[.74rem] text-good"
              >هشدار از {{ f.fa(s.alert_pct) }}٪</span>
              <span
                v-if="s.last_median"
                class="rounded-md border border-white/[.07] bg-surface-2 px-2 py-0.5 font-mono text-[.74rem] text-ink-2"
                dir="ltr"
              >{{ f.money(s.last_median) }}</span>
            </div>
          </div>
          <div class="flex shrink-0 gap-2">
            <NuxtLink :to="`/?q=${encodeURIComponent(s.query)}`"
              class="rounded-full border border-white/[.12] px-3 py-1.5 text-[.8rem] text-ink-2 hover:text-ink">
              اجرا
            </NuxtLink>
            <button type="button" class="rounded-full border border-white/[.12] px-3 py-1.5 text-[.8rem] text-ink-3 hover:text-accent"
                    @click="remove(s.id)">حذف</button>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>
