<script setup lang="ts">
useSeoMeta({ title: 'ورود — خودروبین', robots: 'noindex' })

const { login, pending, error } = useAuth()
const route = useRoute()

const form = reactive({ email: '', password: '' })
const touched = reactive({ email: false })

// Validation appears only after a field is left, never while it is being typed
// into. Telling someone their email is invalid at the third character is
// technically correct and completely useless.
const emailError = computed(() =>
  touched.email && !/.+@.+\..+/.test(form.email.trim()) ? 'ایمیل معتبر وارد کن.' : null)

async function submit() {
  touched.email = true
  if (emailError.value || !form.password) return

  if (await login(form.email.trim(), form.password)) {
    const next = typeof route.query.next === 'string' ? route.query.next : '/account'
    // Same-origin paths only. An open redirect turns a login page into a
    // phishing hop that borrows this domain's credibility.
    await navigateTo(next.startsWith('/') && !next.startsWith('//') ? next : '/account')
  }
}
</script>

<template>
  <AuthShell title="ورود">
    <form class="grid gap-4" novalidate @submit.prevent="submit">
      <FormField
        id="l-email"
        v-model="form.email"
        label="ایمیل"
        type="email"
        dir="ltr"
        autocomplete="username"
        placeholder="you@example.com"
        :error="emailError"
        required
        autofocus
        @blur="touched.email = true"
      />

      <div>
        <PasswordField
          id="l-pass"
          v-model="form.password"
          label="رمز عبور"
          autocomplete="current-password"
        />
        <!-- Directly beneath the field it relates to. Someone reaching for this
             has just failed to remember the thing immediately above it. -->
        <NuxtLink
          :to="form.email.trim() ? `/forgot?email=${encodeURIComponent(form.email.trim())}` : '/forgot'"
          class="mt-1.5 inline-block text-[.8rem] text-ink-3 transition hover:text-ink-2"
        >رمزت را فراموش کرده‌ای؟</NuxtLink>
      </div>

      <!-- Names a way forward rather than only a fault, and nothing is cleared:
           only the password needs retyping. -->
      <div
        v-if="error"
        role="alert"
        class="flex items-start gap-2 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] leading-7 text-accent"
      >
        <span aria-hidden="true">⚠</span>
        <span>
          {{ error }}
          <NuxtLink to="/forgot" class="underline underline-offset-2">بازیابی رمز</NuxtLink>
          یا دوباره امتحان کن.
        </span>
      </div>

      <button
        type="submit"
        :disabled="pending"
        class="min-h-[48px] rounded-xl bg-accent px-5 font-bold text-white transition
               enabled:hover:brightness-110 disabled:opacity-50
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      >{{ pending ? 'در حال ورود…' : 'ورود' }}</button>

      <p class="text-center text-[.85rem] text-ink-3">
        حساب نداری؟
        <NuxtLink to="/register" class="text-accent hover:underline">ساخت حساب</NuxtLink>
      </p>

      <p class="mt-2 border-t border-white/[.07] pt-4 text-center text-[.76rem] leading-7 text-ink-3">
        جست‌وجو به حساب نیاز ندارد — ورود فقط برای ذخیره‌ی جست‌وجو و هشدار قیمت است.
      </p>
    </form>
  </AuthShell>
</template>
