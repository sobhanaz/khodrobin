<script setup lang="ts">
useSeoMeta({
  title: 'تماس با خودروبین',
  description: 'عددی اشتباه بود، منبعی جا مانده، یا چیزی خراب است؟ بنویس.',
})

const form = reactive({ name: '', email: '', subject: '', body: '' })
const touched = reactive({ name: false, email: false, body: false })
const sending = ref(false)
const sent = ref(false)
const failure = ref<string | null>(null)

// Counted in code points, like the server does with utf8.RuneCountInString.
// String.length counts UTF-16 units, so a message of emoji would pass here and
// be rejected there, with the field insisting it was fine.
const bodyLength = computed(() => [...form.body.trim()].length)

/**
 * The same three rules the API enforces, said before the round trip rather
 * than after it. They are deliberately a mirror and not a stricter version: a
 * form that rejects what the server would have accepted is its own bug, and
 * one that accepts what the server rejects sends people a generic failure for
 * something the page could have named.
 */
const errors = computed(() => ({
  name: form.name.trim() ? null : 'نامت را بنویس.',
  email: /.+@.+\..+/.test(form.email.trim()) ? null : 'یک ایمیل معتبر بنویس، وگرنه راهی برای جواب‌دادن نداریم.',
  body: bodyLength.value < 10
    ? 'پیام باید دست‌کم ۱۰ نویسه باشد.'
    : bodyLength.value > 5000 ? 'پیام از ۵۰۰۰ نویسه بیشتر شده.' : null,
}))

// Shown after the field is left or after a submit attempt, never while someone
// is still typing the first letter of their name.
const shown = computed(() => ({
  name: touched.name ? errors.value.name : null,
  email: touched.email ? errors.value.email : null,
  body: touched.body ? errors.value.body : null,
}))

async function submit() {
  if (sending.value) return
  touched.name = touched.email = touched.body = true
  const firstBad = (['name', 'email', 'body'] as const).find(k => errors.value[k])
  if (firstBad) {
    // The error is under the field, so send the person to the field.
    document.getElementById(`c-${firstBad}`)?.focus()
    return
  }
  sending.value = true
  failure.value = null
  try {
    await $fetch(apiUrl('/api/auth/contact'), {
      method: 'POST',
      body: {
        name: form.name.trim(),
        email: form.email.trim(),
        subject: form.subject.trim(),
        body: form.body.trim(),
      },
    })
    sent.value = true
  } catch (e: any) {
    failure.value = e?.data?.error ?? 'ارسال نشد. یک بار دیگر امتحان کن.'
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <main class="mx-auto max-w-[980px] px-5 py-16">
    <h1 class="text-title">تماس با ما</h1>
    <p class="mt-4 max-w-[44ch] text-lead text-ink-2">
      عددی اشتباه بود، منبعی جا مانده، یا چیزی خراب است؟ بنویس.
    </p>

    <div class="mt-12 grid items-start gap-x-14 gap-y-10 lg:grid-cols-[minmax(0,1fr)_16rem]">
      <div>
        <div
          v-if="sent"
          role="status"
          class="rounded-2xl border border-good/30 bg-good/[.1] p-6"
        >
          <h2 class="text-[1.05rem] font-bold text-good">پیامت ثبت شد</h2>
          <p class="mt-3 max-w-[44ch] text-[.92rem] leading-8 text-ink-2">
            پیام اول در پایگاه‌داده ذخیره می‌شود و بعد ایمیل، تا اگر سرویس ایمیل قطع بود گم نشود.
            اگر جواب لازم داشته باشد، به همان نشانی‌ای که نوشتی می‌نویسیم.
          </p>
          <NuxtLink
            to="/"
            class="mt-5 inline-flex min-h-[44px] items-center rounded-xl bg-accent px-5 text-[.85rem] font-bold text-white transition hover:brightness-110
                   focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >بازگشت به جست‌وجو</NuxtLink>
        </div>

        <form v-else class="grid max-w-[38rem] gap-5" novalidate @submit.prevent="submit">
          <FormField
            id="c-name"
            v-model="form.name"
            label="نام"
            autocomplete="name"
            required
            :error="shown.name"
            @focusout="touched.name = true"
          />

          <FormField
            id="c-email"
            v-model="form.email"
            label="ایمیل"
            type="email"
            dir="ltr"
            autocomplete="email"
            required
            hint="فقط برای جواب‌دادن به همین پیام استفاده می‌شود."
            :error="shown.email"
            @focusout="touched.email = true"
          />

          <FormField
            id="c-subject"
            v-model="form.subject"
            label="موضوع"
          />

          <div>
            <label for="c-body" class="mb-1.5 block text-[.85rem] text-ink-2">پیام</label>
            <textarea
              id="c-body"
              v-model="form.body"
              rows="7"
              required
              :aria-invalid="shown.body ? 'true' : undefined"
              :aria-describedby="shown.body ? 'c-body-error' : 'c-body-count'"
              class="w-full rounded-xl border bg-surface px-4 py-3 leading-8 text-ink outline-none transition
                     focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
              :class="shown.body
                ? 'border-accent/60 focus:shadow-[0_0_0_4px_rgba(255,46,77,.14)] focus-visible:outline-accent'
                : 'border-white/[.12] focus:border-focus/60 focus:shadow-[0_0_0_4px_rgba(107,116,230,.16)] focus-visible:outline-focus'"
              @blur="touched.body = true"
            />
            <p
              v-if="shown.body"
              id="c-body-error"
              role="alert"
              class="mt-1.5 flex items-start gap-1.5 text-[.8rem] leading-6 text-accent"
            >
              <span aria-hidden="true">⚠</span><span>{{ shown.body }}</span>
            </p>
            <p v-else id="c-body-count" class="mt-1.5 text-[.76rem] leading-6 text-ink-2">
              دست‌کم ۱۰ نویسه، حداکثر ۵۰۰۰.
            </p>
          </div>

          <p
            v-if="failure"
            role="alert"
            class="rounded-xl border border-accent/30 bg-accent/[.1] px-4 py-3 text-[.85rem] leading-7 text-accent"
          >{{ failure }}</p>

          <div>
            <button
              type="submit"
              :disabled="sending"
              class="min-h-[44px] rounded-xl bg-accent px-6 font-bold text-white transition
                     enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40
                     focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
            >{{ sending ? 'در حال ارسال…' : 'ارسال پیام' }}</button>
          </div>
        </form>
      </div>

      <aside class="text-[.86rem] leading-8 text-ink-2 lg:border-s lg:border-white/[.07] lg:ps-8">
        <h2 class="text-[.9rem] font-bold text-ink-2">چه چیزی بیشتر به درد می‌خورد</h2>
        <ul class="mt-3 space-y-3">
          <li>عددی که غلط به نظر می‌رسد، همراه با نشانی همان کارت.</li>
          <li>سایت آگهی‌ای که هنوز نمی‌خوانیمش.</li>
          <li>خودرویی که زیر مشخصات اشتباه گروه‌بندی شده.</li>
        </ul>
        <p class="mt-6 border-t border-white/[.07] pt-5">
          شماره‌ی فروشنده را نداریم و جمع هم نمی‌کنیم؛ برای تماس باید به آگهی اصلی بروی.
          <NuxtLink
            to="/faq"
            class="text-accent hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
          >بقیه‌ی پرسش‌ها</NuxtLink>.
        </p>
      </aside>
    </div>
  </main>
</template>
