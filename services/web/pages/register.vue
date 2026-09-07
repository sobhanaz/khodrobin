<script setup lang="ts">
useSeoMeta({ title: 'ثبت‌نام — خودروبین', robots: 'noindex' })

const { register, pending, error } = useAuth()
const form = reactive({ email: '', password: '', name: '' })
const done = ref<string | null>(null)

// Length only, and counted the way the server counts it. Composition rules push
// people toward "Password1!" and away from a long Persian phrase.
const passLen = computed(() => [...form.password].length)
const canSubmit = computed(() => /.+@.+\..+/.test(form.email) && passLen.value >= 10)

async function submit() {
  if (!canSubmit.value) return
  const msg = await register(form.email.trim(), form.password, form.name.trim() || undefined)
  if (msg) done.value = msg
}

const field = 'w-full rounded-xl border border-white/[.12] bg-surface px-4 py-3 text-ink outline-none ' +
  'transition placeholder:text-ink-3 focus:border-accent/55 focus:shadow-[0_0_0_4px_rgba(255,46,77,.12)]'
</script>

<template>
  <AuthShell
    title="ساخت حساب"
    subtitle="برای ذخیره‌ی جست‌وجو و هشدار قیمت. جست‌وجو بدون حساب هم کامل کار می‌کند."
  >
    <div v-if="done" class="rounded-2xl border border-good/30 bg-good/[.12] p-5 text-center">
      <div class="font-bold text-good">ایمیلت را چک کن</div>
      <p class="mt-2 text-[.88rem] leading-8 text-ink-2">{{ done }}</p>
      <NuxtLink to="/login" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">رفتن به ورود</NuxtLink>
    </div>

    <form v-else class="grid gap-4" @submit.prevent="submit">
      <div>
        <label for="r-email" class="mb-1.5 block text-[.85rem] text-ink-2">ایمیل</label>
        <input id="r-email" v-model="form.email" type="email" dir="ltr" :class="field"
               autocomplete="email" placeholder="you@example.com" required>
      </div>
      <div>
        <label for="r-name" class="mb-1.5 block text-[.85rem] text-ink-2">نام <span class="text-ink-3">(اختیاری)</span></label>
        <input id="r-name" v-model="form.name" :class="field" autocomplete="name">
      </div>
      <div>
        <label for="r-pass" class="mb-1.5 block text-[.85rem] text-ink-2">رمز عبور</label>
        <input id="r-pass" v-model="form.password" type="password" :class="field"
               autocomplete="new-password" required>
        <div class="mt-1.5 flex items-center gap-2">
          <span class="h-1 flex-1 overflow-hidden rounded-full bg-surface-2">
            <span class="block h-full rounded-full transition-all duration-300"
                  :class="passLen >= 10 ? 'bg-good' : 'bg-warn'"
                  :style="{ width: `${Math.min(100, passLen * 10)}%` }" />
          </span>
          <span class="font-mono text-[.7rem] text-ink-3" dir="ltr">{{ passLen }}/10</span>
        </div>
        <p class="mt-1.5 text-[.76rem] leading-6 text-ink-3">
          حداقل ۱۰ نویسه. یک عبارت فارسی طولانی از یک رمز کوتاهِ پیچیده امن‌تر است.
        </p>
      </div>

      <p v-if="error" class="rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] text-accent">
        {{ error }}
      </p>

      <button type="submit" :disabled="!canSubmit || pending"
        class="rounded-xl bg-accent px-5 py-3 font-bold text-white transition
               enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40">
        {{ pending ? 'در حال ساخت…' : 'ساخت حساب' }}
      </button>

      <p class="text-center text-[.85rem] text-ink-3">
        حساب داری؟ <NuxtLink to="/login" class="text-accent hover:underline">وارد شو</NuxtLink>
      </p>
    </form>
  </AuthShell>
</template>
