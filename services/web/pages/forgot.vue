<script setup lang="ts">
useSeoMeta({ title: 'بازیابی رمز — خودروبین', robots: 'noindex' })

const route = useRoute()
// Prefilled from the login page, so nobody retypes an address they just typed.
const email = ref(typeof route.query.email === 'string' ? route.query.email : '')
const touched = ref(false)
const sending = ref(false)
const sent = ref(false)

const emailError = computed(() =>
  touched.value && !/.+@.+\..+/.test(email.value.trim()) ? 'ایمیل معتبر وارد کن.' : null)

async function submit() {
  touched.value = true
  if (emailError.value || !email.value.trim()) return
  sending.value = true
  try {
    await $fetch(apiUrl('/api/auth/forgot'), { method: 'POST', body: { email: email.value.trim() } })
  } catch {
    // Deliberately ignored. The endpoint answers identically for a known and an
    // unknown address, and the page must not leak the difference either.
  } finally {
    sending.value = false
    sent.value = true
  }
}
</script>

<template>
  <AuthShell
    title="بازیابی رمز"
    subtitle="ایمیلت را بنویس تا لینک انتخاب رمز تازه برایت بفرستیم."
  >
    <div v-if="sent" class="rounded-2xl border border-good/30 bg-good/[.12] p-6 text-center">
      <div class="font-bold text-good">اگر حسابی با این ایمیل باشد، لینک ارسال شد</div>
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        لینک <span class="text-ink">یک ساعت</span> اعتبار دارد و فقط یک بار کار می‌کند.
        نرسید؟ پوشه‌ی اسپم را نگاه کن.
      </p>
      <NuxtLink to="/login" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">
        بازگشت به ورود
      </NuxtLink>
    </div>

    <form v-else class="grid gap-4" novalidate @submit.prevent="submit">
      <FormField
        id="f-email"
        v-model="email"
        label="ایمیل"
        type="email"
        dir="ltr"
        autocomplete="username"
        placeholder="you@example.com"
        :error="emailError"
        required
        autofocus
        @blur="touched = true"
      />

      <button
        type="submit"
        :disabled="sending"
        class="min-h-[48px] rounded-xl bg-accent px-5 font-bold text-white transition
               enabled:hover:brightness-110 disabled:opacity-50
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      >{{ sending ? 'در حال ارسال…' : 'ارسال لینک بازیابی' }}</button>

      <p class="text-center text-[.85rem] text-ink-3">
        یادت آمد؟ <NuxtLink to="/login" class="text-accent hover:underline">برگرد به ورود</NuxtLink>
      </p>
    </form>
  </AuthShell>
</template>
