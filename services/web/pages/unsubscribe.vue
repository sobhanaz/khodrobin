<script setup lang="ts">
useSeoMeta({ title: 'لغو اشتراک | خودروبین', robots: 'noindex' })

const route = useRoute()
const state = ref<'working' | 'ok' | 'failed'>('working')
const message = ref('')

// It unsubscribes on mount, with no confirm step. Someone who clicked
// «لغو اشتراک» in an email has already decided; an "are you sure?" here does
// not win the address back, it converts a quiet opt-out into a spam report,
// and a spam report costs the whole sending domain, including the price alerts
// people did ask for.
onMounted(async () => {
  const token = route.query.token
  if (typeof token !== 'string' || !token) {
    state.value = 'failed'
    message.value = 'لینک نامعتبر است.'
    return
  }
  try {
    await $fetch(apiUrl('/api/auth/unsubscribe') + '?token=' + encodeURIComponent(token))
    state.value = 'ok'
  } catch (e: any) {
    state.value = 'failed'
    message.value = e?.data?.error ?? 'این لینک نامعتبر یا منقضی است.'
  }
})
</script>

<template>
  <AuthShell title="لغو اشتراک">
    <div v-if="state === 'working'" role="status" class="text-ink-2">در حال لغو…</div>

    <!-- The endpoint is idempotent and this screen is too: a second click reads
         exactly like the first. Answering it with "you were already
         unsubscribed" replies to a question nobody asked and leaves the person
         wondering whether the first click had failed. -->
    <div v-else-if="state === 'ok'" role="status" class="rounded-2xl border border-good/30 bg-good/[.12] p-6 text-center">
      <div class="text-[1.05rem] font-bold text-good">لغو شد</div>
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        دیگر ایمیل تبلیغاتی برایت نمی‌فرستیم. اگر حساب داری، هشدارهای قیمتی که خودت ساخته‌ای سر جایشان هستند.
      </p>

      <!-- The way back is on the page. In most mail clients the unsubscribe
           link sits a few pixels from the one people meant to click, so an
           accidental opt-out is common, and making them hunt for the signup
           box again is how it turns into a permanent one. -->
      <p class="mt-5 border-t border-good/20 pt-4 text-[.82rem] leading-7 text-ink-2">
        اشتباهی کلیک کردی؟
        <NuxtLink to="/" class="text-accent hover:underline">از صفحه‌ی اصلی دوباره عضو شو</NuxtLink>.
      </p>
    </div>

    <div v-else role="alert" class="rounded-2xl border border-warn/25 bg-warn/[.12] p-6 text-center">
      <div class="font-bold text-warn">{{ message }}</div>
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        آدرس را کامل از ایمیل کپی کن. اگر باز هم کار نکرد و ایمیل می‌گیری، به ما بگو تا دستی حذفت کنیم.
      </p>
      <NuxtLink to="/contact" class="mt-4 inline-flex min-h-[44px] items-center text-[.85rem] text-accent hover:underline
                              focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus">
        تماس با ما
      </NuxtLink>
    </div>
  </AuthShell>
</template>
