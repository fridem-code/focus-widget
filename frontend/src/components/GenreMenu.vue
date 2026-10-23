<script setup lang="ts">
import { nextTick, onMounted, useTemplateRef } from 'vue'
import type { Genre } from '../../bindings/focuswidget/internal/catalog/models'

const props = defineProps<{ genres: Genre[]; currentId: string }>()
const emit = defineEmits<{ select: [id: string] }>()

const list = useTemplateRef<HTMLElement>('list')
const items = () =>
  Array.from(list.value?.querySelectorAll<HTMLElement>('[role=menuitemradio]') ?? [])

// Focus the current genre when the menu opens, so arrows and Enter work at once.
onMounted(async () => {
  await nextTick()
  const all = items()
  const i = props.genres.findIndex((g) => g.id === props.currentId)
  all[Math.max(0, i)]?.focus()
})

function onKey(e: KeyboardEvent) {
  const all = items()
  const i = all.indexOf(document.activeElement as HTMLElement)
  const move = (to: number) => {
    e.preventDefault()
    all[(to + all.length) % all.length]?.focus()
  }
  if (e.key === 'ArrowDown') move(i + 1)
  else if (e.key === 'ArrowUp') move(i - 1)
  else if (e.key === 'Home') move(0)
  else if (e.key === 'End') move(all.length - 1)
}
</script>

<template>
  <div ref="list" class="menu" role="menu" aria-label="Жанры" @keydown="onKey">
    <button
      v-for="g in genres"
      :key="g.id"
      class="item"
      type="button"
      role="menuitemradio"
      :aria-checked="g.id === currentId"
      @click="emit('select', g.id)"
    >
      <svg
        class="check"
        aria-hidden="true"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.25"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path v-if="g.id === currentId" d="M5 12.5l4.5 4.5L19 7.5" />
      </svg>
      <span>{{ g.name }}</span>
    </button>
  </div>
</template>

<style scoped>
/* Opens above the music row but below the player: it must never cover the
   iframe (SPEC §2). 266px = player bottom + 4px; 142px fits four genres and
   ends just above the music row, longer catalogs scroll. */
.menu {
  position: absolute;
  left: 28px;
  top: 266px;
  width: 210px;
  max-height: 142px;
  overflow-y: auto;
  padding: 4px;
  border-radius: var(--radius-l);
  background: var(--popover);
  border: 1px solid var(--glass-border);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  gap: 1px;
  z-index: 10;
}

.item {
  height: 32px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border-radius: var(--radius-s);
  color: var(--text-title);
  text-align: left;
}

.item[aria-checked='true'] {
  background: var(--surface-hover);
  color: var(--text);
  font-weight: 500;
}

.item:hover,
.item:focus-visible {
  background: var(--surface-hover);
  outline: none;
}

.check {
  flex-shrink: 0;
  color: var(--accent-text);
}
</style>
