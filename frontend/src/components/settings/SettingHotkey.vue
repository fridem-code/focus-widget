<script setup lang="ts">
import { ref, useId } from 'vue'
import { recordKey, showCombo } from '../../lib/hotkeys'

const props = defineProps<{
  label: string
  /** The shortcut, "" when off. */
  value: string
  /** Why the shortcut does not work (another app has it), if it doesn't. */
  problem?: string
  /** The action that already has a shortcut, null if none does. */
  takenBy: (combo: string) => string | null
}>()
const emit = defineEmits<{ commit: [combo: string]; recording: [on: boolean] }>()

const id = useId()
const recording = ref(false)
const hint = ref('')

function start() {
  if (recording.value) return
  recording.value = true
  hint.value = ''
  emit('recording', true)
}

function stop() {
  if (!recording.value) return
  recording.value = false
  hint.value = ''
  emit('recording', false)
}

function commit(combo: string) {
  stop()
  if (combo !== props.value) emit('commit', combo)
}

function onKey(e: KeyboardEvent) {
  if (!recording.value) return
  // Tab still moves on (and leaving the field cancels), so the keyboard
  // never gets stuck here.
  if (e.code === 'Tab' && !e.ctrlKey && !e.altKey && !e.metaKey) return
  e.preventDefault()
  const r = recordKey(e)
  switch (r.kind) {
    case 'cancel':
      stop()
      break
    case 'clear':
      commit('')
      break
    case 'invalid':
      hint.value = r.reason
      break
    case 'combo': {
      const other = r.combo === props.value ? null : props.takenBy(r.combo)
      if (other) hint.value = `Уже у «${other}»`
      else commit(r.combo)
      break
    }
  }
}
</script>

<template>
  <div class="hotkey">
    <div class="row">
      <span :id="id">{{ label }}</span>
      <div class="controls">
        <button
          class="combo"
          :class="{ recording, off: !value && !recording, bad: !!problem && !recording }"
          type="button"
          :aria-labelledby="id"
          :aria-describedby="problem || hint ? `${id}-note` : undefined"
          :aria-pressed="recording"
          @click="start"
          @keydown="onKey"
          @blur="stop"
        >
          {{ recording ? 'Нажмите сочетание…' : value ? showCombo(value) : 'Выключено' }}
        </button>
        <button
          v-if="value && !recording"
          class="icon-button small"
          type="button"
          :aria-label="`Выключить «${label}»`"
          @click="commit('')"
        >
          <svg
            aria-hidden="true"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          >
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
      </div>
    </div>
    <p v-if="recording" :id="`${id}-note`" class="note" role="status">
      {{ hint || 'Esc — отмена, Backspace — выключить' }}
    </p>
    <p v-else-if="problem" :id="`${id}-note`" class="note problem">{{ problem }}</p>
  </div>
</template>

<style scoped>
.row {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--text-title);
}

.controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.combo {
  min-width: 150px;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--field-border);
  border-radius: var(--radius-s);
  background: var(--field-bg);
  color: var(--text);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.combo:hover {
  background: var(--surface-hover);
}

.combo.off {
  color: var(--text-muted);
}

.combo.recording {
  border-color: var(--accent-text);
  color: var(--accent-text);
}

.combo.bad {
  border-color: var(--danger);
}

.icon-button.small {
  width: 26px;
  height: 26px;
}

.note {
  margin: -4px 0 4px;
  font-size: 12px;
  color: var(--text-muted);
  text-align: right;
}

.note.problem {
  color: var(--danger);
}
</style>
