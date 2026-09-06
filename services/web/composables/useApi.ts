/**
 * Resolve the API base for the current execution context.
 *
 * During SSR a relative URL would resolve against the Nuxt server itself, so
 * the server must call the Go service by its compose hostname. In the browser
 * the base is empty: requests go same-origin and Caddy routes /api/* across,
 * which means no CORS surface exists at all.
 */
export function useApiBase(): string {
  const config = useRuntimeConfig()
  return import.meta.server ? config.apiBase : config.public.apiBase
}

export function apiUrl(path: string): string {
  return `${useApiBase()}${path}`
}
