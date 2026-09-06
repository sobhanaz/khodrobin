import type { SearchResponse, StatsResponse } from '~/types'

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

export function useSearch() {
  const query = ref('')
  const mode = ref<string>('relevant')
  const data = ref<SearchResponse | null>(null)
  const pending = ref(false)
  const failed = ref(false)

  async function run(q?: string) {
    if (q !== undefined) query.value = q
    pending.value = true
    failed.value = false
    try {
      data.value = await $fetch<SearchResponse>(apiUrl('/api/v1/search'), {
        params: { q: query.value, mode: mode.value, limit: 24 },
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

  return { query, mode, data, pending, failed, run }
}

export function useStats() {
  return useFetch<StatsResponse>(() => apiUrl('/api/v1/stats'), { key: 'stats' })
}
