<script setup lang="ts">
/**
 * Where you are in the signup, said in words first.
 *
 * Six fields on one screen reads as a form to escape from, so registration is
 * split in two. A split with no indicator is worse than no split at all: the
 * person has no idea whether pressing the button submits the form or opens
 * another page of it.
 */
const props = defineProps<{ current: number, steps: string[] }>()
const f = useFormat()

const label = computed(() =>
  `گام ${f.fa(props.current)} از ${f.fa(props.steps.length)}: ${props.steps[props.current - 1]}`)
</script>

<template>
  <div>
    <!-- The text carries the meaning and is announced on every change. The bars
         below are decoration and are hidden from assistive tech, because a row
         of unlabelled bars tells a screen reader nothing. -->
    <p class="text-[.82rem] font-bold text-ink-2" aria-live="polite">{{ label }}</p>

    <div class="mt-2 flex gap-1.5" aria-hidden="true">
      <span
        v-for="(s, i) in steps"
        :key="s"
        class="h-[3px] flex-1 rounded-full transition-colors duration-300"
        :class="i < current ? 'bg-accent' : 'bg-surface-2'"
      />
    </div>
  </div>
</template>
