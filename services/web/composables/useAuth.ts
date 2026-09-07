import type { Ref } from 'vue'

export interface AuthUser {
  id: number
  email: string
  name: string | null
  verified: boolean
  admin: boolean
}

interface LoginResponse {
  access_token: string
  expires_in: number
  user?: AuthUser
}

/**
 * Session state, shared across every component.
 *
 * The access token is held in memory only — never localStorage. A token in
 * localStorage survives a tab close and is readable by any script on the page,
 * which turns one XSS bug into a stolen session. The refresh token lives in an
 * HttpOnly cookie the browser will not show JavaScript, so a reload restores
 * the session without the page ever holding a long-lived credential.
 */
export function useAuth() {
  const token = useState<string | null>('auth.token', () => null)
  const user = useState<AuthUser | null>('auth.user', () => null)
  const ready = useState<boolean>('auth.ready', () => false)
  const pending = ref(false)
  const error = ref<string | null>(null)

  const isLoggedIn = computed(() => !!user.value)

  async function call<T>(path: string, opts: Record<string, unknown> = {}): Promise<T> {
    return $fetch<T>(apiUrl(path), {
      credentials: 'include',
      ...opts,
      headers: {
        ...(token.value ? { Authorization: `Bearer ${token.value}` } : {}),
        ...(opts.headers as Record<string, string> | undefined),
      },
    })
  }

  function apply(res: LoginResponse) {
    token.value = res.access_token
    if (res.user) user.value = res.user
    // Refresh a minute before expiry rather than after a 401: a request that
    // fails and retries is a visible stutter in the UI.
    if (import.meta.client) scheduleRefresh(res.expires_in)
  }

  let timer: ReturnType<typeof setTimeout> | null = null
  function scheduleRefresh(expiresIn: number) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => { void refresh() }, Math.max(30, expiresIn - 60) * 1000)
  }

  async function refresh(): Promise<boolean> {
    try {
      const res = await $fetch<LoginResponse>(apiUrl('/api/auth/refresh'), {
        method: 'POST', credentials: 'include',
      })
      apply(res)
      if (!user.value) await loadMe()
      return true
    } catch {
      token.value = null
      user.value = null
      return false
    }
  }

  async function loadMe() {
    try {
      user.value = await call<AuthUser>('/api/auth/me')
    } catch {
      user.value = null
    }
  }

  async function login(email: string, password: string) {
    pending.value = true; error.value = null
    try {
      apply(await $fetch<LoginResponse>(apiUrl('/api/auth/login'), {
        method: 'POST', credentials: 'include', body: { email, password },
      }))
      return true
    } catch (e: any) {
      error.value = e?.data?.error ?? 'ورود انجام نشد. دوباره تلاش کن.'
      return false
    } finally {
      pending.value = false
    }
  }

  async function register(email: string, password: string, name?: string) {
    pending.value = true; error.value = null
    try {
      const res = await $fetch<{ message: string }>(apiUrl('/api/auth/register'), {
        method: 'POST', body: { email, password, ...(name ? { name } : {}) },
      })
      return res.message
    } catch (e: any) {
      error.value = e?.data?.error ?? 'ثبت‌نام انجام نشد. دوباره تلاش کن.'
      return null
    } finally {
      pending.value = false
    }
  }

  async function logout() {
    try { await $fetch(apiUrl('/api/auth/logout'), { method: 'POST', credentials: 'include' }) } catch {}
    if (timer) clearTimeout(timer)
    token.value = null
    user.value = null
    await navigateTo('/')
  }

  // Called once on mount: a page load has no token, only the cookie, so the
  // session is restored by asking for a fresh access token.
  async function init() {
    if (ready.value) return
    await refresh()
    ready.value = true
  }

  return { token, user, ready, pending, error, isLoggedIn, login, register, logout, refresh, loadMe, init, call }
}

/** Redirect guard for pages that require a session. */
export function useRequireAuth(): Ref<boolean> {
  const { isLoggedIn, ready, init } = useAuth()
  onMounted(async () => {
    await init()
    if (!isLoggedIn.value) await navigateTo('/login?next=' + encodeURIComponent(useRoute().fullPath))
  })
  return computed(() => ready.value && isLoggedIn.value) as Ref<boolean>
}
