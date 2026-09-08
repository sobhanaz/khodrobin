<script setup lang="ts">
useSeoMeta({ title: 'انتخاب رمز تازه در خودروبین', robots: 'noindex' })

const route = useRoute()
const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))

const password = ref('')
const touched = ref(false)
const pending = ref(false)
const error = ref<string | null>(null)
const done = ref(false)

// This screen sets a password, so it enforces the same three rules register
// does. A policy applied on one of the two screens that set a password is not a
// policy, it is a suggestion with a gap in it.
const { satisfied } = usePasswordRules(password)

const passError = computed(() => {
  if (!touched.value || satisfied.value) return null
  return password.value.length === 0 ? 'رمز تازه لازم است.' : 'هنوز همه‌ی شرط‌های زیر را ندارد.'
})

async function submit() {
  touched.value = true
  if (!satisfied.value || !token.value) return
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
           defeat the whole point, so the page says so rather than hiding it. -->
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">
        همه‌ی نشست‌های قبلی بسته شدند. با رمز تازه وارد شو.
      </p>
      <NuxtLink to="/login" class="mt-4 inline-block rounded-xl bg-accent px-4 py-2 text-[.85rem] font-bold text-white">
        ورود
      </NuxtLink>
    </div>

    <form v-else class="grid gap-4" novalidate @submit.prevent="submit">
      <!-- focusout, not blur: the listener lands on the component's root element
           and blur does not bubble to it. -->
      <div @focusout="touched = true">
        <PasswordField
          id="rs-pass"
          v-model="password"
          label="رمز تازه"
          autocomplete="new-password"
          :error="passError"
          hint="یک عبارت فارسی طولانی بنویس، بعد یک حرف بزرگ لاتین و یک نماد هم به آن اضافه کن."
        />
        <!-- The live checklist replaces the old length bar. Two widgets stating
             one policy is one widget too many, and the bar only knew about the
             length rule, so it could show a full green track on a password the
             server would refuse. -->
        <PasswordRules :password="password" />
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
        :disabled="pending"
        class="min-h-[48px] rounded-xl bg-accent px-5 font-bold text-white transition
               enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      >{{ pending ? 'در حال تغییر…' : 'تغییر رمز' }}</button>
    </form>
  </AuthShell>
</template>
