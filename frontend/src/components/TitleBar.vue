<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { WindowService } from '../../bindings/focuswidget/services'
import { cleanLabel, pomodoros, progressText } from '../lib/stats'

const props = defineProps<{
  /** What the user is working on (T24); "" shows the prompt. */
  label: string
  /** Today's finished focus sessions and the daily goal, 0 for none (T25). */
  done: number
  goal: number
}>()
const emit = defineEmits<{ compact: []; settings: []; stats: []; label: [text: string] }>()

// The cross hides to the tray; only the tray's "Выход" quits (SPEC §3.4).
function hide() {
  void WindowService.Hide()
}

const editing = ref(false)
const draft = ref('')
const input = ref<HTMLInputElement>()

async function edit() {
  draft.value = props.label
  editing.value = true
  await nextTick()
  input.value?.focus()
  input.value?.select()
}

// Enter and leaving the field save; Esc keeps the label as it was.
function commit() {
  if (!editing.value) return
  editing.value = false
  const text = cleanLabel(draft.value)
  if (text !== props.label) emit('label', text)
}

function cancel() {
  editing.value = false
}

const reached = computed(() => props.goal > 0 && props.done >= props.goal)
const statsLabel = computed(() =>
  props.goal > 0
    ? `Статистика · сегодня ${props.done} из ${props.goal}`
    : `Статистика · сегодня ${pomodoros(props.done)}`,
)
</script>

<template>
  <header class="titlebar">
    <div class="name">
      <span class="dot" aria-hidden="true"></span>
      <input
        v-if="editing"
        ref="input"
        v-model="draft"
        class="label-input"
        type="text"
        maxlength="60"
        aria-label="Над чем работаю"
        placeholder="Над чем работаю?"
        @keydown.enter.prevent="commit"
        @keydown.esc.prevent="cancel"
        @blur="commit"
      />
      <button
        v-else
        class="label"
        :class="{ empty: !label }"
        type="button"
        :title="label || undefined"
        :aria-label="
          label ? `Над чем работаю: ${label}. Изменить` : 'Над чем работаю? Добавить подпись'
        "
        @click="edit"
      >
        {{ label || 'Над чем работаю?' }}
      </button>
    </div>
    <div class="actions">
      <button
        class="progress"
        :class="{ reached }"
        type="button"
        :aria-label="statsLabel"
        @click="emit('stats')"
      >
        <svg
          aria-hidden="true"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M5 20v-6M12 20V8M19 20V4" />
        </svg>
        <span aria-hidden="true">{{ progressText(done, goal) }}</span>
      </button>
      <button class="icon-button" type="button" aria-label="Настройки" @click="emit('settings')">
        <svg
          aria-hidden="true"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12" />
          <circle cx="16" cy="6" r="2" />
          <circle cx="10" cy="12" r="2" />
          <circle cx="18" cy="18" r="2" />
        </svg>
      </button>
      <button
        class="icon-button"
        type="button"
        aria-label="Свернуть в компактный режим"
        @click="emit('compact')"
      >
        <svg
          aria-hidden="true"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M4 14h6v6M20 10h-6V4M14 10l7-7M3 21l7-7" />
        </svg>
      </button>
      <button class="icon-button" type="button" aria-label="Скрыть в трей" @click="hide">
        <svg
          aria-hidden="true"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M6 6l12 12M18 6L6 18" />
        </svg>
      </button>
    </div>
  </header>
</template>

<style scoped>
.titlebar {
  --wails-draggable: drag;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 2px 0 6px;
}

.name {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.dot {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
}

/* The label keeps the drag area around it: only the text itself is a button. */
.label {
  --wails-draggable: no-drag;
  min-width: 0;
  max-width: 100%;
  padding: 2px 6px;
  margin-left: -6px;
  border-radius: var(--radius-s);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  color: var(--text-title);
  cursor: text;
}

.label.empty {
  font-weight: 500;
  color: var(--text-muted);
}

.label:hover {
  background: var(--surface-hover);
}

.label-input {
  --wails-draggable: no-drag;
  flex: 1 1 auto;
  min-width: 0;
  height: 26px;
  padding: 0 6px;
  margin-left: -6px;
  border: 1px solid var(--field-border);
  border-radius: var(--radius-s);
  background: var(--field-bg);
  color: var(--text);
  font: inherit;
  font-weight: 600;
  user-select: text;
}

.label-input:focus {
  outline: 2px solid var(--accent-text);
  outline-offset: -1px;
}

.actions {
  --wails-draggable: no-drag;
  flex-shrink: 0;
  display: flex;
  gap: 2px;
}

.progress {
  height: 36px;
  padding: 0 8px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border-radius: var(--radius-m);
  color: var(--icon);
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.progress:hover {
  background: var(--surface-hover);
}

.progress.reached {
  color: var(--accent-text);
}
</style>
