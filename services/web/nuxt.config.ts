import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  compatibilityDate: '2026-09-01',
  devtools: { enabled: false },
  css: ['~/assets/css/main.css'],
  vite: { plugins: [tailwindcss()] },

  app: {
    // Persian-first. Setting dir here rather than per-component means every
    // logical CSS property resolves correctly without a single RTL override.
    head: {
      htmlAttrs: { lang: 'fa', dir: 'rtl' },
      title: 'خودروبین — ترب برای خودروی دست‌دوم',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
        { name: 'color-scheme', content: 'dark' },
        // A shared link with no card is a shared link nobody clicks, and this
        // one gets pasted into Telegram and WhatsApp more than anywhere else.
        { property: 'og:image', content: 'https://khodrobin.noxioai.com/og.png' },
        { property: 'og:image:width', content: '1200' },
        { property: 'og:image:height', content: '630' },
        { property: 'og:type', content: 'website' },
        { property: 'og:locale', content: 'fa_IR' },
        { name: 'twitter:card', content: 'summary_large_image' },
        { name: 'theme-color', content: '#07080b' },
        {
          name: 'description',
          content: 'جست‌وجوی خودروی دست‌دوم در دیوار، باما و همراه‌مکانیک. یک خودرو، همه‌ی آگهی‌ها، یک قیمت میانه.',
        },
      ],
      link: [
        // Self-hosted, not Google Fonts. The audience for this site — and the
        // reviewer it was built for — is in Iran, where fonts.gstatic.com is
        // routinely slow or unreachable. A blocked stylesheet does not fail
        // loudly; it silently drops the page to the Tahoma fallback, which is
        // the one outcome a Persian typographic layout cannot survive.
        // One variable file also replaces five weight requests.
        {
          rel: 'preload', as: 'font', type: 'font/woff2',
          href: '/fonts/Vazirmatn.woff2', crossorigin: '',
        },
        // An SVG favicon so the mark stays sharp on every display and can flip
        // its ink under a light tab strip; the PNG is what iOS uses, because it
        // masks the corners itself and ignores transparency.
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
      ],
    },
  },

  runtimeConfig: {
    // Server-side calls go straight to the Go service over the compose network.
    apiBase: process.env.NUXT_API_BASE || 'http://api:8080',
    public: {
      // The browser calls same-origin; Caddy routes /api/* to the Go service.
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '',
    },
  },

  nitro: { preset: 'node-server' },
})
