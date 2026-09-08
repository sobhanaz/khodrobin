<script setup lang="ts">
import type { Ref } from 'vue'

useSeoMeta({ title: 'مدیریت - خودروبین', robots: 'noindex, nofollow' })

const { user, call, init, isLoggedIn } = useAuth()
const f = useFormat()

interface AdminSummary {
  counts: {
    users: number
    verified: number
    saved_searches: number
    active_sessions: number
    unread_contact: number
    subscribers: number
    subscribers_confirmed: number
    marketing_optin: number
    /**
     * The one list, counted by the one query that also writes the CSV.
     *
     * It used to be added up in the browser as confirmed + opted-in, which is a
     * different set from the one the export selects: two numbers wearing the
     * same label, and whichever the operator read last was the one they sized a
     * campaign against. Optional here only so an older API answers with «-»
     * instead of a figure this page invented.
     */
  }
  mail: { configured: boolean }
}

interface AdminUser {
  id: number
  email: string
  display_name: string | null
  verified: boolean
  is_admin: boolean
  marketing_consent: boolean
  created_at: string
  last_login_at: string | null
}

interface Subscriber {
  id: number
  email: string
  confirmed_at: string | null
  unsubscribed_at: string | null
  source: string | null
  created_at: string
}

const summary = ref<AdminSummary | null>(null)
const { data: stats, refresh: refreshStats } = useStats()
const denied = ref(false)
const loading = ref(true)

/**
 * Both tables page identically. One loader instead of two, because two copies
 * of an offset calculation drift apart the moment someone fixes an off-by-one
 * in only one of them.
 */
function paged<T>(path: string, field: string, limit = 20) {
  const rows = ref([]) as Ref<T[]>
  const total = ref(0)
  const offset = ref(0)
  const busy = ref(true)
  const failed = ref(false)

  async function load() {
    busy.value = true
    try {
      const res = await call<Record<string, any>>(`${path}?limit=${limit}&offset=${offset.value}`)
      rows.value = res[field] ?? []
      total.value = res.total ?? 0
      failed.value = false
    } catch {
      failed.value = true
    } finally {
      busy.value = false
    }
  }

  function page(delta: number) {
    const next = offset.value + delta * limit
    if (next < 0 || next >= total.value) return
    offset.value = next
    void load()
  }

  // reactive(), so the template can say `users.rows`: a ref nested inside a
  // plain object is not unwrapped there, only top-level setup bindings are.
  return reactive({ rows, total, offset, limit, busy, failed, load, page })
}

const users = paged<AdminUser>('/api/auth/admin/users', 'users')
const subs = paged<Subscriber>('/api/auth/admin/subscribers', 'subscribers')

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
  if (!denied.value) await Promise.all([users.load(), subs.load()])
})

const date = new Intl.DateTimeFormat('fa-IR', { dateStyle: 'short' })
const on = (iso: string | null) => (iso ? date.format(new Date(iso)) : '-')

const yn = (v: boolean) =>
  v ? 'border-good/30 bg-good/[.12] text-good' : 'border-white/[.1] bg-white/[.03] text-ink-2'

/**
 * Order matters. Someone who confirmed and then left is «لغو کرده», never
 * «تأییدشده» — read the other way round, a suppression entry looks like a
 * healthy address and gets mailed anyway.
 */
function subState(s: Subscriber) {
  if (s.unsubscribed_at) return { label: 'لغو کرده', cls: 'border-white/[.14] bg-white/[.04] text-ink-2' }
  if (s.confirmed_at) return { label: 'تأییدشده', cls: 'border-good/30 bg-good/[.12] text-good' }
  return { label: 'در انتظار تأیید', cls: 'border-warn/30 bg-warn/[.12] text-warn' }
}

// Every address we hold, against the subset that gave us permission. A
// dashboard that shows only the first number flatters the owner into mailing
// people who never asked.
//
// captured sums two tables and so counts anyone in both of them twice; that is
// stated under the sentence. It is deliberately not subtracted from `mailable`
// to produce a third figure — an over-count minus a clean count is a number
// this page could not stand behind, and it read as precise.
const captured = computed(() => {
  const c = summary.value?.counts
  return c ? c.users + c.subscribers : null
})
// subscribers_confirmed, not a `mailable` key: the API has never emitted one.
// Because the field was declared optional it type-checked, `?? null` rendered
// «-», and the operator's single most important number was a permanent dash
// under a comment asserting it came from the same query that writes the CSV.
// Now it is the count that query actually returns.
const mailable = computed(() => summary.value?.counts.subscribers_confirmed ?? null)

const exporting = ref(false)
const exportError = ref<string | null>(null)
const exported = ref(false)

async function exportCsv() {
  exporting.value = true
  exportError.value = null
  exported.value = false
  try {
    // The CSV route sits behind admin auth like everything else here, so a bare
    // <a href> arrives with no bearer token and 404s exactly like a non-admin.
    const blob = await call<Blob>('/api/auth/admin/subscribers.csv', { responseType: 'blob' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `khodrobin-subscribers-${new Date().toISOString().slice(0, 10)}.csv`
    // In the document and revoked on the next tick, both deliberately: Firefox
    // ignores click() on a detached anchor, and revoking in the same tick has
    // cancelled the save often enough that the operator sees a button that
    // "works" and a downloads folder that stays empty.
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 0)
    exported.value = true
  } catch {
    exportError.value = 'خروجی گرفته نشد.'
  } finally {
    exporting.value = false
  }
}

/** Index freshness in plain words. A number of seconds is not an answer. */
const freshness = computed(() => {
  const built = stats.value?.built_at
  if (!built) return null
  const mins = (Date.now() - new Date(built).getTime()) / 60000
  const stale = mins > 360 // two missed crawl cycles
  if (mins < 60) return { text: `${f.fa(Math.round(mins))} دقیقه پیش`, stale }
  return { text: `${f.fa(Math.round(mins / 60))} ساعت پیش`, stale }
})

// Focus is the lavender token, never the accent. Red is a verdict on a value;
// a control that is merely selected has not been judged.
const ring = 'outline-none focus-visible:outline focus-visible:outline-2 ' +
  'focus-visible:outline-offset-2 focus-visible:outline-focus'

const dedupRate = computed(() => {
  const s = stats.value?.stats as any
  if (!s?.listings_captured) return null
  return Math.round((s.duplicates_collapsed / s.listings_captured) * 100)
})
</script>

<template>
  <main class="mx-auto max-w-[980px] px-5 py-14">
    <h1 class="text-[1.6rem] font-black">مدیریت</h1>

    <p v-if="loading" class="mt-6 text-ink-2">در حال بارگذاری…</p>

    <div v-else-if="denied" class="mt-6 rounded-2xl border border-white/[.07] bg-surface p-8 text-center">
      <div class="font-bold">دسترسی نداری</div>
      <p class="mt-2 text-[.88rem] text-ink-2">این صفحه فقط برای حساب مدیر است.</p>
      <NuxtLink to="/" :class="`mt-4 inline-flex min-h-11 items-center text-[.85rem] text-accent hover:underline ${ring}`">بازگشت</NuxtLink>
    </div>

    <template v-else>
      <p class="mt-2 font-mono text-[.85rem] text-ink-2" dir="ltr">{{ user?.email }}</p>

      <!-- Pipeline health first. These are the numbers that say whether the
           product is telling the truth right now; account counts are trivia by
           comparison. -->
      <section class="mt-8">
        <h2 class="mb-3 text-cap font-bold text-ink-2">سلامت داده</h2>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div
            class="rounded-2xl border p-5"
            :class="freshness?.stale
              ? 'border-warn/30 bg-warn/[.12]'
              : 'border-white/[.07] bg-surface'"
          >
            <div class="text-[.75rem] text-ink-2">آخرین بازسازی ایندکس</div>
            <div class="mt-1 text-[1.15rem] font-bold" :class="freshness?.stale ? 'text-warn' : 'text-ink'">
              {{ freshness?.text ?? '-' }}
            </div>
            <div class="mt-1 text-[.72rem] text-ink-2">
              {{ freshness?.stale ? 'دو چرخه‌ی خزش جا افتاده' : 'در بازه‌ی عادی' }}
            </div>
          </div>

          <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-2">آگهی‌های یکتا</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold" dir="ltr">
              {{ stats ? f.money(stats.stats.indexed) : '-' }}
            </div>
            <div v-if="dedupRate !== null" class="mt-1 text-[.72rem] text-ink-2">
              {{ f.fa(dedupRate) }}٪ تکراری حذف شد
            </div>
          </div>

          <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-2">خودروها</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold" dir="ltr">
              {{ stats ? f.money(stats.stats.specs) : '-' }}
            </div>
            <div class="mt-1 text-[.72rem] text-ink-2">
              {{ stats ? f.fa(stats.stats.multi_source_specs) : '-' }} چندمنبعی
            </div>
          </div>

          <div class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-2">آگهی‌های پرچم‌خورده</div>
            <div class="mt-1 font-mono text-[1.15rem] font-bold text-warn" dir="ltr">
              {{ stats ? f.money(stats.stats.flagged_offers) : '-' }}
            </div>
            <div class="mt-1 text-[.72rem] text-ink-2">تناقض داخلی داده</div>
          </div>
        </div>

        <div v-if="stats" class="mt-3 flex flex-wrap gap-2">
          <span
            v-for="s in stats.stats.sources"
            :key="s"
            class="rounded-full border border-good/30 bg-good/[.12] px-3 py-1 font-mono text-[.72rem] text-good"
          >{{ s }}</span>
          <button
            type="button"
            :class="`inline-flex min-h-11 items-center rounded-full border border-white/[.12] px-4
                     text-[.75rem] text-ink-2 transition hover:text-ink ${ring}`"
            @click="refreshStats()"
          >به‌روزرسانی</button>
        </div>
      </section>

      <section class="mt-10">
        <h2 class="mb-3 text-cap font-bold text-ink-2">حساب‌ها</h2>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div v-for="c in [
            { l: 'کاربران', v: summary?.counts.users },
            { l: 'تأییدشده', v: summary?.counts.verified },
            { l: 'جست‌وجوی ذخیره‌شده', v: summary?.counts.saved_searches },
            { l: 'نشست فعال', v: summary?.counts.active_sessions },
            { l: 'پیام خوانده‌نشده', v: summary?.counts.unread_contact },
            { l: 'مشترک خبرنامه', v: summary?.counts.subscribers },
            { l: 'مشترک تأییدشده', v: summary?.counts.subscribers_confirmed },
            { l: 'رضایت بازاریابی کاربران', v: summary?.counts.marketing_optin },
          ]" :key="c.l" class="rounded-2xl border border-white/[.07] bg-surface p-5">
            <div class="text-[.75rem] text-ink-2">{{ c.l }}</div>
            <div class="mt-1 text-[1.15rem] font-bold">{{ c.v == null ? '-' : f.fa(c.v) }}</div>
          </div>
        </div>
      </section>

      <section class="mt-10">
        <h2 class="mb-3 text-cap font-bold text-ink-2">کاربران</h2>
        <AdminTable
          :cols="['ایمیل', 'نام', 'تأیید', 'مدیر', 'رضایت بازاریابی', 'ثبت‌نام', 'آخرین ورود']"
          :rows="users.rows" :total="users.total" :offset="users.offset" :limit="users.limit"
          :loading="users.busy" :failed="users.failed"
          @page="users.page($event)"
        >
          <template #row="{ row }">
            <tr class="border-b border-white/[.04] last:border-0">
              <td class="whitespace-nowrap px-4 py-3 font-mono text-[.78rem]" dir="ltr">{{ row.email }}</td>
              <td class="px-4 py-3">{{ row.display_name || '-' }}</td>
              <td class="px-4 py-3">
                <span class="rounded-full border px-2.5 py-1 text-[.72rem]" :class="yn(row.verified)">
                  {{ row.verified ? 'بله' : 'خیر' }}
                </span>
              </td>
              <td class="px-4 py-3 text-ink-2">{{ row.is_admin ? 'مدیر' : '-' }}</td>
              <!-- Spelled out rather than a tick, because a blank cell reads as
                   "no data" and this column is a legal record of a yes or a no. -->
              <td class="px-4 py-3">
                <span class="rounded-full border px-2.5 py-1 text-[.72rem]" :class="yn(row.marketing_consent)">
                  {{ row.marketing_consent ? 'داده' : 'نداده' }}
                </span>
              </td>
              <td class="whitespace-nowrap px-4 py-3 text-ink-2">{{ on(row.created_at) }}</td>
              <td class="whitespace-nowrap px-4 py-3 text-ink-2">{{ on(row.last_login_at) }}</td>
            </tr>
          </template>
        </AdminTable>
      </section>

      <section class="mt-10">
        <h2 class="mb-3 text-cap font-bold text-ink-2">فهرست ایمیل</h2>

        <!-- The one sentence the owner needs, and one number inside it.
             The permitted figure comes from the same query that writes the CSV,
             so the sentence and the file can never disagree; nothing here is
             added up or subtracted in the browser. -->
        <div class="glass rounded-2xl border border-white/[.07] p-5 sm:p-6">
          <p class="text-[.95rem] leading-8">
            از <b>{{ captured == null ? '-' : f.fa(captured) }}</b> نشانی‌ای که داریم،
            تنها <b class="text-good">{{ mailable == null ? '-' : f.fa(mailable) }}</b> نشانی
            اجازه‌ی ایمیل تبلیغاتی داده است. خروجی CSV دقیقاً همین فهرست است.
          </p>
          <p class="mt-2 text-[.82rem] leading-7 text-ink-2">
            بقیه یا هنگام ثبت‌نام تیک بازاریابی را نزده‌اند، یا تأیید دومرحله‌ای خبرنامه را
            کامل نکرده‌اند، یا از خبرنامه لغو اشتراک کرده‌اند. ارسال به این گروه، دامنه را
            به فهرست سیاه می‌برد.
          </p>
          <p class="mt-2 text-[.78rem] leading-6 text-ink-2">
            کاربران و مشترکان دو جدول جدا هستند؛ کسی که در هر دو باشد در عدد کل دو بار شمرده
            می‌شود، پس تفاضل این دو عدد رقم دقیقی نیست.
          </p>
        </div>

        <div class="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2">
          <button
            type="button" :disabled="exporting"
            :class="`inline-flex min-h-11 items-center rounded-full border border-good/30 bg-good/[.12] px-4
                     text-[.85rem] text-good transition hover:bg-good/[.2] disabled:opacity-40 ${ring}`"
            @click="exportCsv()"
          >{{ exporting ? 'در حال آماده‌سازی…' : 'خروجی CSV فهرست قابل ارسال' }}</button>
          <span class="text-[.78rem] leading-6 text-ink-2">
            همان فهرستی که عدد بالا می‌گوید.
          </span>
        </div>
        <p v-if="exportError" role="alert" class="mt-2 text-[.8rem] text-accent">{{ exportError }}</p>
        <!-- No row count. It was derived by splitting the file on newline, which
             over-reports the moment a quoted cell contains one, and it was a
             second number for the list the sentence above already states. -->
        <p v-else-if="exported" role="status" class="mt-2 text-[.8rem] text-good">
          فایل خروجی ساخته شد.
        </p>

        <div class="mt-5">
          <AdminTable
            :cols="['ایمیل', 'وضعیت', 'تأیید', 'لغو اشتراک', 'منبع', 'ثبت‌نام']"
            :rows="subs.rows" :total="subs.total" :offset="subs.offset" :limit="subs.limit"
            :loading="subs.busy" :failed="subs.failed"
            @page="subs.page($event)"
          >
            <template #row="{ row }">
              <tr class="border-b border-white/[.04] last:border-0">
                <td class="whitespace-nowrap px-4 py-3 font-mono text-[.78rem]" dir="ltr">{{ row.email }}</td>
                <td class="px-4 py-3">
                  <span class="whitespace-nowrap rounded-full border px-2.5 py-1 text-[.72rem]" :class="subState(row).cls">
                    {{ subState(row).label }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-ink-2">{{ on(row.confirmed_at) }}</td>
                <td class="whitespace-nowrap px-4 py-3" :class="row.unsubscribed_at ? 'text-warn' : 'text-ink-2'">
                  {{ on(row.unsubscribed_at) }}
                </td>
                <td class="px-4 py-3 font-mono text-[.76rem] text-ink-2" dir="ltr">{{ row.source || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-ink-2">{{ on(row.created_at) }}</td>
              </tr>
            </template>
          </AdminTable>
        </div>
      </section>

      <section class="mt-10">
        <h2 class="mb-3 text-cap font-bold text-ink-2">سرویس‌ها</h2>
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
            :class="`flex items-center justify-between rounded-2xl border border-white/[.07] bg-surface p-5 hover:border-white/[.12] ${ring}`"
          >
            <span class="text-[.9rem]">بررسی آمادگی سرویس جست‌وجو</span>
            <span class="font-mono text-[.74rem] text-ink-2" dir="ltr">/readyz ↗</span>
          </a>
        </div>
      </section>
    </template>
  </main>
</template>
