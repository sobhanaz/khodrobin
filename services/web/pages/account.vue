<script setup lang="ts">
useSeoMeta({ title: 'حساب من - خودروبین', robots: 'noindex' })

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

// The stored mode is an id; the tabs on the results page already own the
// Persian name for each one, so a saved row says the same word the user picked.
const modeLabel = (id: string) => RANKING_MODES.find(m => m.id === id)?.label ?? id

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

// Focus is the lavender token, never the accent: red is this product's verdict
// on a wrong value, and a merely selected control has not been judged yet.
const ring = 'outline-none focus-visible:outline focus-visible:outline-2 ' +
  'focus-visible:outline-offset-2 focus-visible:outline-focus'

// min-h-11 is the 44px target. The text stays small; the hit area does not.
const pill = `inline-flex min-h-11 items-center rounded-full border border-white/[.14] px-4 ` +
  `text-[.8rem] text-ink-2 transition hover:text-ink ${ring}`

const dt = 'mt-3 text-ink-2 first:mt-0 sm:mt-0'
</script>

<template>
  <main class="mx-auto max-w-[760px] px-5 py-14">
    <h1 class="text-[1.6rem] font-black">حساب من</h1>
    <p v-if="user" class="mt-2 font-mono text-[.85rem] text-ink-2" dir="ltr">{{ user.email }}</p>

    <div
      v-if="user && !user.verified"
      class="mt-5 rounded-2xl border border-warn/25 bg-warn/[.12] p-4 text-[.88rem] leading-8 text-warn"
    >
      ایمیلت هنوز تأیید نشده. برای ساختن هشدار قیمت باید تأییدش کنی؛ لینک تأیید در ایمیلت است.
    </div>

    <section class="mt-9">
      <h2 class="mb-3 text-[1.05rem] font-bold">جست‌وجوی جدید</h2>
      <form class="grid gap-4 rounded-2xl border border-white/[.07] bg-surface p-5" @submit.prevent="add">
        <FormField
          id="saved-query"
          v-model="form.query"
          label="متن جست‌وجو"
          placeholder="ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵"
          required
        />
        <FormField
          id="saved-label"
          v-model="form.label"
          label="نام این جست‌وجو"
          hint="برای اینکه بعداً در فهرست پیدایش کنی."
        />

        <div class="grid gap-2 rounded-xl border border-white/[.07] bg-bg-2 p-4">
          <!-- The checkbox arms a real watcher now. It is named after what it
               does rather than after the mail it sends, because the mail only
               arrives when the price actually moves. -->
          <label class="flex min-h-11 items-center gap-3 text-[.9rem]">
            <input
              v-model="form.alert"
              type="checkbox"
              :disabled="!user?.verified"
              aria-describedby="alert-note"
              :class="`size-5 shrink-0 accent-[var(--color-accent)] ${ring}`"
            >
            <span>هشدار تغییر قیمت</span>
          </label>

          <!-- This said the threshold was saved but no mail would be sent,
               which was true and is not any more: the watcher now re-runs the
               search every three hours and mails when the median moves past the
               threshold. The copy has to move with the feature, or the page goes
               on apologising for something that works. -->
          <p id="alert-note" class="text-[.8rem] leading-7 text-ink-2">
            <template v-if="!user?.verified">اول باید ایمیلت را تأیید کنی. </template>
            هر ۳ ساعت جست‌وجویت دوباره اجرا می‌شود و اگر میانه‌ی قیمت بیشتر از این آستانه
            جابه‌جا شود، ایمیل می‌گیری.
          </p>

          <div v-if="form.alert" class="mt-1 flex items-center gap-3">
            <label for="alert-pct" class="shrink-0 text-[.85rem] text-ink-2">آستانه</label>
            <input
              id="alert-pct"
              v-model.number="form.pct"
              type="range"
              min="1"
              max="50"
              :class="`min-h-11 flex-1 accent-[var(--color-accent)] ${ring}`"
            >
            <span class="shrink-0 text-[.85rem] text-ink-2">{{ f.fa(form.pct) }}٪</span>
          </div>
        </div>

        <button
          type="submit"
          :class="`min-h-11 justify-self-start rounded-xl bg-accent px-5 font-bold text-white transition hover:brightness-110 ${ring}`"
        >
          ذخیره
        </button>
      </form>
    </section>

    <p
      v-if="error"
      role="alert"
      class="mt-4 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] text-accent"
    >
      {{ error }}
    </p>

    <section class="mt-10">
      <h2 class="mb-3 text-[1.05rem] font-bold">جست‌وجوهای ذخیره‌شده</h2>

      <p v-if="loading" class="text-ink-2">در حال بارگذاری…</p>
      <p v-else-if="!searches.length" class="rounded-2xl border border-white/[.07] bg-surface p-6 text-center text-ink-2">
        هنوز چیزی ذخیره نکرده‌ای.
      </p>

      <div v-else class="grid gap-3">
        <!-- Every value carries its own label. The old card stacked the name
             above the query in near-identical type, so a row reading
             «۲۰۷s» over «۲۰۷» told nobody which half was which, or what the
             row would do when run. -->
        <article
          v-for="s in searches"
          :key="s.id"
          class="rounded-2xl border border-white/[.07] bg-surface p-5"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <h3
              class="min-w-0 break-words text-[.95rem]"
              :class="s.label ? 'font-bold' : 'font-normal text-ink-2'"
            >{{ s.label || 'جست‌وجوی بی‌نام' }}</h3>

            <div class="flex shrink-0 gap-2">
              <NuxtLink :to="`/?q=${encodeURIComponent(s.query)}`" :class="pill">اجرا</NuxtLink>
              <button type="button" :class="`${pill} hover:text-accent`" @click="remove(s.id)">حذف</button>
            </div>
          </div>

          <!-- One column under sm, two above it. Stacked, the row gap alone put
               a label as far from its own value as from the next pair, so the
               space goes above each label instead and the pair reads as one. -->
          <dl class="mt-3 grid gap-x-5 gap-y-1 text-[.85rem] leading-7 sm:grid-cols-[auto_1fr] sm:gap-y-2">
            <dt :class="dt">جست‌وجو</dt>
            <dd class="min-w-0 break-words">{{ s.query }}</dd>

            <dt :class="dt">مرتب‌سازی</dt>
            <dd>{{ modeLabel(s.mode) }}</dd>

            <dt :class="dt">هشدار قیمت</dt>
            <dd :class="s.alert_pct ? 'text-warn' : ''">
              <template v-if="s.alert_pct">
                از {{ f.fa(s.alert_pct) }}٪ (ذخیره شده، هنوز ارسال نمی‌شود)
              </template>
              <template v-else>ندارد</template>
            </dd>

            <template v-if="s.last_median">
              <dt :class="dt">میانه‌ی آخر</dt>
              <!-- dir on a <span> around the number, not on the <dd>. On the dd
                   it also flips the cell's alignment, and the price drifted to
                   the far side of the card away from the label it belongs to. -->
              <dd><span class="font-mono" dir="ltr">{{ f.money(s.last_median) }}</span></dd>
            </template>
          </dl>
        </article>
      </div>
    </section>
  </main>
</template>
