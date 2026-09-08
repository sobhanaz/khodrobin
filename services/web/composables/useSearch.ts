import type { LocationQuery } from 'vue-router'
import { FILTER_KEYS } from '~/types'
import type { FilterOverrides, SearchResponse, StatsResponse } from '~/types'

export const RANKING_MODES = [
  { id: 'relevant', label: 'مرتبط‌ترین' },
  { id: 'cheapest', label: 'ارزان‌ترین' },
  { id: 'lowkm', label: 'کم‌کارکردترین' },
  { id: 'value', label: 'بهترین ارزش' },
] as const

export const EXAMPLE_QUERIES = [
  'ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵',
  'پراید مدل ۹۸',
  'کوییک دنده‌ای',
  'سمند سورن زیر ۲ میلیارد',
  'پژو ۲۰۷ اتوماتیک',
]

/**
 * The marketplaces, keyed the way the index keys them. Mirrors SOURCE_FA in
 * build_index.py; the slugs themselves come live from /api/v1/stats and this
 * map only translates them, so a source added upstream degrades to showing
 * its slug rather than disappearing from the rail.
 */
export const SOURCE_FA: Record<string, string> = {
  divar: 'دیوار', bama: 'باما', hamrah: 'همراه‌مکانیک',
  khodro45: 'خودرو۴۵', sheypoor: 'شیپور',
}

/**
 * Filter overrides exactly as they stand in the URL.
 *
 * '' survives on purpose: ?price_max= (and bare ?price_max, which the router
 * reads as null) both mean "the user removed this filter" and must reach the
 * API as an empty param so it suppresses the value parsed from the text.
 * Dropping empties here would turn every chip removal into a no-op.
 */
export function filtersFromQuery(q: LocationQuery): FilterOverrides {
  const out: FilterOverrides = {}
  for (const k of FILTER_KEYS) {
    if (!(k in q)) continue
    const v = q[k]
    out[k] = typeof v === 'string' ? v : Array.isArray(v) ? String(v[0] ?? '') : ''
  }
  return out
}

const FA_TO_EN: Record<string, string> = {}
for (const [i, d] of [...'۰۱۲۳۴۵۶۷۸۹'].entries()) FA_TO_EN[d] = String(i)
for (const [i, d] of [...'٠١٢٣٤٥٦٧٨٩'].entries()) FA_TO_EN[d] = String(i)

/** Users type prices and years in whichever digits their keyboard is in. */
export function toEnDigits(s: string): string {
  return s.replace(/[۰-۹٠-٩]/g, d => FA_TO_EN[d] ?? d)
}

/**
 * A typed price, in tomans. Mirrors priceScale in the Go parser: Iranians
 * quote car prices in shorthand, so a small bare number means millions —
 * «۵۰۰» is 500 million tomans, not a typo. The UI always previews the
 * interpretation next to the field so the shorthand is never a surprise.
 */
export function parseTomans(raw: string): number | null {
  const s = toEnDigits(raw).replace(/[,٬\s]/g, '')
  if (!s) return null
  const n = Number(s)
  if (!Number.isFinite(n) || n <= 0) return null
  return Math.round(n < 10_000 ? n * 1_000_000 : n)
}

/**
 * A typed model year, normalized to Jalali. Mirrors normalizeYear in the Go
 * parser — the two must agree, or the rail would filter on a different year
 * than the same digits typed into the search box.
 */
export function parseJalaliYear(raw: string): number | null {
  const s = toEnDigits(raw).trim()
  if (!/^\d{1,4}$/.test(s)) return null
  const n = Number(s)
  if (n >= 1300 && n <= 1450) return n
  if (n >= 60 && n <= 99) return 1300 + n
  if (n >= 0 && n <= 20) return 1400 + n
  if (n >= 1990 && n <= 2100) return n - 621
  return null
}

export function useSearch() {
  const route = useRoute()
  const query = ref('')
  const mode = ref<string>('relevant')
  const data = ref<SearchResponse | null>(null)
  const pending = ref(false)
  const failed = ref(false)

  // Filter state lives in the URL and nowhere else. The chips, the rail and
  // this fetch all read the same route.query, which is the only reason the
  // three can never disagree about what is filtered.
  const filters = computed<FilterOverrides>(() => filtersFromQuery(route.query))

  async function run(q?: string) {
    if (q !== undefined) query.value = q
    pending.value = true
    failed.value = false
    try {
      data.value = await $fetch<SearchResponse>(apiUrl('/api/v1/search'), {
        // Overrides ride along even when empty: empty is the "cleared" signal
        // that tells the API to suppress what it parsed from the text.
        params: { q: query.value, mode: mode.value, limit: 24, ...filters.value },
      })
    } catch {
      failed.value = true
    } finally {
      pending.value = false
    }
  }

  // Changing the ranking mode re-runs the same query rather than re-sorting
  // client-side: the score depends on the matched set, so the server owns it.
  watch(mode, () => run())

  return { query, mode, data, pending, failed, filters, run }
}

export function useStats() {
  return useFetch<StatsResponse>(() => apiUrl('/api/v1/stats'), { key: 'stats' })
}
