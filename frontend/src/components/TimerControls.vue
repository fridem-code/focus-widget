<script setup lang="ts">
import { computed } from 'vue'
import type { State } from '../../bindings/focuswidget/internal/timer/models'
import { canResetPhase, mainAction } from '../lib/timerView'

const props = defineProps<{ state: State }>()
const emit = defineEmits<{ start: []; pause: []; resume: []; skip: []; resetPhase: [] }>()

const action = computed(() => mainAction(props.state))
const labels = { start: 'Старт', pause: 'Пауза', resume: 'Продолжить' } as const

function onMain() {
  if (action.value === 'pause') emit('pause')
  else if (action.value === 'resume') emit('resume')
  else emit('start')
}
</script>

<template>
  <div class="controls">
    <button
      class="round"
      type="button"
      aria-label="Сбросить фазу"
      :disabled="!canResetPhase(state)"
      @click="emit('resetPhase')"
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
        <path d="M3 12a9 9 0 1 0 3-6.7L3 8" />
        <path d="M3 3v5h5" />
      </svg>
    </button>
    <button class="main" type="button" @click="onMain">
      <svg
        v-if="action === 'pause'"
        aria-hidden="true"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="currentColor"
      >
        <rect x="6" y="5" width="4" height="14" rx="1" />
        <rect x="14" y="5" width="4" height="14" rx="1" />
      </svg>
      <svg v-else aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
        <path d="M7 4.5l13 7.5-13 7.5z" />
      </svg>
      {{ labels[action] }}
    </button>
    <button class="round" type="button" aria-label="Пропустить фазу" @click="emit('skip')">
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
        <path d="M5 5l10 7-10 7z" />
        <path d="M19 5v14" />
      </svg>
    </button>
  </div>
</template>

<style scoped>
.controls {
  margin-top: auto;
  padding-bottom: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.round {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--round);
  color: var(--text-title);
}

.round:not(:disabled):hover {
  background: var(--surface-hover);
}

.round:disabled {
  background: var(--round-disabled);
  color: var(--disabled);
}

/* Fixed width so the side buttons don't shift between Старт/Пауза/Продолжить. */
.main {
  height: 48px;
  width: 168px;
  border-radius: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--accent);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
}

.main:hover {
  filter: brightness(1.08);
}
</style>
