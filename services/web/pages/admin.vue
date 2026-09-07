<script setup lang="ts">
useSeoMeta({ title: 'مدیریت — خودروبین', robots: 'noindex, nofollow' })

const { user, call, init, isLoggedIn } = useAuth()
const f = useFormat()

interface AdminSummary {
  counts: {
    users: number
    verified: number
    saved_searches: number
    active_sessions: number
    unread_contact: number
  }
  mail: { configured: boolean }
}

const summary = ref<AdminSummary | null>(null)
const { data: stats, refresh: refreshStats } = useStats()
const denied = ref(false)
const loading = ref(true)

onMounted(async () => {
  await init()
  if (!isLoggedIn.value) return navigateTo('/login?next=/admin')
  try {
    summary.value = await call<AdminSummary>('/api/auth/admin/summary')
  } catch {
    // The API answers 404 to a non-admin rather than 403, so this page cannot
    // tell "no such route" from "not allowed" — and should not try to.
    denied.value = true
  } finally {
    loading.value = false
  }
})

/** Index freshness in plain words. A number of seconds is not an answer. */
const freshness = computed(() => {
  const built = stats.value?.built_at
  if (!built) return null
  const mins = (Date.now() - new Date(built).getTime()) / 60000
  const stale = mins > 360 // two missed crawl cycles
  if (mins < 60) return { text: `${f.fa(Math.round(mins))} دقیقه پیش`, stale }
  return { text: `${f.fa(Math.round(mins / 60))} ساعت پیش`, stale }
})

const dedupRate = computed(() => {
  const s = stats.value?.stats as any
  if (!s?.listings_captured) return null
  return Math.round((s.duplicates_collapsed / s.listings_captured) * 100)
})
</script>

<template>
  <main class="mx-auto max-w-[980px] px-5 py-14">
    <h1 class="text-[1.6rem] font-black tracking-tight">مدیریت</h1>

    <p v-if="loading" class="mt-6 text-ink-3">در حال بارگذاری…</p>

    <div v-else-if="denied" class="mt-6 rounded-2xl border border-white/[.07] bg-surface p-8 text-center">
      <div class="font-bold">دسترسی نداری</div>
      <p class="mt-2 text-[.88rem] text-ink-2">این صفحه فقط برای حساب مدیر است.</p>
      <NuxtLink to="/" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">بازگشت</NuxtLink>
    </div>

    <template v-else>
      <p class="mt-2 font-mono text-[.85rem] text-ink-3" dir="ltr">{{ user?.email }}</p>

      <!-- Pipeline health first. These are the numbers that say whether the
           product is telling the truth right now; account counts are trivia by
           comparison. -->
      <section class="mt-8">
        <h2 class="mb-3 text-[.8rem] font-mono uppercase tracking-wider text-ink-3">سلامت داده</h2>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div
            class="rounded-2xl border p-5"
            :class="freshness?.stale
              ? 'border-warn/30 bg-warn/[.12]'
              : 'border-white/[.07] bg-surface'"
          >
            <div class="text-[.75rem] text-ink-3">آخرین بازسازی ایندکس</div>
            <div class="mt-1 text-[1.15rem] font-bold" :class="freshness?.stale ? 'text-warn' : 'text-ink'">
              {{ freshness?.text ?? '—' }}
            </div>
            <div class="mt-1 text-[.72rem] text-ink-3">
              {{ freshness?.stale ? 'دو چرخه‌ی خزش جا افتاده' : 'در بازه‌ی عادی' }}
            </div>
          </div>

          <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-3">آگهی‌های یکتا</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold" dir="ltr">
              {{ stats ? f.money(stats.stats.indexed) : '—' }}
            </div>
            <div v-if="dedupRate !== null" class="mt-1 text-[.72rem] text-ink-3">
              {{ f.fa(dedupRate) }}٪ تکراری حذف شد
            </div>
          </div>

          <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-3">خودروها</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold" dir="ltr">
              {{ stats ? f.money(stats.stats.specs) : '—' }}
            </div>
            <div class="mt-1 text-[.72rem] text-ink-3">
              {{ stats ? f.fa(stats.stats.multi_source_specs) : '—' }} چندمنبعی
            </div>
          </div>

          <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-3">آگهی‌های پرچم‌خورده</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold text-warn" dir="ltr">
              {{ stats ? f.money(stats.stats.flagged_offers) : '—' }}
            </div>
            <div class="mt-1 text-[.72rem] text-ink-3">تناقض داخلی داده</div>
          </div>
        </div>

        <div v-if="stats" class="mt-3 flex flex-wrap gap-2">
          <span
            v-for="s in stats.stats.sources"
            :key="s"
            class="rounded-full border border-good/30 bg-good/[.12] px-3 py-1 font-mono text-[.72rem] text-good"
          >{{ s }}</span>
          <button type="button" class="rounded-full border border-white/[.12] px-3 py-1 text-[.72rem] text-ink-3 hover:text-ink"
                  @click="refreshStats()">به‌روزرسانی</button>
        </div>
      </section>

      <section class="mt-10">
        <h2 class="mb-3 text-[.8rem] font-mono uppercase tracking-wider text-ink-3">حساب‌ها</h2>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div v-for="c in [
            { l: 'کاربران', v: summary?.counts.users },
            { l: 'تأییدشده', v: summary?.counts.verified },
            { l: 'جست‌وجوی ذخیره‌شده', v: summary?.counts.saved_searches },
            { l: 'نشست فعال', v: summary?.counts.active_sessions },
            { l: 'پیام خوانده‌نشده', v: summary?.counts.unread_contact },
          ]" :key="c.l" class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-3">{{ c.l }}</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold" dir="ltr">{{ c.v ?? '—' }}</div>
          </div>
        </div>
      </section>

      <section class="mt-10">
        <h2 class="mb-3 text-[.8rem] font-mono uppercase tracking-wider text-ink-3">سرویس‌ها</h2>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="flex items-center justify-between rounded-2xl border border-white/[.07] bg-surface p-5">
            <span class="text-[.9rem]">ارسال ایمیل</span>
            <span
              class="rounded-full px-3 py-1 text-[.74rem]"
              :class="summary?.mail.configured
                ? 'bg-good/[.12] text-good border border-good/30'
                : 'bg-warn/[.12] text-warn border border-warn/25'"
            >{{ summary?.mail.configured ? 'پیکربندی شده' : 'پیکربندی نشده' }}</span>
          </div>
          <a
            href="/readyz" target="_blank" rel="noopener"
            class="flex items-center justify-between rounded-2xl border border-white/[.07] bg-surface p-5 hover:border-white/[.12]"
          >
            <span class="text-[.9rem]">بررسی آمادگی سرویس جست‌وجو</span>
            <span class="font-mono text-[.74rem] text-ink-3" dir="ltr">/readyz ↗</span>
          </a>
        </div>
      </section>
    </template>
  </main>
</template>
