/**
 * Written rather than static because the disallow list has to track the app.
 *
 * Account and admin pages are noindex in their own head as well; this stops a
 * crawler spending its budget fetching them at all.
 */
export default defineEventHandler((event) => {
  setHeader(event, 'Content-Type', 'text/plain; charset=utf-8')
  return [
    'User-agent: *',
    'Allow: /',
    'Disallow: /account',
    'Disallow: /admin',
    'Disallow: /login',
    'Disallow: /register',
    'Disallow: /verify',
    'Disallow: /reset',
    'Disallow: /forgot',
    'Disallow: /api/',
    '',
    'Sitemap: https://khodrobin.noxioai.com/sitemap.xml',
    '',
  ].join('\n')
})
