<script setup lang="ts">
useSeoMeta({ title: 'تأیید ایمیل در خودروبین', robots: 'noindex' })

const route = useRoute()
const state = ref<'working' | 'code' | 'ok' | 'failed'>('working')
const message = ref('')
const code = ref('')
const busy = ref(false)

onMounted(async () => {
  const token = route.query.token
  if (typeof token !== 'string' || !token) {
    // No token: the visitor came here from the email to type the six-digit code
    // instead of clicking the button. It is the path for clients that strip link
    // buttons, and for people who would rather type.
    state.value = 'code'
    return
  }
  try {
    const res = await $fetch<{ message: string }>(
      apiUrl('/api/auth/verify') + '?token=' + encodeURIComponent(token))
    state.value = 'ok'
    message.value = res.message
    // A newly verified account still needs to log in: verification proves the
    // address, it does not prove whoever opened the link owns the password.
  } catch (e: any) {
    state.value = 'failed'
    message.value = e?.data?.error ?? 'این لینک منقضی شده یا قبلاً استفاده شده است.'
  }
})

async function submitCode() {
  const digits = code.value.trim()
  if (!/^\d{6}$/.test(digits) || busy.value) return
  busy.value = true
  try {
    const res = await $fetch<{ message: string }>(apiUrl('/api/auth/verify/code'), {
      method: 'POST',
      body: { code: digits },
    })
    state.value = 'ok'
    message.value = res.message
  } catch (e: any) {
    state.value = 'failed'
    message.value = e?.data?.error ?? 'این کد معتبر نیست یا منقضی شده است.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="تأیید ایمیل">
    <div v-if="state === 'working'" class="text-ink-2">در حال بررسی لینک…</div>

    <div v-else-if="state === 'code'">
      <p class="mb-4 text-[.88rem] leading-8 text-ink-2">
        کد ۶ رقمی داخل ایمیل را اینجا وارد کن.
      </p>
      <form class="grid gap-4" novalidate @submit.prevent="submitCode">
        <label class="grid gap-2">
          <span class="text-[.82rem] font-bold text-ink-2">کد تأیید</span>
          <input
            v-model="code"
            type="text"
            inputmode="numeric"
            maxlength="6"
            dir="ltr"
            autocomplete="one-time-code"
            autofocus
            class="min-h-[52px] w-full rounded-xl border border-white/[.07] bg-surface-2 px-4 text-center
                   font-mono text-xl tracking-[.5em] text-ink outline-none transition
                   focus:border-focus/60 focus:shadow-[0_0_0_4px_rgba(107,116,230,.16)]
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          />
        </label>

        <button
          type="submit"
          :disabled="busy || !/^\d{6}$/.test(code.trim())"
          class="min-h-[48px] rounded-xl bg-accent px-5 font-bold text-white transition
                 enabled:hover:brightness-110 disabled:opacity-50
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >{{ busy ? 'در حال بررسی…' : 'تأیید با کد' }}</button>
      </form>

      <p class="mt-4 border-t border-white/[.07] pt-4 text-center text-[.78rem] leading-7 text-ink-2">
        کد را نگرفتی یا منقضی شده؟ از صفحه‌ی
        <NuxtLink to="/register" class="text-accent hover:underline">ثبت‌نام</NuxtLink>
        دوباره ایمیلت را بده تا یکی تازه بیاید، یا از همان لینک داخل ایمیل استفاده کن.
      </p>
    </div>

    <div v-else-if="state === 'ok'" class="rounded-2xl border border-good/30 bg-good/[.12] p-6 text-center">
      <div class="text-[1.05rem] font-bold text-good">{{ message }}</div>
      <p class="mt-2 text-[.88rem] text-ink-2">حالا می‌توانی هشدار قیمت بسازی.</p>
      <NuxtLink to="/login" class="mt-4 inline-block rounded-xl bg-accent px-4 py-2 text-[.85rem] font-bold text-white">
        ورود
      </NuxtLink>
    </div>

    <div v-else class="rounded-2xl border border-warn/25 bg-warn/[.12] p-6 text-center">
      <div class="font-bold text-warn">{{ message }}</div>
      <NuxtLink to="/login" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">
        بازگشت به ورود
      </NuxtLink>
    </div>
  </AuthShell>
</template>