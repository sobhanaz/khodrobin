interface SpecLite { key: string, offer_count: number, source_count: number }

/**
 * Generated from the live index rather than kept as a file.
 *
 * The catalogue changes every three hours, so a static sitemap would be wrong
 * almost immediately. Specs are ordered by how much they actually offer a
 * visitor — corroborated across sources first — because a crawl budget is
 * finite and a single-offer card is the least useful page we have.
 */
export default defineEventHandler(async (event) => {
  const base = 'https://khodrobin.noxioai.com'
  const config = useRuntimeConfig()

  let specs: SpecLite[] = []
  try {
    const res = await $fetch<{ result: { specs: SpecLite[] } }>(
      `${config.apiBase}/api/v1/search`, { params: { q: '', mode: 'relevant', limit: 100 } })
    specs = res.result.specs
  } catch {
    // A sitemap that 500s teaches a crawler to stop asking. An empty but valid
    // one keeps the static pages listed and can be refetched later.
    specs = []
  }

  const staticPages = ['', '/about', '/faq', '/contact']
  const urls = [
    ...staticPages.map(p => ({ loc: base + p, priority: p === '' ? '1.0' : '0.6', freq: 'weekly' })),
    ...specs
      .slice()
      .sort((a, b) => (b.source_count - a.source_count) || (b.offer_count - a.offer_count))
      .map(s => ({
        loc: `${base}/car/${s.key}`,
        // Multi-source cards are the ones worth indexing: they carry a median
        // backed by more than one seller.
        priority: s.source_count > 1 ? '0.8' : '0.4',
        freq: 'daily',
      })),
  ]

  setHeader(event, 'Content-Type', 'application/xml; charset=utf-8')
  setHeader(event, 'Cache-Control', 'public, max-age=1800')
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.map(u => `  <url><loc>${u.loc}</loc><changefreq>${u.freq}</changefreq><priority>${u.priority}</priority></url>`).join('\n')}
</urlset>`
})
