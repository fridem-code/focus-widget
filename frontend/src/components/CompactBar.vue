<script setup lang="ts">
import { computed } from 'vue'
import { Phase, Status, type State } from '../../bindings/focuswidget/internal/timer/models'
import { formatTime, mainAction, phaseLabel, progress } from '../lib/timerView'

const props = defineProps<{ state: State }>()
const emit = defineEmits<{
  start: []
  pause: []
  resume: []
  skip: []
  expand: []
}>()

const paused = computed(() => props.state.status === Status.Paused)
const action = computed(() => mainAction(props.state))
const labels = { start: 'Старт', pause: 'Пауза', resume: 'Продолжить' } as const
// Music always waits in compact mode (SPEC §3.4); say so while a focus runs.
const musicHeld = computed(
  () => props.state.phase === Phase.Focus && props.state.status === Status.Running,
)
const percent = computed(() => `${(progress(props.state) * 100).toFixed(2)}%`)

function onMain() {
  if (action.value === 'pause') emit('pause')
  else if (action.value === 'resume') emit('resume')
  else emit('start')
}
</script>

<template>
  <div class="bar" :class="{ paused }">
    <div class="info">
      <div class="label">
        <svg
          v-if="paused"
          aria-hidden="true"
          width="10"
          height="10"
          viewBox="0 0 24 24"
          fill="currentColor"
        >
          <rect x="5" y="4" width="5" height="16" rx="1" />
          <rect x="14" y="4" width="5" height="16" rx="1" />
        </svg>
        <span v-else class="dot" aria-hidden="true"></span>
        {{ phaseLabel(state, 'compact') }}
        <svg
          v-if="musicHeld"
          class="music-off"
          role="img"
          aria-label="Музыка на паузе"
          width="13"
          height="13"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M9 18V5l12-2v13" />
          <circle cx="6" cy="18" r="3" />
          <circle cx="18" cy="16" r="3" />
          <path d="M2 2l20 20" />
        </svg>
      </div>
      <div class="time" role="timer">{{ formatTime(state.remainingMs) }}</div>
    </div>

    <div class="buttons">
      <button class="main" type="button" :aria-label="labels[action]" @click="onMain">
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
        <svg
          v-else
          aria-hidden="true"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="currentColor"
        >
          <path d="M7 4.5l13 7.5-13 7.5z" />
        </svg>
      </button>
      <button class="icon-button" type="button" aria-label="Пропустить фазу" @click="emit('skip')">
        <svg
          aria-hidden="true"
          width="17"
          height="17"
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
      <button class="icon-button" type="button" aria-label="Развернуть" @click="emit('expand')">
        <svg
          aria-hidden="true"
          width="17"
          height="17"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7" />
        </svg>
      </button>
    </div>

    <div
      class="progress"
      role="progressbar"
      aria-label="Прогресс фазы"
      :aria-valuenow="Math.round(progress(state) * 100)"
      aria-valuemin="0"
      aria-valuemax="100"
      :style="{ width: percent }"
    ></div>
  </div>
</template>

<style scoped>
/* The whole bar drags the window; its buttons don't. */
.bar {
  --wails-draggable: drag;
  position: relative;
  height: 100%;
  padding: 12px 12px 12px 18px;
  display: flex;
  align-items: center;
  gap: 12px;
  overflow: hidden;
}

.info {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--accent-text);
  white-space: nowrap;
}

.paused .label {
  color: var(--pill-neutral-text);
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-text);
}

.music-off {
  margin-left: 4px;
  color: var(--text-muted);
}

.time {
  font-size: 40px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
  color: var(--text-strong);
}

.paused .time {
  opacity: 0.5;
}

.buttons {
  --wails-draggable: no-drag;
  display: flex;
  align-items: center;
  gap: 4px;
}

.buttons .icon-button {
  color: var(--icon-strong);
}

.main {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--accent);
  color: #fff;
}

.main:hover {
  filter: brightness(1.08);
}

.progress {
  position: absolute;
  left: 0;
  bottom: 0;
  height: 3px;
  background: var(--accent-text);
  transition: width 0.3s linear;
}

.paused .progress {
  opacity: 0.4;
}
</style>
