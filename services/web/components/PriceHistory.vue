<script setup lang="ts">
const props = defineProps<{ specKey: string }>()
const f = useFormat()

// Local shapes for the history contract. types.ts is another surface's file
// this run, and two small interfaces are not worth a cross-file dependency.
interface HistoryPoint { t: string, median: number, offers: number }
interface HistoryResponse { key: string, points: HistoryPoint[] | null }

const points = ref<HistoryPoint[] | null>(null)
const pending = ref(true)
// Distinguishes "nothing recorded yet" from "we could not ask". They are not
// the same claim and must not share a sentence.
const unavailable = ref(false)

/**
 * A 404 and an empty list mean the same thing to a buyer: no history yet. The
 * endpoint 404s on deploys older than the history file, and returns [] for a
 * spec whose median was never reliable enough to record. Neither is a failure
 * the reader can act on, so both land on the honest empty state.
 *
 * Everything else does NOT mean that, and used to. A bare catch collapsed a
 * dropped connection and a 500 into «ثبت قیمت از امروز شروع شده» — so a car
 * with thirty days of recorded history told a reader on a flaky connection
 * that recording had started today. The comment justified two cases while the
 * code swallowed every case, which is this project's most repeated bug.
 */
onMounted(async () => {
  try {
    const res = await $fetch<HistoryResponse>(apiUrl(`/api/v1/history/${props.specKey}`))
    points.value = [...(res.points ?? [])].sort((a, b) => a.t.localeCompare(b.t))
  } catch (err: any) {
    if (err?.status === 404 || err?.statusCode === 404) {
      points.value = []
    } else {
      // Assert nothing about the data we failed to read.
      unavailable.value = true
      points.value = []
    }
  } finally {
    pending.value = false
  }
})

const latest = computed(() => {
  const pts = points.value
  return pts?.length ? pts[pts.length - 1]! : null
})

const domain = computed(() => {
  const pts = points.value ?? []
  if (!pts.length) return null
  let lo = Infinity, hi = -Infinity, t0 = Infinity, t1 = -Infinity
  for (const p of pts) {
    lo = Math.min(lo, p.median); hi = Math.max(hi, p.median)
    const ms = Date.parse(p.t); t0 = Math.min(t0, ms); t1 = Math.max(t1, ms)
  }
  return { lo, hi, t0, t1 }
})

/**
 * Coordinates in the 0–100 viewBox, x by real timestamp rather than by index:
 * the crawler skips a cycle whenever the median is not reliable, and an
 * index-based axis would silently squeeze those gaps shut. A flat or
 * single-point series sits on the centre line instead of dividing by zero.
 */
const coords = computed(() => {
  const d = domain.value
  if (!d) return []
  return (points.value ?? []).map((p) => {
    const x = d.t1 > d.t0 ? ((Date.parse(p.t) - d.t0) / (d.t1 - d.t0)) * 100 : 50
    const y = d.hi > d.lo ? 8 + (1 - (p.median - d.lo) / (d.hi - d.lo)) * 84 : 50
    return { x, y }
  })
})

const line = computed(() => coords.value.map(c => `${c.x.toFixed(2)},${c.y.toFixed(2)}`).join(' '))
const area = computed(() => {
  const c = coords.value
  if (c.length < 2) return ''
  return `${line.value} ${c[c.length - 1]!.x.toFixed(2)},100 ${c[0]!.x.toFixed(2)},100`
})
const dot = computed(() => coords.value.length ? coords.value[coords.value.length - 1]! : null)

// fa-IR defaults to the Persian calendar, so this is a Jalali date for free.
const day = new Intl.DateTimeFormat('fa-IR', { day: 'numeric', month: 'long' })
const fmtDay = (iso: string) => day.format(new Date(iso))
</script>

<template>
  <div>
    <div v-if="pending" class="h-24 animate-pulse rounded-xl bg-surface-2" aria-hidden="true" />

    <p v-else-if="unavailable" class="text-[.88rem] leading-7 text-ink-3">
      نمودار قیمت فعلاً در دسترس نیست.
    </p>

    <p v-else-if="!points?.length" class="text-[.88rem] leading-7 text-ink-2">
      ثبت قیمت از امروز شروع شده؛ نمودار با هر به‌روزرسانی (هر ۳ ساعت) کامل‌تر می‌شود.
    </p>

    <template v-else>
      <div class="flex flex-wrap items-end gap-x-6 gap-y-2">
        <div>
          <div class="text-[.68rem] text-ink-3">آخرین میانه</div>
          <div dir="ltr" class="text-right">
            <span class="font-mono text-[1.05rem] font-bold tabular-nums">{{ f.money(latest!.median) }}</span>
            <span class="ms-1 font-sans text-[.72rem] text-ink-3">تومان</span>
          </div>
        </div>
        <template v-if="points.length > 1">
          <div>
            <div class="text-[.68rem] text-ink-3">کمترین میانه</div>
            <div class="mono-nums text-[.86rem] text-ink-2">{{ f.money(domain!.lo) }}</div>
          </div>
          <div>
            <div class="text-[.68rem] text-ink-3">بیشترین میانه</div>
            <div class="mono-nums text-[.86rem] text-ink-2">{{ f.money(domain!.hi) }}</div>
          </div>
        </template>
        <div class="text-[.72rem] text-ink-3">
          {{ fmtDay(latest!.t) }} · از {{ f.fa(latest!.offers) }} آگهی
        </div>
      </div>

      <div
        class="relative mt-4 h-24"
        dir="ltr"
        role="img"
        :aria-label="`نمودار میانه‌ی قیمت، ${f.fa(points.length)} ثبت`"
      >
        <svg v-if="points.length > 1" class="absolute inset-0 size-full" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
          <polygon :points="area" class="fill-ink opacity-[.05]" />
          <polyline
            :points="line"
            fill="none"
            class="stroke-ink-2"
            stroke-width="2"
            vector-effect="non-scaling-stroke"
            stroke-linejoin="round"
            stroke-linecap="round"
          />
        </svg>
        <span
          class="absolute size-[9px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-ink shadow-[0_0_0_3px_var(--color-surface)]"
          :style="{ left: `${dot!.x}%`, top: `${dot!.y}%` }"
          aria-hidden="true"
        />
        <!-- One point is a dot with its value beside it, not a chart. Padding
             or interpolating a second point would draw a trend nobody measured. -->
        <span
          v-if="points.length === 1"
          class="absolute left-1/2 top-[68%] -translate-x-1/2 font-mono text-[.72rem] tabular-nums text-ink-3"
        >{{ f.money(latest!.median) }}</span>
      </div>

      <div v-if="points.length > 1" class="mt-1 flex justify-between text-[.64rem] text-ink-3">
        <span>{{ fmtDay(points[0]!.t) }}</span>
        <span>{{ fmtDay(latest!.t) }}</span>
      </div>

      <p class="mt-3 text-[.72rem] leading-6 text-ink-3">
        <template v-if="points.length === 1">اولین ثبت؛ نمودار با به‌روزرسانی‌های بعدی (هر ۳ ساعت) شکل می‌گیرد.</template>
        <template v-else>هر ۳ ساعت به‌روز می‌شود؛ فقط میانه‌های قابل اتکا (دست‌کم ۳ آگهی) ثبت می‌شوند.</template>
      </p>
    </template>
  </div>
</template>
