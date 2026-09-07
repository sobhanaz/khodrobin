<script setup lang="ts">
useSeoMeta({ title: 'تأیید ایمیل — خودروبین', robots: 'noindex' })

const route = useRoute()
const state = ref<'working' | 'ok' | 'failed'>('working')
const message = ref('')

onMounted(async () => {
  const token = route.query.token
  if (typeof token !== 'string' || !token) {
    state.value = 'failed'
    message.value = 'لینک نامعتبر است.'
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
</script>

<template>
  <AuthShell title="تأیید ایمیل">
    <div v-if="state === 'working'" class="text-ink-2">در حال بررسی لینک…</div>

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
