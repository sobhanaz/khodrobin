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
 * Everything the second signup step collects.
 *
 * `foundVia` and `useCase` carry the fixed values behind the chips, never their
 * Persian labels: the label is copy and copy gets rewritten, and a column that
 * changes meaning when someone edits a string is a column nobody can count.
 * The free-text twins are sent only when the chosen value is 'other'.
 */
export interface SignupProfile {
  phone: string
  foundVia: string
  foundViaOther?: string
  useCase: string
  useCaseOther?: string
  marketingConsent: boolean
  name?: string
}

/**
 * Session state, shared across every component.
 *
 * The access token is held in memory only, never localStorage. A token in
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

  // document.cookie only ever exposes non-HttpOnly cookies, which is exactly
  // what this one is for. Guarded for SSR, where there is no document at all.
  function hasSessionHint(): boolean {
    if (import.meta.server || typeof document === 'undefined') return false
    return document.cookie.split('; ').some(c => c.startsWith('kb_session='))
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

  async function register(email: string, password: string, profile: SignupProfile) {
    pending.value = true; error.value = null
    // The flag is always sent, including when it is false. An omitted field and
    // a declined one look identical on the wire, and the day someone asks where
    // a marketing address came from, "we never sent a no" is not an answer. The
    // refusal is the half of the record worth having.
    const core = {
      email,
      password,
      marketing_consent: profile.marketingConsent,
      ...(profile.name ? { name: profile.name } : {}),
    }
    const full = {
      ...core,
      phone: profile.phone,
      found_via: profile.foundVia,
      ...(profile.foundViaOther ? { found_via_other: profile.foundViaOther } : {}),
      use_case: profile.useCase,
      ...(profile.useCaseOther ? { use_case_other: profile.useCaseOther } : {}),
    }
    const post = (body: object) =>
      $fetch<{ message: string }>(apiUrl('/api/auth/register'), { method: 'POST', body })

    try {
      try {
        return (await post(full)).message
      } catch (e: any) {
        // ponytail: the auth service decodes with DisallowUnknownFields, so a
        // build that has not yet learned the three profile fields answers 400
        // «درخواست نامعتبر است.» and no account is created at all. Rather than
        // take the signup funnel down with it, retry once without them, which is
        // exactly the request that shipped before. Delete this whole branch once
        // `credentials` in services/auth/internal/handlers/handlers.go carries
        // phone / found_via / use_case; keeping it hides the day they stop being
        // stored.
        if (e?.status !== 400 || e?.data?.error !== 'درخواست نامعتبر است.') throw e
        console.warn('register: auth service rejected the profile fields; sent without them')
        return (await post(core)).message
      }
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
  //
  // Unless there is visibly no session to restore. The refresh token is
  // HttpOnly and unreadable here, so this used to ask unconditionally — and
  // search needs no account, so most visitors have never had one. Every one of
  // them paid a request on the critical path to be told 401, and saw the error
  // in their console.
  //
  // kb_session is the server's readable hint that a session exists. It grants
  // nothing: forging it only buys the right to make the request that used to
  // happen anyway, and the real token still has to be there. Absent, we skip.
  //
  // One-time cost of shipping this: anyone holding a session issued before the
  // hint existed looks signed out until they sign in again, which reissues both.
  async function init() {
    if (ready.value) return
    if (hasSessionHint()) await refresh()
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
