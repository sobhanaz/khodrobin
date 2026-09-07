<script setup lang="ts">
useSeoMeta({ title: 'ورود — خودروبین', robots: 'noindex' })

const { login, pending, error } = useAuth()
const route = useRoute()
const form = reactive({ email: '', password: '' })

async function submit() {
  if (await login(form.email.trim(), form.password)) {
    const next = typeof route.query.next === 'string' ? route.query.next : '/account'
    // Only same-origin paths: an open redirect turns a login page into a
    // phishing hop that borrows this domain's credibility.
    await navigateTo(next.startsWith('/') && !next.startsWith('//') ? next : '/account')
  }
}

const field = 'w-full rounded-xl border border-white/[.12] bg-surface px-4 py-3 text-ink outline-none ' +
  'transition placeholder:text-ink-3 focus:border-accent/55 focus:shadow-[0_0_0_4px_rgba(255,46,77,.12)]'
</script>

<template>
  <AuthShell title="ورود">
    <form class="grid gap-4" @submit.prevent="submit">
      <div>
        <label for="l-email" class="mb-1.5 block text-[.85rem] text-ink-2">ایمیل</label>
        <input id="l-email" v-model="form.email" type="email" dir="ltr" :class="field"
               autocomplete="email" required>
      </div>
      <div>
        <label for="l-pass" class="mb-1.5 block text-[.85rem] text-ink-2">رمز عبور</label>
        <input id="l-pass" v-model="form.password" type="password" :class="field"
               autocomplete="current-password" required>
      </div>

      <p v-if="error" class="rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] text-accent">
        {{ error }}
      </p>

      <button type="submit" :disabled="pending"
        class="rounded-xl bg-accent px-5 py-3 font-bold text-white transition
               enabled:hover:brightness-110 disabled:opacity-40">
        {{ pending ? 'در حال ورود…' : 'ورود' }}
      </button>

      <div class="flex items-center justify-between text-[.85rem]">
        <NuxtLink to="/register" class="text-accent hover:underline">ساخت حساب</NuxtLink>
        <NuxtLink to="/forgot" class="text-ink-3 hover:text-ink-2">رمزت را فراموش کردی؟</NuxtLink>
      </div>
    </form>
  </AuthShell>
</template>
