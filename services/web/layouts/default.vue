<script setup lang="ts">
const { user, isLoggedIn, init, logout } = useAuth()
const route = useRoute()
const open = ref(false)

onMounted(() => { void init() })
watch(() => route.fullPath, () => { open.value = false })

const links = [
  { to: '/search', label: 'جست‌وجو' },
  { to: '/about', label: 'درباره' },
  { to: '/faq', label: 'پرسش‌ها' },
  { to: '/contact', label: 'تماس' },
]
</script>

<template>
  <div class="min-h-screen">
    <header
      class="sticky top-0 z-50 border-b border-white/[.07]"
      style="background: rgba(7,8,11,.72); backdrop-filter: blur(14px) saturate(140%)"
    >
      <div class="mx-auto flex h-[62px] max-w-[1080px] items-center gap-3 px-5">
        <NuxtLink to="/" class="flex shrink-0 items-center gap-2.5 text-[1.06rem] font-black">
          <BrandMark :size="26" class="shrink-0 text-ink" />
          خودروبین
        </NuxtLink>

        <nav class="hidden items-center gap-1 md:flex" aria-label="اصلی">
          <NuxtLink
            v-for="l in links"
            :key="l.to"
            :to="l.to"
            class="rounded-full px-3 py-1.5 text-[.84rem] transition"
            :class="route.path === l.to ? 'bg-white/[.06] text-ink' : 'text-ink-3 hover:text-ink-2'"
          >{{ l.label }}</NuxtLink>
        </nav>

        <span class="flex-1" />

        <div class="hidden items-center gap-2 md:flex">
          <template v-if="isLoggedIn">
            <NuxtLink v-if="user?.admin" to="/admin"
              class="rounded-full border border-white/[.12] px-3 py-1.5 text-[.8rem] text-ink-2 hover:text-ink">
              مدیریت
            </NuxtLink>
            <NuxtLink to="/account"
              class="rounded-full border border-white/[.12] px-3 py-1.5 text-[.8rem] text-ink-2 hover:text-ink">
              حساب من
            </NuxtLink>
            <button type="button" class="text-[.8rem] text-ink-3 hover:text-ink" @click="logout()">خروج</button>
          </template>
          <template v-else>
            <NuxtLink to="/login" class="text-[.84rem] text-ink-2 hover:text-ink">ورود</NuxtLink>
            <NuxtLink to="/register"
              class="rounded-full bg-accent px-4 py-1.5 text-[.84rem] font-bold text-white transition hover:brightness-110">
              ثبت‌نام
            </NuxtLink>
          </template>
        </div>

        <button
          type="button"
          class="grid size-11 place-items-center rounded-lg border border-white/[.12] md:hidden
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2
                 focus-visible:outline-focus"
          :aria-expanded="open"
          :aria-label="open ? 'بستن منو' : 'باز کردن منو'"
          aria-controls="mobile-nav"
          @click="open = !open"
        >
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M4 7h16M4 12h16M4 17h16" stroke-linecap="round" />
          </svg>
        </button>
      </div>

      <!-- Every row in the drawer is a full-width 44px target. A navigation menu
           on a phone is the one place where a mis-tap costs the most: it sends
           you to the wrong page and you have to come back. -->
      <div v-if="open" id="mobile-nav" class="border-t border-white/[.07] px-5 py-2 md:hidden">
        <NuxtLink v-for="l in links" :key="l.to" :to="l.to"
          class="flex min-h-[44px] items-center text-[.9rem] text-ink-2">{{ l.label }}</NuxtLink>
        <div class="mt-1 flex gap-4 border-t border-white/[.07] pt-2">
          <template v-if="isLoggedIn">
            <NuxtLink to="/account" class="flex min-h-[44px] items-center text-[.9rem] text-ink-2">حساب من</NuxtLink>
            <button type="button" class="min-h-[44px] text-[.9rem] text-ink-3" @click="logout()">خروج</button>
          </template>
          <template v-else>
            <NuxtLink to="/login" class="text-[.9rem] text-ink-2">ورود</NuxtLink>
            <NuxtLink to="/register" class="text-[.9rem] font-bold text-accent">ثبت‌نام</NuxtLink>
          </template>
        </div>
      </div>
    </header>

    <slot />

    <AppFooter />
  </div>
</template>

<style scoped>
</style>
