<script setup lang="ts">
useSeoMeta({ title: 'انتخاب رمز تازه — خودروبین', robots: 'noindex' })

const route = useRoute()
const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))

const password = ref('')
const touched = ref(false)
const pending = ref(false)
const error = ref<string | null>(null)
const done = ref(false)

const passLen = computed(() => [...password.value].length)
const passError = computed(() =>
  touched.value && passLen.value > 0 && passLen.value < 10
    ? `${10 - passLen.value} نویسه‌ی دیگر لازم است.` : null)

async function submit() {
  touched.value = true
  if (passLen.value < 10 || !token.value) return
  pending.value = true; error.value = null
  try {
    await $fetch(apiUrl('/api/auth/reset'), {
      method: 'POST', body: { token: token.value, password: password.value },
    })
    done.value = true
  } catch (e: any) {
    error.value = e?.data?.error ?? 'تغییر رمز انجام نشد.'
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <AuthShell title="انتخاب رمز تازه">
    <div v-if="!token" class="rounded-2xl border border-warn/25 bg-warn/[.12] p-6 text-center">
      <div class="font-bold text-warn">لینک نامعتبر است</div>
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        آدرس را کامل از ایمیل کپی کن، یا لینک تازه بگیر.
      </p>
      <NuxtLink to="/forgot" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">
        درخواست لینک تازه
      </NuxtLink>
    </div>

    <div v-else-if="done" class="rounded-2xl border border-good/30 bg-good/[.12] p-6 text-center">
      <div class="font-bold text-good">رمز عوض شد</div>
      <!-- Every other session was revoked server-side. If the reset happened
           because someone else had access, leaving their session alive would
           defeat the whole point — so the page says so rather than hiding it. -->
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        همه‌ی نشست‌های قبلی بسته شدند. با رمز تازه وارد شو.
      </p>
      <NuxtLink to="/login" class="mt-4 inline-block rounded-xl bg-accent px-4 py-2 text-[.85rem] font-bold text-white">
        ورود
      </NuxtLink>
    </div>

    <form v-else class="grid gap-4" novalidate @submit.prevent="submit">
      <PasswordField
        id="rs-pass"
        v-model="password"
        label="رمز تازه"
        autocomplete="new-password"
        :error="passError"
        hint="حداقل ۱۰ نویسه. با دکمه‌ی چشم می‌توانی ببینی چه نوشته‌ای."
      />

      <div class="-mt-1 flex items-center gap-2" aria-hidden="true">
        <span class="h-1 flex-1 overflow-hidden rounded-full bg-surface-2">
          <span
            class="block h-full rounded-full transition-all duration-300"
            :class="passLen >= 10 ? 'bg-good' : 'bg-warn'"
            :style="{ width: `${Math.min(100, passLen * 10)}%` }"
          />
        </span>
        <span class="font-mono text-[.7rem]" :class="passLen >= 10 ? 'text-good' : 'text-ink-3'" dir="ltr">
          {{ passLen >= 10 ? '✓' : `${passLen}/10` }}
        </span>
      </div>

      <div
        v-if="error"
        role="alert"
        class="flex items-start gap-2 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] leading-7 text-accent"
      >
        <span aria-hidden="true">⚠</span>
        <span>{{ error }} <NuxtLink to="/forgot" class="underline underline-offset-2">لینک تازه بگیر</NuxtLink>.</span>
      </div>

      <button
        type="submit"
        :disabled="passLen < 10 || pending"
        class="min-h-[48px] rounded-xl bg-accent px-5 font-bold text-white transition
               enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      >{{ pending ? 'در حال تغییر…' : 'تغییر رمز' }}</button>
    </form>
  </AuthShell>
</template>
