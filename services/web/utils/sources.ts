/**
 * The marketplaces this product reads, in display order.
 *
 * This existed as two hardcoded arrays that had already drifted: SpecCard
 * listed four and LookupBox listed five, so every search result card on the
 * site silently omitted Sheypoor's badge while the lookup panel showed it. A
 * card that renders "not listed here" for a source we do crawl is not a
 * cosmetic bug — it is the product asserting something false about its own
 * coverage.
 *
 * The order and spelling mirror SOURCE_FA in services/crawler/build_index.py,
 * which is where a source is actually named. When a sixth source lands there,
 * it lands here, and both surfaces move together.
 */
export const ALL_SOURCES = ['دیوار', 'باما', 'همراه‌مکانیک', 'خودرو۴۵', 'شیپور'] as const
