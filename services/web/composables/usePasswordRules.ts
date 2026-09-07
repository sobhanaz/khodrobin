/**
 * The password policy, defined exactly once.
 *
 * It is applied on four screens — register, reset, login and the admin console —
 * and the Go service enforces it independently on two endpoints. Five copies of
 * "at least one symbol" is five chances for one of them to disagree, and the one
 * that disagrees is always found by a user who cannot get in.
 *
 * The Go side is the authority; this exists so the browser can say the same
 * thing live rather than after a round trip. Keep the two definitions in step:
 * runes not bytes, unicode.IsUpper, and "symbol" meaning not a letter, not a
 * digit, not a space.
 */
export interface PasswordRule {
  key: 'length'
  label: string
  ok: boolean
}

export const MIN_PASSWORD_RUNES = 10

export function usePasswordRules(password: Ref<string> | (() => string)) {
  const pw = computed(() => (typeof password === 'function' ? password() : password.value))

  // Counted in runes, the way the server counts it. Counting UTF-16 code units
  // would let a shorter Persian password through than an English one, and
  // .length on a string containing an emoji would count it twice.
  const runes = computed(() => [...pw.value].length)

  const hasLength = computed(() => runes.value >= MIN_PASSWORD_RUNES)
  // Not a letter, not a digit, not whitespace. Defined by exclusion because the
  // set of symbols someone might reach for is not enumerable — a Persian user
  // typing «؟» or «×» is doing exactly what this rule asks for.
  const hasSymbol = computed(() => /[^\p{L}\p{N}\s]/u.test(pw.value))
  // \p{Lu}, not A-Z: a Cyrillic or Greek capital satisfies the same intent, and
  // the Go side's unicode.IsUpper agrees.
  const hasUpper = computed(() => /\p{Lu}/u.test(pw.value))

  const rules = computed(() => [
    { key: 'length', label: `دست‌کم ${MIN_PASSWORD_RUNES} نویسه`, ok: hasLength.value },
    { key: 'symbol', label: 'دست‌کم یک نماد — مثل ! یا @ یا ؟', ok: hasSymbol.value },
    { key: 'upper', label: 'دست‌کم یک حرف بزرگ لاتین — مثل A', ok: hasUpper.value },
  ])

  const satisfied = computed(() => rules.value.every(r => r.ok))
  const met = computed(() => rules.value.filter(r => r.ok).length)

  return { runes, rules, satisfied, met, hasLength, hasSymbol, hasUpper }
}
