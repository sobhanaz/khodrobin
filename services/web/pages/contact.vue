<script setup lang="ts">
useSeoMeta({
  title: 'تماس با خودروبین',
  description: 'سؤال، ایراد یا پیشنهاد؟ بنویس.',
})

const form = reactive({ name: '', email: '', subject: '', body: '' })
const sending = ref(false)
const sent = ref(false)
const error = ref<string | null>(null)

const bodyLength = computed(() => form.body.trim().length)
const canSend = computed(() =>
  form.name.trim().length > 1 && /.+@.+\..+/.test(form.email) && bodyLength.value >= 10)

async function submit() {
  if (!canSend.value || sending.value) return
  sending.value = true; error.value = null
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
    error.value = e?.data?.error ?? 'ارسال نشد. دوباره تلاش کن.'
  } finally {
    sending.value = false
  }
}

const field = 'w-full rounded-xl border border-white/[.12] bg-surface px-4 py-3 text-ink ' +
  'outline-none transition placeholder:text-ink-3 focus:border-accent/55 focus:shadow-[0_0_0_4px_rgba(255,46,77,.12)]'
</script>

<template>
  <main class="mx-auto max-w-[640px] px-5 py-16">
    <h1 class="text-[clamp(1.7rem,4vw,2.4rem)] font-black tracking-tight">تماس با ما</h1>
    <p class="mt-3 text-ink-2">
      ایرادی دیدی، عددی اشتباه بود، یا منبعی هست که باید اضافه کنیم؟ بنویس.
    </p>

    <div v-if="sent" class="mt-8 rounded-2xl border border-good/30 bg-good/[.12] p-6 text-center">
      <div class="text-[1.05rem] font-bold text-good">پیامت رسید</div>
      <p class="mt-2 text-[.9rem] text-ink-2">ممنون. اگر جواب لازم داشته باشد، به همان ایمیل می‌نویسیم.</p>
      <NuxtLink to="/" class="mt-4 inline-block text-[.85rem] text-accent hover:underline">بازگشت به جست‌وجو</NuxtLink>
    </div>

    <form v-else class="mt-8 grid gap-4" @submit.prevent="submit">
      <div>
        <label for="c-name" class="mb-1.5 block text-[.85rem] text-ink-2">نام</label>
        <input id="c-name" v-model="form.name" :class="field" autocomplete="name" required>
      </div>
      <div>
        <label for="c-email" class="mb-1.5 block text-[.85rem] text-ink-2">ایمیل</label>
        <input id="c-email" v-model="form.email" type="email" dir="ltr" :class="field"
               autocomplete="email" placeholder="you@example.com" required>
      </div>
      <div>
        <label for="c-subject" class="mb-1.5 block text-[.85rem] text-ink-2">موضوع <span class="text-ink-3">(اختیاری)</span></label>
        <input id="c-subject" v-model="form.subject" :class="field">
      </div>
      <div>
        <label for="c-body" class="mb-1.5 block text-[.85rem] text-ink-2">پیام</label>
        <textarea id="c-body" v-model="form.body" rows="6" :class="field" required />
        <div class="mt-1 text-left font-mono text-[.7rem] text-ink-3" dir="ltr">
          {{ bodyLength }} / 10 min
        </div>
      </div>

      <p v-if="error" class="rounded-xl border border-accent/30 bg-accent/[.12] px-4 py-3 text-[.85rem] text-accent">
        {{ error }}
      </p>

      <button
        type="submit"
        :disabled="!canSend || sending"
        class="rounded-xl bg-accent px-5 py-3 font-bold text-white transition
               enabled:hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40"
      >{{ sending ? 'در حال ارسال…' : 'ارسال پیام' }}</button>

      <p class="text-[.78rem] leading-7 text-ink-3">
        پیام‌ها ذخیره می‌شوند و بعد ایمیل — تا اگر سرویس ایمیل قطع بود، پیامت گم نشود.
      </p>
    </form>
  </main>
</template>
