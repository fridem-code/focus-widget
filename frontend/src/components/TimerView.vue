<script setup lang="ts">
import { computed } from 'vue'
import { Status, type State } from '../../bindings/focuswidget/internal/timer/models'
import { formatTime, phaseLabel, pillTone, progress } from '../lib/timerView'

const props = defineProps<{ state: State }>()

const paused = computed(() => props.state.status === Status.Paused)
const percent = computed(() => `${(progress(props.state) * 100).toFixed(2)}%`)
</script>

<template>
  <section class="timer" :class="{ paused }" aria-live="off">
    <div class="pill" :class="pillTone(state)">
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
      {{ phaseLabel(state) }}
    </div>
    <div class="time" role="timer">{{ formatTime(state.remainingMs) }}</div>
    <div
      class="track"
      role="progressbar"
      aria-label="Прогресс фазы"
      :aria-valuenow="Math.round(progress(state) * 100)"
      aria-valuemin="0"
      aria-valuemax="100"
    >
      <div class="fill" :style="{ width: percent }"></div>
    </div>
  </section>
</template>

<style scoped>
.timer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding-top: 4px;
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}

.pill.accent {
  background: var(--accent-soft);
  color: var(--accent-text);
}

.pill.neutral {
  background: var(--pill-neutral);
  color: var(--pill-neutral-text);
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--accent-text);
}

.time {
  font-size: 76px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
  color: var(--text-strong);
}

.track {
  width: 260px;
  height: 4px;
  border-radius: 2px;
  background: var(--track);
  overflow: hidden;
}

.fill {
  height: 100%;
  border-radius: 2px;
  background: var(--accent-text);
  transition: width 0.3s linear;
}

.paused .time,
.paused .fill {
  opacity: 0.5;
}
</style>
