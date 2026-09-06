/**
 * Persian number formatting.
 *
 * Two formatters on purpose. Quantities are grouped («۱٬۸۷۵ آگهی»); years are
 * identifiers and must not be («۱۴۰۴», never «۱٬۴۰۴»).
 */
const fa = new Intl.NumberFormat('fa-IR')
const faPlain = new Intl.NumberFormat('fa-IR', { useGrouping: false })
const en = new Intl.NumberFormat('en-US')

export const useFormat = () => ({
  /** Persian digits with grouping — for counts. */
  fa: (n: number) => fa.format(n),
  /** Persian digits without grouping — for years. */
  year: (n: number) => faPlain.format(n),
  /**
   * Prices stay in Latin digits. They are long, compared at a glance, and set
   * in a tabular monospace face; Persian digits at this length are markedly
   * harder to scan.
   */
  money: (n: number) => en.format(n),
  /** «۱٫۲ میلیارد» for compact display. */
  short: (n: number) =>
    n >= 1e9 ? `${faPlain.format(Math.round((n / 1e9) * 10) / 10)} میلیارد`
             : `${fa.format(Math.round(n / 1e6))} میلیون`,
})
