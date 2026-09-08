<script setup lang="ts">
withDefaults(defineProps<{
  /** Footer placement: same form, no heading and no benefit list. */
  compact?: boolean
}>(), { compact: false })

// Two instances live on this page (section and footer), so the label/input
// pairing cannot use a literal id — duplicate ids silently break the second
// label and a screen reader lands on an unnamed field.
const uid = useId()

const email = ref('')
const touched = ref(false)
const state = ref<'idle' | 'sending' | 'sent' | 'error'>('idle')
const sentTo = ref('')

const valid = computed(() => /.+@.+\..+/.test(email.value.trim()))
const fieldError = computed(() =>
  touched.value && email.value.length > 0 && !valid.value ? 'ایمیل معتبر وارد کن.' : null)

async function submit() {
  touched.value = true
  if (!valid.value || state.value === 'sending') return
  state.value = 'sending'
  const addr = email.value.trim()
  try {
    await $fetch(apiUrl('/api/auth/subscribe'), {
      method: 'POST',
      body: { email: addr, source: 'landing' },
    })
    sentTo.value = addr
    state.value = 'sent'
  } catch {
    // The endpoint answers 202 for every address on purpose, so a failure here
    // is the network or the service — never "this address is already taken".
    // Saying anything about the address would rebuild the enumeration oracle
    // the 202 exists to remove.
    state.value = 'error'
  }
}

function startOver() {
  state.value = 'idle'
  touched.value = false
  email.value = ''
}
</script>

<template>
  <div class="glass rounded-2xl p-5 sm:p-7">
    <div v-if="!compact">
      <h2 class="text-[clamp(1.25rem,3.2vw,1.6rem)] font-black tracking-tight">
        وقتی قیمت خودرویت تکان خورد، خبردار شو
      </h2>
      <!-- What they are signing up for, before the field, not in fine print
           after it. A list someone did not know they joined is a spam report. -->
      <ul class="mt-3 grid gap-1.5 text-[.88rem] leading-8 text-ink-2">
        <li class="flex gap-2">
          <span class="text-good" aria-hidden="true">•</span>
          <span><b class="font-bold text-ink">هشدار قیمت</b> — وقتی میانه‌ی بازار برای خودروهایی که دنبالشان هستی جابه‌جا شد.</span>
        </li>
        <li class="flex gap-2">
          <span class="text-good" aria-hidden="true">•</span>
          <span><b class="font-bold text-ink">خبر قابلیت تازه</b> — وقتی منبع یا امکان جدیدی اضافه شد.</span>
        </li>
      </ul>
      <p class="mt-2 text-[.82rem] leading-7 text-ink-3">
        همین دو تا. نه چیز دیگری، نه هفته‌ای چند بار.
      </p>
    </div>

    <form
      v-if="state !== 'sent'"
      class="mt-4 flex flex-col gap-2 sm:flex-row sm:items-start"
      novalidate
      @submit.prevent="submit"
    >
      <div class="flex-1">
        <label :for="uid" class="mb-1.5 block text-[.85rem] text-ink-2">ایمیل</label>
        <input
          :id="uid"
          v-model="email"
          type="email"
          dir="ltr"
          autocomplete="email"
          placeholder="you@example.com"
          :aria-invalid="fieldError ? 'true' : undefined"
          :aria-describedby="fieldError ? `${uid}-err` : `${uid}-note`"
          class="min-h-[48px] w-full rounded-xl border bg-surface px-4 py-3 text-ink outline-none transition
                 placeholder:text-ink-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
          :class="fieldError
            ? 'border-accent/60 focus-visible:outline-accent'
            : 'border-white/[.12] focus:border-accent/55 focus:shadow-[0_0_0_4px_rgba(255,46,77,.12)] focus-visible:outline-accent'"
          @blur="touched = true"
        >
        <p v-if="fieldError" :id="`${uid}-err`" role="alert" class="mt-1.5 flex items-start gap-1.5 text-[.8rem] text-accent">
          <span aria-hidden="true">⚠</span><span>{{ fieldError }}</span>
        </p>
      </div>

      <button
        type="submit"
        :disabled="!valid || state === 'sending'"
        class="min-h-[48px] shrink-0 rounded-xl bg-accent px-5 font-bold text-white transition
               enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50
               focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent
               sm:mt-[30px]"
      >{{ state === 'sending' ? 'در حال ارسال…' : 'عضویت' }}</button>
    </form>

    <!-- The live region is always in the DOM. Mounting it together with the
         message is the classic way to make an aria-live announcement that
         nobody hears: the region has to exist before its content changes. -->
    <div aria-live="polite" class="empty:hidden">
      <div v-if="state === 'sent'" class="mt-4 rounded-xl border border-good/30 bg-good/[.12] p-5">
        <!-- Double opt-in, so «عضو شدی» would be false: the address is not on
             the list until the link is clicked. -->
        <div class="text-center font-bold text-good">ایمیلت را چک کن</div>
        <p class="mt-2 text-center text-[.86rem] leading-8 text-ink-2">
          یک لینک تأیید فرستادیم. تا وقتی رویش کلیک نکنی در فهرست نیستی و هیچ ایمیلی برایت نمی‌آید.
        </p>
        <p class="mt-1 text-center font-mono text-[.8rem] text-ink-3" dir="ltr">{{ sentTo }}</p>
        <p class="mt-4 border-t border-good/20 pt-3 text-center text-[.8rem] leading-7 text-ink-2">
          نرسید؟ پوشه‌ی <span class="text-ink">اسپم</span> را نگاه کن — دامنه‌ی تازه معمولاً آنجا می‌افتد.
        </p>
        <div class="mt-3 text-center">
          <button
            type="button"
            class="min-h-[44px] rounded-full border border-white/[.12] px-4 text-[.82rem] text-ink-3 transition hover:text-ink"
            @click="startOver"
          >ایمیل را عوض کن</button>
        </div>
      </div>

      <p
        v-else-if="state === 'error'"
        class="mt-3 flex items-start gap-2 rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] leading-7 text-accent"
      >
        <span aria-hidden="true">⚠</span><span>ارسال انجام نشد. یک لحظه بعد دوباره امتحان کن.</span>
      </p>
    </div>

    <p v-if="state !== 'sent'" :id="`${uid}-note`" class="mt-3 text-[.76rem] leading-7 text-ink-3">
      اول یک ایمیل تأیید می‌آید؛ بدون کلیکِ تو چیزی نمی‌فرستیم.
      پای هر ایمیل یک لینک <span class="text-ink-2">لغو اشتراک</span> هست که با یک کلیک کار می‌کند.
      آدرست را به هیچ‌کس نمی‌دهیم.
    </p>
  </div>
</template>
