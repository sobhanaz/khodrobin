<script setup lang="ts">
withDefaults(defineProps<{
  /** Rendered size in pixels. The art is drawn on a 64 grid and scales cleanly. */
  size?: number
  /**
   * Whether this instance carries the accessible name.
   *
   * The header pairs the mark with the word «خودروبین», and announcing both
   * would read the brand twice to a screen reader. There the mark is decorative
   * and the text is the name; standing alone it has to name itself.
   */
  labelled?: boolean
}>(), { size: 28, labelled: false })
</script>

<template>
  <svg
    :width="size" :height="size" viewBox="0 0 64 64"
    :role="labelled ? 'img' : undefined"
    :aria-label="labelled ? 'خودروبین' : undefined"
    :aria-hidden="labelled ? undefined : 'true'"
    fill="none" xmlns="http://www.w3.org/2000/svg"
  >
    <!-- The ring is a gauge, not decoration: one arc of a full circle, which is
         exactly what this product computes for every car — where one price sits
         in the distribution of all the prices for that car. -->
    <circle cx="32" cy="32" r="25" stroke="currentColor" stroke-width="4.5" opacity=".96" />
    <circle
      cx="32" cy="32" r="25"
      stroke="var(--color-accent, #ff2e4d)" stroke-width="4.5" stroke-linecap="butt"
      stroke-dasharray="45.8 157.1" transform="rotate(-90 32 32)"
    />

    <!-- Side profile, drawn heavy on purpose. At 16px in a browser tab a thin
         car turns to grey mush, and the tab is the smallest size this has to
         survive. -->
    <path
      d="M19.4 38.6v-4.2c0-1 .6-1.9 1.5-2.3l3.2-1.4 3.2-4.1c.7-.9 1.7-1.4 2.8-1.4h6.1c1.1 0 2.2.5 2.9 1.4l3.2 4.1 3.2 1.4c.9.4 1.5 1.3 1.5 2.3v4.2c0 .8-.7 1.5-1.5 1.5H20.9c-.8 0-1.5-.7-1.5-1.5Z"
      fill="currentColor"
    />
    <circle cx="25.2" cy="40.4" r="2.9" fill="currentColor" />
    <circle cx="38.8" cy="40.4" r="2.9" fill="currentColor" />
  </svg>
</template>
