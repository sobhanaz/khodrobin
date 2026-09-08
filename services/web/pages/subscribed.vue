<script setup lang="ts">
useSeoMeta({ title: 'عضویت در خبرنامه — خودروبین', robots: 'noindex' })

const route = useRoute()
const state = ref<'working' | 'ok' | 'failed'>('working')
const message = ref('')

// This click is the second half of the signup, and the half that matters. The
// form on the home page only records that someone typed an address; anyone can
// type someone else's. Confirming from the inbox is what proves the address
// belongs to the person asking — which is also the difference between a list
// that reaches people and one that trains mail providers to filter the domain.
onMounted(async () => {
  const token = route.query.token
  if (typeof token !== 'string' || !token) {
    state.value = 'failed'
    message.value = 'لینک نامعتبر است.'
    return
  }
  try {
    await $fetch(apiUrl('/api/auth/subscribe/confirm') + '?token=' + encodeURIComponent(token))
    state.value = 'ok'
  } catch (e: any) {
    state.value = 'failed'
    message.value = e?.data?.error ?? 'این لینک نامعتبر یا منقضی است.'
  }
})
</script>

<template>
  <AuthShell title="عضویت در خبرنامه">
    <div v-if="state === 'working'" role="status" class="text-ink-2">در حال تأیید…</div>

    <div v-else-if="state === 'ok'" role="status" class="rounded-2xl border border-good/30 bg-good/[.12] p-6 text-center">
      <div class="text-[1.05rem] font-bold text-good">عضو شدی</div>
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        از این به بعد خبرنامه‌ی خودروبین به همین نشانی می‌رسد. لغوش هم یک کلیک است، از پای هر ایمیل.
      </p>
      <NuxtLink to="/" class="mt-4 inline-block rounded-xl bg-accent px-4 py-2 text-[.85rem] font-bold text-white">
        شروع جست‌وجو
      </NuxtLink>
    </div>

    <div v-else role="alert" class="rounded-2xl border border-warn/25 bg-warn/[.12] p-6 text-center">
      <div class="font-bold text-warn">{{ message }}</div>
      <!-- Naming the two causes matters here: a confirm link that has expired
           and one that was already used look identical from this side, and only
           one of them means the person still has to do something. -->
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        یا قبلاً از همین لینک استفاده شده، یا مهلتش گذشته. اگر خبرنامه‌ای نگرفتی، از صفحه‌ی اصلی دوباره ایمیلت را بده.
      </p>
      <NuxtLink to="/" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">
        بازگشت به صفحه‌ی اصلی
      </NuxtLink>
    </div>
  </AuthShell>
</template>
