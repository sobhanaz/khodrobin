/**
 * The guest's starred cars, in this browser and nowhere else.
 *
 * Search here is anonymous by design, so this cannot sit behind an account:
 * making someone register before they can keep three cars in a list is asking
 * for the commitment before the product has earned it. localStorage is the
 * whole backend. Signing in genuinely adds price alerts on a saved SEARCH; it
 * does not sync this list, and /saved says so rather than implying otherwise.
 */
export interface SavedSpec {
  key: string
  name: string
  /**
   * The median AT THE MOMENT OF SAVING, which is the entire point of the
   * feature: /saved subtracts it from today's median to say what moved.
   * null when the index called that median unreliable — see SaveButton.
   */
  median: number | null
  at: string
}

const KEY = 'khodrobin.saved.v1'
const CAP = 60

/**
 * A spec key is exactly six slug segments: brand/model/trim/gearbox/year/bucket.
 *
 * The check matters because this string is pasted straight into an API path,
 * and localStorage is the one input on this page a user can hand-edit. Six
 * segments of [a-z0-9-] cannot express a traversal or a query string. If the
 * crawler ever widens the slug alphabet, old saves stop validating and vanish
 * from the list — visible, and better than the alternative.
 */
const KEY_SHAPE = /^[a-z0-9-]+(?:\/[a-z0-9-]+){5}$/

/**
 * null means storage could not be read at all — private mode on some browsers
 * throws on access, and there is no localStorage during SSR. That is different
 * from "read fine, nothing there", and the caller must not treat a throw as
 * proof the list is empty and then overwrite a list it still holds in memory.
 */
function read(): SavedSpec[] | null {
  if (import.meta.server) return null
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as { specs?: unknown }
    if (!Array.isArray(parsed?.specs)) return []
    // De-duplicated on the way IN, not only on the way out. save() already
    // drops an older copy, but nothing stops a second tab, an older build or a
    // hand-edit from leaving two rows with the same key, and /saved rendered
    // both: two cards for one car, quoting two different "when you saved it"
    // prices. The list is newest-first, so the first sighting is the keeper.
    const out: SavedSpec[] = []
    const seen = new Set<string>()
    for (const s of parsed.specs as Partial<SavedSpec>[]) {
      if (!s || typeof s.key !== 'string' || !KEY_SHAPE.test(s.key)) continue
      if (typeof s.name !== 'string' || seen.has(s.key)) continue
      seen.add(s.key)
      out.push({
        key: s.key,
        name: s.name,
        median: typeof s.median === 'number' && s.median > 0 ? s.median : null,
        at: typeof s.at === 'string' ? s.at : '',
      })
      if (out.length === CAP) break
    }
    return out
  } catch {
    return null
  }
}

function write(specs: SavedSpec[]) {
  // A quota or private-mode failure costs this session's saves and nothing
  // else. The in-memory list has already been updated, so the click the user
  // just made still looks like it worked until they reload — which is the
  // honest amount of function to keep when the browser refuses to store.
  try {
    localStorage.setItem(KEY, JSON.stringify({ specs }))
  } catch {}
}

export function useSaved() {
  const list = useState<SavedSpec[]>('khodrobin.saved', () => [])

  function load() {
    const stored = read()
    if (stored) list.value = stored
  }

  // There is no localStorage during SSR, so the server always renders "nothing
  // saved" and the real list arrives on mount, after hydration has matched, so
  // no mismatch. Re-reading on mount is also how a save made in another tab
  // turns up here.
  onMounted(load)

  // Every write re-reads storage first, and that is the whole point.
  //
  // These used to build the new array from list.value, this tab's memory, and
  // then replace the entire stored blob. Anything a second tab had saved since
  // this tab mounted was destroyed, silently: save A here, save B over there,
  // save C here, and B is gone. The comment above promised cross-tab visibility
  // while the writes quietly clobbered it, because re-reading on MOUNT is not
  // re-reading on WRITE.
  //
  // read() returns null when storage is unreadable, so a private-mode browser
  // still falls back to this tab's memory and degrades exactly as before.
  function save(entry: { key: string, name: string, median: number | null }) {
    const base = read() ?? list.value
    list.value = [
      { ...entry, at: new Date().toISOString() },
      ...base.filter(s => s.key !== entry.key),
    ].slice(0, CAP)
    write(list.value)
  }

  function unsave(key: string) {
    list.value = (read() ?? list.value).filter(s => s.key !== key)
    write(list.value)
  }

  const isSaved = (key: string) => list.value.some(s => s.key === key)

  function clear() {
    list.value = []
    write([])
  }

  return { list, save, unsave, isSaved, clear }
}
