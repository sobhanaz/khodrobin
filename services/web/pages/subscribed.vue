<script setup lang="ts">
useSeoMeta({ title: 'عضویت در خبرنامه | خودروبین', robots: 'noindex' })

const route = useRoute()
const state = ref<'working' | 'ok' | 'failed'>('working')
const message = ref('')

// This click is the second half of the signup, and the half that matters. The
// form on the home page only records that someone typed an address; anyone can
// type someone else's. Confirming from the inbox is what proves the address
// belongs to the person asking, which is also the difference between a list
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
    message.value = e?.data?.error ?? 'این لینک به هیچ نشانی‌ای نمی‌خورد.'
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
      <NuxtLink to="/" class="mt-4 inline-flex min-h-[44px] items-center rounded-xl bg-accent px-4 text-[.85rem] font-bold text-white
                              focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus">
        شروع جست‌وجو
      </NuxtLink>
    </div>

    <div v-else role="alert" class="rounded-2xl border border-warn/25 bg-warn/[.12] p-6">
      <div class="font-bold text-warn">{{ message }}</div>
      <!-- The two causes named here are the two the server can actually
           produce. Confirmation tokens have no expiry column, so «مهلتش گذشته»
           described a state the schema cannot reach, and a second click is not
           a failure either: the confirm is a COALESCE, so clicking twice
           succeeds twice. What is left is a token that matches no row (a link
           the mail client cut) and a row excluded because the address already
           left the list. -->
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        یا آدرس هنگام کپی از ایمیل ناقص شده (بعضی برنامه‌های ایمیل لینک‌های بلند را می‌شکنند)،
        یا این نشانی قبلاً از فهرست حذف شده و دیگر با همین لینک تأیید نمی‌شود.
      </p>
      <p class="mt-2 text-[.82rem] leading-7 text-ink-2">
        لینک تأیید مهلت ندارد، پس کلیک دوباره روی همان لینک مشکلی درست نمی‌کند.
        اگر باز هم نشد، از صفحه‌ی اصلی دوباره ایمیلت را بده.
      </p>
      <NuxtLink to="/" class="mt-4 inline-flex min-h-[44px] items-center text-[.85rem] text-accent hover:underline
                              focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus">
        بازگشت به صفحه‌ی اصلی
      </NuxtLink>
    </div>
  </AuthShell>
</template>
