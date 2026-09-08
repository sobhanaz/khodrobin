<script setup lang="ts">
defineProps<{ title: string, subtitle?: string }>()
</script>

<template>
  <!--
    Two columns, because one was leaving a hole.

    The single centred column ended around 840px on a 1440px screen and the
    footer did not start until 1040px, on a page 2156px tall. The old
    min-h-[70vh] was creating most of that height and nothing was using it.

    The second column answers the question the form cannot: why an account
    exists here at all. Search is anonymous and always will be, so a login page
    that says nothing is asking for a password in exchange for nothing visible.

    Below lg it collapses to one column with the form first, which is also the
    DOM order, so no tab sequence or screen-reader order has to be repaired.
  -->
  <main
    class="mx-auto grid w-full max-w-[1080px] gap-12 px-5 py-[var(--space-section)]
           lg:grid-cols-[minmax(0,25rem)_minmax(0,1fr)] lg:gap-16"
  >
    <div class="min-w-0">
      <!-- No tracking-tight. Persian letters join, and negative tracking pulls
           the connecting strokes into each other. -->
      <h1 class="text-[1.6rem] font-black leading-[1.35]">{{ title }}</h1>
      <p v-if="subtitle" class="mt-2 text-[.9rem] leading-8 text-ink-2">{{ subtitle }}</p>
      <div class="mt-7"><slot /></div>
    </div>

    <aside class="min-w-0 lg:border-s lg:border-white/[.07] lg:ps-16">
      <slot name="aside">
        <h2 class="text-[.95rem] font-bold text-ink">حساب به چه دردی می‌خورد؟</h2>

        <dl class="mt-5 grid gap-5">
          <div>
            <dt class="text-[.86rem] font-bold text-ink">ذخیره‌ی جست‌وجو</dt>
            <dd class="mt-1 text-[.82rem] leading-7 text-ink-2">
              جست‌وجویی که ساخته‌ای را نگه می‌دارد تا دفعه‌ی بعد از اول ننویسی‌اش.
            </dd>
          </div>
          <div>
            <dt class="text-[.86rem] font-bold text-ink">هشدار قیمت</dt>
            <dd class="mt-1 text-[.82rem] leading-7 text-ink-2">
              وقتی میانه‌ی قیمت همان خودرو پایین آمد، خبرش با ایمیل می‌آید. لازم نیست هر روز سر بزنی.
            </dd>
          </div>
        </dl>

        <p class="mt-6 border-t border-white/[.07] pt-5 text-[.8rem] leading-7 text-ink-2">
          جست‌وجو هیچ‌وقت پشت ورود نیست و نخواهد رفت. بدون حساب هم کامل کار می‌کند؛ حساب فقط همین دو تا را
          اضافه می‌کند.
        </p>
      </slot>
    </aside>
  </main>
</template>
