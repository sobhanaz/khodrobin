<script setup lang="ts">
const props = withDefaults(defineProps<{
  password: string
  /**
   * Whether failing a rule is an error or merely information.
   *
   * On register and reset the rules are a gate: the account cannot be created
   * until they pass, so an unmet rule is shown as something to fix.
   *
   * On login they are not a gate and must never become one. Accounts created
   * before this policy existed have passwords that satisfy only the length
   * rule — including the first admin account. A login form that refused those
   * would lock out precisely the people who have been here longest, and there
   * is no way for them to fix it without logging in first.
   */
  enforcing?: boolean
}>(), { enforcing: true })

const { rules, satisfied } = usePasswordRules(() => props.password)

// Nothing to say about an empty field. Rules that appear before you have typed
// anything read as a list of complaints.
const show = computed(() => props.password.length > 0)
</script>

<template>
  <div v-if="show" class="mt-2">
    <ul class="grid gap-1" :aria-label="enforcing ? 'شرط‌های رمز عبور' : 'وضعیت رمز عبور'">
      <li
        v-for="r in rules"
        :key="r.key"
        class="flex items-center gap-2 text-[.76rem] leading-6 transition-colors"
        :class="r.ok ? 'text-good' : enforcing ? 'text-ink-3' : 'text-ink-3/70'"
      >
        <!-- Shape as well as colour. A green tick and a grey dot that differ
             only in hue say nothing to a colour-blind reader. -->
        <span
          class="grid size-[15px] shrink-0 place-items-center rounded-full border text-[9px]"
          :class="r.ok ? 'border-good/50 bg-good/15' : 'border-white/15'"
          aria-hidden="true"
        >{{ r.ok ? '✓' : '' }}</span>
        <span>{{ r.label }}</span>
        <span class="sr-only">{{ r.ok ? '— انجام شد' : '— هنوز نه' }}</span>
      </li>
    </ul>

    <!-- The whole reason this component renders on the login page. A password
         that predates the policy still works, and saying so prevents the reading
         someone would otherwise reach for: that their correct password is being
         rejected. It also turns an idle checklist into the one moment we know
         the person is holding their password and could improve it. -->
    <p
      v-if="!enforcing && !satisfied"
      class="mt-2 rounded-lg border border-white/[.08] bg-surface-2/60 px-3 py-2 text-[.74rem] leading-6 text-ink-3"
    >
      رمزهای قدیمی‌تر همچنان کار می‌کنند — این فقط اطلاع‌رسانی است.
      <NuxtLink to="/forgot" class="text-ink-2 underline underline-offset-2 hover:text-ink">
        می‌خواهی رمز قوی‌تری بگذاری؟
      </NuxtLink>
    </p>
  </div>
</template>
