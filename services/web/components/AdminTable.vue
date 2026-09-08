<script setup lang="ts" generic="T">
defineProps<{
  cols: string[]
  rows: T[]
  total: number
  offset: number
  limit: number
  loading: boolean
  failed: boolean
}>()

const emit = defineEmits<{ page: [delta: number] }>()
const f = useFormat()

// Focus is the lavender token, never the accent.
const ring = 'outline-none focus-visible:outline focus-visible:outline-2 ' +
  'focus-visible:outline-offset-2 focus-visible:outline-focus'
</script>

<template>
  <div>
    <!-- The scrollbar belongs to the table, not the document. Seven columns do
         not fit 375px, and a page that scrolls sideways as a whole puts the
         header and the nav off-screen too — the operator loses their place. -->
    <div class="overflow-x-auto rounded-2xl border border-white/[.07] bg-surface">
      <table class="w-full min-w-[680px] text-[.84rem]">
        <thead>
          <tr class="border-b border-white/[.07] text-right text-[.74rem] text-ink-2">
            <th v-for="c in cols" :key="c" class="whitespace-nowrap px-4 py-3 font-normal">{{ c }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td :colspan="cols.length" class="px-4 py-8 text-center text-ink-2">در حال بارگذاری…</td>
          </tr>
          <tr v-else-if="failed">
            <td :colspan="cols.length" class="px-4 py-8 text-center text-accent">بارگذاری نشد.</td>
          </tr>
          <tr v-else-if="!rows.length">
            <td :colspan="cols.length" class="px-4 py-8 text-center text-ink-2">چیزی برای نمایش نیست.</td>
          </tr>
          <template v-for="(row, i) in rows" v-else :key="i">
            <slot name="row" :row="row" />
          </template>
        </tbody>
      </table>
    </div>

    <div class="mt-3 flex items-center justify-between gap-3 text-[.78rem] text-ink-2">
      <span dir="rtl">
        <template v-if="total">
          {{ f.fa(offset + 1) }} تا {{ f.fa(Math.min(offset + limit, total)) }} از {{ f.fa(total) }}
        </template>
        <template v-else>-</template>
      </span>
      <div class="flex gap-2">
        <button
          type="button" :disabled="offset === 0"
          :class="`inline-flex min-h-11 items-center rounded-full border border-white/[.12] px-4
                   transition hover:text-ink disabled:opacity-35 disabled:hover:text-ink-2 ${ring}`"
          @click="emit('page', -1)"
        >قبلی</button>
        <button
          type="button" :disabled="offset + limit >= total"
          :class="`inline-flex min-h-11 items-center rounded-full border border-white/[.12] px-4
                   transition hover:text-ink disabled:opacity-35 disabled:hover:text-ink-2 ${ring}`"
          @click="emit('page', 1)"
        >بعدی</button>
      </div>
    </div>
  </div>
</template>
