<script setup lang="ts">
useSeoMeta({ title: 'ثبت‌نام — خودروبین', robots: 'noindex' })

const { register, pending, error } = useAuth()

const form = reactive({ email: '', password: '', name: '' })
const touched = reactive({ email: false, password: false })
const done = ref<string | null>(null)

// Counted in runes, the way the server counts it. Counting bytes would let a
// shorter Persian password through than an English one.
const passLen = computed(() => [...form.password].length)

const emailError = computed(() =>
  touched.email && !/.+@.+\..+/.test(form.email.trim()) ? 'ایمیل معتبر وارد کن.' : null)
const passError = computed(() =>
  touched.password && passLen.value > 0 && passLen.value < 10
    ? `${10 - passLen.value} نویسه‌ی دیگر لازم است.` : null)

const canSubmit = computed(() => /.+@.+\..+/.test(form.email.trim()) && passLen.value >= 10)

async function submit() {
  touched.email = touched.password = true
  if (!canSubmit.value) return
  const msg = await register(form.email.trim(), form.password, form.name.trim() || undefined)
  if (msg) done.value = msg
}

// Resend, because "no email arrived" is the single most common way a signup
// dies — and it is usually spam filtering, not a bug the user can see.
const resending = ref(false)
const resent = ref(false)
async function resend() {
  if (resending.value) return
  resending.value = true
  try {
    await $fetch(apiUrl('/api/auth/resend'), { method: 'POST', body: { email: form.email.trim() } })
    resent.value = true
  } catch {
    resent.value = true // The endpoint answers the same either way by design.
  } finally {
    resending.value = false
  }
}

function startOver() {
  done.value = null
  resent.value = false
  form.password = ''
}
</script>

<template>
  <AuthShell
    title="ساخت حساب"
    subtitle="برای ذخیره‌ی جست‌وجو و هشدار قیمت. جست‌وجو بدون حساب هم کامل کار می‌کند."
  >
    <!-- Success states the next action explicitly. "Account created" is not an
         instruction; "check your inbox" is. -->
    <div v-if="done" class="rounded-2xl border border-good/30 bg-good/[.12] p-6">
      <div class="text-center font-bold text-good">ایمیلت را چک کن</div>
      <p class="mt-2 text-center text-[.88rem] leading-8 text-ink-2">{{ done }}</p>
      <p class="mt-1 text-center font-mono text-[.8rem] text-ink-3" dir="ltr">{{ form.email }}</p>

      <div class="mt-5 border-t border-good/20 pt-4 text-center text-[.82rem] leading-8 text-ink-2">
        <p>نرسید؟ پوشه‌ی <span class="text-ink">اسپم</span> را نگاه کن — دامنه‌ی تازه معمولاً آنجا می‌افتد.</p>
        <div class="mt-3 flex flex-wrap justify-center gap-2">
          <button
            type="button"
            :disabled="resending || resent"
            class="min-h-[44px] rounded-full border border-white/[.12] px-4 text-[.82rem] text-ink-2
                   transition enabled:hover:text-ink disabled:opacity-50"
            @click="resend"
          >{{ resent ? 'دوباره فرستاده شد' : resending ? 'در حال ارسال…' : 'ارسال دوباره‌ی لینک' }}</button>
          <button
            type="button"
            class="min-h-[44px] rounded-full border border-white/[.12] px-4 text-[.82rem] text-ink-3 transition hover:text-ink"
            @click="startOver"
          >ایمیل را عوض کن</button>
        </div>
      </div>

      <NuxtLink to="/login" class="mt-5 block text-center text-[.85rem] text-accent hover:underline">
        رفتن به ورود
      </NuxtLink>
    </div>

    <form v-else class="grid gap-4" novalidate @submit.prevent="submit">
      <FormField
        id="r-email"
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

      <PasswordField
        id="r-pass"
        v-model="form.password"
        label="رمز عبور"
        autocomplete="new-password"
        :error="passError"
        hint="حداقل ۱۰ نویسه. یک عبارت فارسی طولانی از یک رمز کوتاهِ پیچیده امن‌تر است."
      />

      <!-- Progress toward the one rule that exists, not a scolding strength
           meter. There is no confirm-password field: the visibility toggle
           prevents typos at the source, and retyping an invisible password
           causes more errors than it catches. -->
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

      <!-- Optional and marked so. Every required field is a reason to abandon,
           so only the two that are genuinely needed are required. -->
      <FormField
        id="r-name"
        v-model="form.name"
        label="نام"
        autocomplete="name"
        hint="فقط برای اینکه ایمیل‌ها با اسمت شروع شود."
      />

      <div
        v-if="error"
        role="alert"
        class="flex items-start gap-2 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] leading-7 text-accent"
      >
        <span aria-hidden="true">⚠</span><span>{{ error }}</span>
      </div>

      <button
        type="submit"
        :disabled="!canSubmit || pending"
        class="min-h-[48px] rounded-xl bg-accent px-5 font-bold text-white transition
               enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      >{{ pending ? 'در حال ساخت حساب…' : 'ساخت حساب' }}</button>

      <p class="text-center text-[.85rem] text-ink-3">
        حساب داری؟
        <NuxtLink to="/login" class="text-accent hover:underline">وارد شو</NuxtLink>
      </p>

      <p class="mt-2 border-t border-white/[.07] pt-4 text-center text-[.76rem] leading-7 text-ink-3">
        فقط ایمیلت را نگه می‌داریم تا هشدار قیمت بفرستیم. هیچ ایمیل تبلیغاتی‌ای نمی‌فرستیم.
      </p>
    </form>
  </AuthShell>
</template>
