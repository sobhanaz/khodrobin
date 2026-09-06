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
        {
          name: 'description',
          content: 'جست‌وجوی خودروی دست‌دوم در دیوار، باما و همراه‌مکانیک. یک خودرو، همه‌ی آگهی‌ها، یک قیمت میانه.',
        },
      ],
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Vazirmatn:wght@300;400;500;700;900&display=swap',
        },
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
