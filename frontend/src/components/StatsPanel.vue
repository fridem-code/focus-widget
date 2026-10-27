<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Summary } from '../../bindings/focuswidget/internal/stats/models'
import {
  barShare,
  clockTime,
  dayName,
  formatMinutes,
  pomodoroWord,
  pomodoros,
  streakText,
  weekday,
} from '../lib/stats'

const props = defineProps<{
  /** Null until the first summary arrives. */
  summary: Summary | null
  /** Sessions a day to aim for; 0 is no goal (T25). */
  goal: number
}>()
const emit = defineEmits<{ back: []; hide: [] }>()

const today = computed(() => props.summary?.today ?? { date: '', focus: 0, minutes: 0 })
const week = computed(() => props.summary?.week ?? [])
const goalShare = computed(() => (props.goal > 0 ? Math.min(1, today.value.focus / props.goal) : 0))
// Newest first: the one just finished is on top.
const sessions = computed(() => [...(props.summary?.sessions ?? [])].reverse())

// Bars are at most this tall; a day with any focus gets at least a sliver.
const CHART_PX = 96
const barHeight = (minutes: number, share: number) =>
  minutes > 0 ? Math.max(4, Math.round(share * CHART_PX)) : 0
// The day under the mouse, whose numbers replace the week's total.
const hovered = ref<number | null>(null)
const barTitle = (d: { date: string; focus: number; minutes: number }) =>
  `${dayName(d.date)} · ${pomodoros(d.focus)} · ${formatMinutes(d.minutes)}`
</script>

<template>
  <!-- Focus colours whatever the phase: the screen is about focus time. -->
  <div class="stats" data-phase="focus">
    <header class="header">
      <button class="icon-button" type="button" aria-label="Назад к таймеру" @click="emit('back')">
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
          <path d="M15 5l-7 7 7 7" />
        </svg>
      </button>
      <h1>Статистика</h1>
      <button class="icon-button" type="button" aria-label="Скрыть в трей" @click="emit('hide')">
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
    </header>

    <div class="scroll">
      <section aria-labelledby="st-today" class="today">
        <h2 id="st-today">Сегодня</h2>
        <p class="hero">
          <span class="count">{{ today.focus }}</span>
          <span class="hero-unit">
            {{ pomodoroWord(today.focus) }} · {{ formatMinutes(today.minutes) }}
          </span>
        </p>
        <template v-if="goal > 0">
          <div
            class="goal-track"
            role="progressbar"
            aria-label="Цель на день"
            :aria-valuenow="today.focus"
            aria-valuemin="0"
            :aria-valuemax="goal"
          >
            <div class="goal-fill" :style="{ width: `${(goalShare * 100).toFixed(1)}%` }"></div>
          </div>
          <p class="note">
            {{
              today.focus >= goal
                ? `Цель на день выполнена: ${today.focus} из ${goal}`
                : `Цель на день: ${today.focus} из ${goal}`
            }}
          </p>
        </template>
        <p v-else class="note">Цель на день не задана — её можно поставить в настройках</p>
      </section>

      <section aria-labelledby="st-week">
        <div class="section-head">
          <h2 id="st-week">7 дней</h2>
          <span class="total">{{
            hovered !== null && week[hovered]
              ? barTitle(week[hovered])
              : `${formatMinutes(summary?.weekMinutes ?? 0)} фокуса`
          }}</span>
        </div>
        <div class="chart" aria-hidden="true" @mouseleave="hovered = null">
          <div
            v-for="(d, i) in week"
            :key="d.date"
            class="col"
            :class="{ today: i === week.length - 1, hovered: hovered === i }"
            @mouseenter="hovered = i"
          >
            <div class="bar-area">
              <div
                class="bar"
                :style="{ height: `${barHeight(d.minutes, barShare(d, week))}px` }"
              ></div>
            </div>
            <span class="day">{{ weekday(d.date) }}</span>
          </div>
        </div>
        <!-- The chart's numbers for screen readers. -->
        <table class="sr-only">
          <caption>
            Фокус по дням
          </caption>
          <tr>
            <th scope="col">День</th>
            <th scope="col">Помидоры</th>
            <th scope="col">Минуты</th>
          </tr>
          <tr v-for="d in week" :key="d.date">
            <th scope="row">{{ dayName(d.date) }}</th>
            <td>{{ d.focus }}</td>
            <td>{{ d.minutes }}</td>
          </tr>
        </table>
      </section>

      <section aria-labelledby="st-streak">
        <h2 id="st-streak">Серия</h2>
        <p class="streak">{{ streakText(summary?.streak ?? 0) }}</p>
      </section>

      <section aria-labelledby="st-sessions">
        <h2 id="st-sessions">Сессии сегодня</h2>
        <p v-if="!sessions.length" class="note">
          Пока ни одной: засчитывается фокус, доведённый до конца
        </p>
        <ul v-else class="sessions">
          <li v-for="s in sessions" :key="s.end">
            <span class="time">{{ clockTime(s.end) }}</span>
            <span class="what" :class="{ none: !s.label }" :title="s.label || undefined">
              {{ s.label || 'без подписи' }}
            </span>
            <span class="min">{{ s.minutes }} мин</span>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<style scoped>
.stats {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.header {
  --wails-draggable: drag;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
}

.header .icon-button {
  --wails-draggable: no-drag;
}

h1 {
  flex: 1 1 auto;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--track) transparent;
  padding: 2px 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

h2 {
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-muted);
}

p {
  margin: 0;
}

.hero {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.count {
  font-size: 44px;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  color: var(--text-strong);
}

.hero-unit {
  color: var(--text-title);
}

.goal-track {
  height: 6px;
  border-radius: 3px;
  background: var(--track);
  overflow: hidden;
}

.goal-fill {
  height: 100%;
  border-radius: 3px;
  background: var(--accent-text);
}

.note {
  font-size: 12px;
  color: var(--text-muted);
}

.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.total {
  font-size: 12px;
  color: var(--text-title);
  font-variant-numeric: tabular-nums;
}

/* One series, so no legend: the heading names it. Bars grow from the
   baseline with rounded tops. Pointing at a day shows its numbers in place
   of the week's total, where nothing can clip them. */
.chart {
  position: relative;
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  padding-top: 4px;
}

.col {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.bar-area {
  width: 100%;
  height: 96px;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  border-bottom: 1px solid var(--track);
}

.bar {
  width: 60%;
  max-width: 28px;
  border-radius: 4px 4px 0 0;
  background: var(--accent-text);
  opacity: 0.55;
}

.today .bar {
  opacity: 1;
}

.day {
  font-size: 11px;
  color: var(--text-muted);
}

.today .day {
  font-weight: 600;
  color: var(--text-title);
}

.hovered .bar-area {
  background: var(--surface);
  border-radius: var(--radius-s) var(--radius-s) 0 0;
}

.hovered .bar {
  opacity: 1;
}

.streak {
  color: var(--text-title);
}

.sessions {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
}

.sessions li {
  height: 30px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid var(--track);
}

.sessions li:last-child {
  border-bottom: 0;
}

.time,
.min {
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
  color: var(--text-muted);
}

.what {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text);
}

.what.none {
  color: var(--text-muted);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}
</style>
