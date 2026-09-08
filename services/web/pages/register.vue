<script setup lang="ts">
useSeoMeta({ title: 'ساخت حساب در خودروبین', robots: 'noindex' })

const { register, pending, error } = useAuth()

const STEPS = ['ایمیل و رمز', 'چند سؤال کوتاه']

/**
 * Six fields on one screen is a form people leave, so registration is split in
 * two. The whole answer set lives in one object that neither step owns, so
 * going back costs nothing: the second step's inputs unmount, their values do
 * not. Losing typed answers to a back button loses the signup itself.
 */
const form = reactive({
  email: '', password: '', confirm: '',
  phone: '', foundVia: '', foundViaOther: '',
  useCase: '', useCaseOther: '',
  name: '', marketing: false,
})
const touched = reactive({
  email: false, password: false, confirm: false,
  phone: false, foundVia: false, useCase: false,
})

const step = ref(1)
const done = ref<string | null>(null)

// The live checklist is the policy, and on this page it gates. Same three rules
// the Go service enforces, defined once in usePasswordRules.
const { satisfied: passwordOk } = usePasswordRules(() => form.password)

const emailOk = computed(() => /.+@.+\..+/.test(form.email.trim()))
// Errors appear once a field has been left or the step has been submitted,
// never while it is being typed into. Telling someone their email is invalid at
// the third character is technically correct and completely useless.
const emailError = computed(() => (touched.email && !emailOk.value ? 'ایمیل معتبر وارد کن.' : null))

const passwordError = computed(() => {
  if (!touched.password || passwordOk.value) return null
  return form.password.length === 0 ? 'رمز عبور لازم است.' : 'هنوز همه‌ی شرط‌های زیر را ندارد.'
})

const confirmOk = computed(() => form.password.length > 0 && form.confirm === form.password)
const confirmError = computed(() => {
  if (!touched.confirm || confirmOk.value) return null
  return form.confirm.length === 0
    ? 'رمز را یک بار دیگر بنویس.'
    : 'با رمز بالا یکی نیست. با دکمه‌ی چشم می‌توانی هر دو را ببینی.'
})

/**
 * Iranian mobile numbers arrive in five shapes and three scripts.
 *
 *   ۰۹۱۲۳۴۵۶۷۸۹   9123456789   09123456789   +989123456789   00989123456789
 *
 * All of them are the same number, so all of them are accepted and normalised
 * to one form before anything is sent. Refusing a number because of the country
 * code the person's own phone shows them is the kind of validation that teaches
 * people the form is broken.
 */
function toLatinDigits(s: string): string {
  return s
    .replace(/[۰-۹]/g, d => String(d.charCodeAt(0) - 0x06F0)) // Persian
    .replace(/[٠-٩]/g, d => String(d.charCodeAt(0) - 0x0660)) // Arabic-Indic
}

function normalisePhone(raw: string): string | null {
  const digits = toLatinDigits(raw).replace(/[\s()‐-―.-]/g, '')
  // Every Iranian mobile is a 9 followed by nine digits. Everything that can
  // come before it is a way of writing the country code, including none at all.
  const m = /^(?:\+98|0098|98|0)?(9\d{9})$/.exec(digits)
  return m ? `+98${m[1]}` : null
}

const phone = computed(() => normalisePhone(form.phone))
// Names the shape rather than only refusing. "Invalid" tells someone their
// number is wrong; this tells them what a right one looks like.
const phoneError = computed(() => {
  if (!touched.phone || phone.value) return null
  return form.phone.trim().length === 0
    ? 'شماره‌ی موبایل لازم است.'
    : 'شماره‌ی موبایل ایرانی بنویس: با ۰۹ شروع شود و ۱۱ رقم باشد. شکل بین‌المللی هم قبول است.'
})

const FOUND_VIA = [
  { value: 'search', label: 'جست‌وجوی گوگل' },
  { value: 'friend', label: 'معرفی دوست' },
  { value: 'telegram', label: 'تلگرام' },
  { value: 'social', label: 'شبکه‌های اجتماعی' },
  { value: 'torob', label: 'ترب' },
]

const USE_CASE = [
  { value: 'buy', label: 'می‌خواهم ماشین بخرم' },
  { value: 'sell', label: 'می‌خواهم ماشین بفروشم' },
  { value: 'watch', label: 'قیمت‌ها را دنبال می‌کنم' },
  { value: 'research', label: 'تحقیق بازار' },
  { value: 'work', label: 'کارم است؛ نمایشگاه یا واسطه' },
]

const chosen = (value: string, other: string) =>
  value !== '' && (value !== 'other' || other.trim().length > 0)

const foundViaError = computed(() =>
  touched.foundVia && !chosen(form.foundVia, form.foundViaOther) ? 'یکی را انتخاب کن.' : null)
const useCaseError = computed(() =>
  touched.useCase && !chosen(form.useCase, form.useCaseOther) ? 'یکی را انتخاب کن.' : null)

const step1Ok = computed(() => emailOk.value && passwordOk.value && confirmOk.value)
const step2Ok = computed(() =>
  !!phone.value && chosen(form.foundVia, form.foundViaOther) && chosen(form.useCase, form.useCaseOther))

// Focus follows the step. Without this the keyboard lands nowhere when the
// button underneath it disappears, and a screen reader never hears that the
// page changed at all.
const panel = ref<HTMLElement | null>(null)
watch(step, async () => {
  await nextTick()
  panel.value?.focus()
})

function next() {
  touched.email = touched.password = touched.confirm = true
  if (step1Ok.value) step.value = 2
}

async function submit() {
  touched.phone = touched.foundVia = touched.useCase = true
  // A step-1 field can still be invalid if it was edited and the person came
  // forward again. Send them back to it rather than failing at the server.
  if (!step1Ok.value) { step.value = 1; return }
  if (!step2Ok.value) return

  // One request, at the end. An account created halfway and finished by a
  // second call is an account that exists in a state nobody designed.
  const msg = await register(form.email.trim(), form.password, {
    name: form.name.trim() || undefined,
    phone: phone.value!,
    foundVia: form.foundVia,
    foundViaOther: form.foundVia === 'other' ? form.foundViaOther.trim() : undefined,
    useCase: form.useCase,
    useCaseOther: form.useCase === 'other' ? form.useCaseOther.trim() : undefined,
    marketingConsent: form.marketing,
  })
  if (msg) done.value = msg
}

// Resend, because "no email arrived" is the single most common way a signup
// dies, and it is usually spam filtering rather than a bug the user can see.
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
  form.password = form.confirm = ''
  touched.password = touched.confirm = false
  step.value = 1
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
      <p class="mono-nums mt-1 text-center text-[.8rem] text-ink-2">{{ form.email }}</p>

      <div class="mt-5 border-t border-good/20 pt-4 text-center text-[.82rem] leading-8 text-ink-2">
        <p>نرسید؟ پوشه‌ی <span class="text-ink">اسپم</span> را نگاه کن. دامنه‌ی تازه معمولاً آنجا می‌افتد.</p>
        <div class="mt-3 flex flex-wrap justify-center gap-2">
          <button
            type="button"
            :disabled="resending || resent"
            class="min-h-[44px] rounded-full border border-white/[.12] px-4 text-[.82rem] text-ink-2
                   transition enabled:hover:text-ink disabled:opacity-50
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            @click="resend"
          >{{ resent ? 'دوباره فرستاده شد' : resending ? 'در حال ارسال…' : 'ارسال دوباره‌ی لینک' }}</button>
          <button
            type="button"
            class="min-h-[44px] rounded-full border border-white/[.12] px-4 text-[.82rem] text-ink-2 transition hover:text-ink
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            @click="startOver"
          >ایمیل را عوض کن</button>
        </div>
      </div>

      <NuxtLink to="/login" class="mt-5 block text-center text-[.85rem] text-accent hover:underline">
        رفتن به ورود
      </NuxtLink>
    </div>

    <form v-else class="grid gap-5" novalidate @submit.prevent="step === 1 ? next() : submit()">
      <SignupSteps :current="step" :steps="STEPS" />

      <div ref="panel" tabindex="-1" class="grid gap-4 outline-none">
        <template v-if="step === 1">
          <!-- focusout, not blur: the listener lands on the component's root
               element and blur does not bubble to it, so a @blur here would
               never fire and the field would only ever be validated on submit. -->
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
            @focusout="touched.email = true"
          />

          <div @focusout="touched.password = true">
            <!--
              The old hint recommended «یک عبارت فارسی طولانی» and stopped there.
              Persian script has no uppercase letters, so under the capital-letter
              rule that advice was impossible to follow: someone who did exactly
              what it said still could not submit, and nothing on the page said
              why. The phrase advice stays, because length is what actually
              resists guessing, but the two characters the policy also needs are
              now named. Do not shorten this back to "write a long Persian
              phrase"; that is what created the contradiction. It describes the
              recipe rather than showing a literal example password, because a
              Persian string ending in a Latin letter and a symbol is reordered
              by the bidi algorithm and displays in an order nobody typed.
            -->
            <PasswordField
              id="r-pass"
              v-model="form.password"
              label="رمز عبور"
              autocomplete="new-password"
              :error="passwordError"
              hint="یک عبارت فارسی طولانی بنویس، بعد یک حرف بزرگ لاتین و یک نماد هم به آن اضافه کن."
            />
            <PasswordRules :password="form.password" />
          </div>

          <!-- The reveal toggle prevents typos at the source; this catches the
               one it cannot, a password typed correctly on the wrong keyboard
               layout, which looks right and is not. -->
          <PasswordField
            id="r-confirm"
            v-model="form.confirm"
            label="تکرار رمز عبور"
            autocomplete="new-password"
            :error="confirmError"
            @focusout="touched.confirm = true"
          />
        </template>

        <template v-else>
          <FormField
            id="r-phone"
            v-model="form.phone"
            label="شماره‌ی موبایل"
            type="tel"
            dir="ltr"
            autocomplete="tel"
            placeholder="09123456789"
            :error="phoneError"
            required
            @focusout="touched.phone = true"
          />

          <SignupChips
            v-model="form.foundVia"
            v-model:other="form.foundViaOther"
            name="found-via"
            legend="از کجا ما را پیدا کردی؟"
            :options="FOUND_VIA"
            other-label="جای دیگر"
            other-field-label="کجا؟"
            :error="foundViaError"
          />

          <SignupChips
            v-model="form.useCase"
            v-model:other="form.useCaseOther"
            name="use-case"
            legend="برای چه کاری می‌خواهی‌اش؟"
            :options="USE_CASE"
            other-label="چیز دیگر"
            other-field-label="برای چه کاری؟"
            :error="useCaseError"
          />

          <FormField
            id="r-name"
            v-model="form.name"
            label="نام"
            autocomplete="name"
            hint="فقط برای اینکه ایمیل‌ها با اسمت شروع شود."
          />

          <!-- Unchecked, and nothing on this page ever ticks it. A pre-ticked box
               harvests an address rather than a permission, and the list it
               builds is the one that gets reported as spam, which costs
               deliverability for the price alerts people actually asked for. The
               label wraps the box so the whole row is the hit target, not an
               18px square. -->
          <label
            class="flex cursor-pointer items-start gap-3 rounded-xl border border-white/[.09] bg-surface/60 px-4 py-3.5
                   transition hover:border-white/[.16] focus-within:border-focus/55"
          >
            <input
              v-model="form.marketing"
              type="checkbox"
              class="mt-1 size-[18px] shrink-0 accent-accent
                     focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            >
            <span class="text-[.83rem] leading-7 text-ink-2">
              خبرنامه‌ی خودروبین را هم برایم بفرست.
              <span class="mt-0.5 block text-[.78rem] leading-6 text-ink-2/85">
                گاهی درباره‌ی روند قیمت بازار و قابلیت‌های تازه. لغو با یک کلیک، از پای هر ایمیل.
              </span>
            </span>
          </label>
        </template>
      </div>

      <div
        v-if="error"
        role="alert"
        class="flex items-start gap-2 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] leading-7 text-accent"
      >
        <span aria-hidden="true">⚠</span><span>{{ error }}</span>
      </div>

      <!-- Never disabled for validity, only while a request is in flight. A
           button that is dead because something above it is wrong, and does not
           say which thing, is the worst version of this form: pressing it is how
           people ask what is missing. -->
      <div class="flex flex-wrap gap-3">
        <button
          v-if="step === 2"
          type="button"
          class="min-h-[48px] rounded-xl border border-white/[.14] px-5 text-[.9rem] font-bold text-ink-2 transition
                 hover:border-white/25 hover:text-ink
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          @click="step = 1"
        >بازگشت</button>

        <button
          type="submit"
          :disabled="pending"
          class="min-h-[48px] flex-1 rounded-xl bg-accent px-5 font-bold text-white transition
                 enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50
                 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >{{ step === 1 ? 'ادامه' : pending ? 'در حال ساخت حساب…' : 'ساخت حساب' }}</button>
      </div>

      <p class="text-center text-[.85rem] text-ink-2">
        حساب داری؟
        <NuxtLink to="/login" class="text-accent hover:underline">وارد شو</NuxtLink>
      </p>

      <!-- The old line promised no marketing email at all, unconditionally. The
           checkbox above made that promise false the moment it shipped, so it is
           narrowed rather than deleted: the guarantee people came here for, that
           the address is for their alerts and not for a list, still holds, and
           the one case where it does not is the one they chose themselves. It
           says «خودت خواسته باشی» rather than pointing at the tick, because this
           note is shown on both steps and the tick only exists on the second. -->
      <p class="mt-1 border-t border-white/[.07] pt-4 text-center text-[.78rem] leading-7 text-ink-2">
        ایمیل و شماره‌ات برای ورود، هشدار قیمت و پشتیبانی است. هیچ ایمیل تبلیغاتی‌ای نمی‌فرستیم مگر خودت
        خواسته باشی، و لغوش هم یک کلیک است از پای هر ایمیل.
      </p>
    </form>
  </AuthShell>
</template>
