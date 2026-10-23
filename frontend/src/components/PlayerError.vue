<script setup lang="ts">
import { computed } from 'vue'
import type { PlayerErrorKind } from '../lib/music'

const props = defineProps<{ kind: PlayerErrorKind; code: number | null; genreName: string }>()
const emit = defineEmits<{ retry: []; chooseGenre: [] }>()

// Texts from the design's "Ошибки плеера" artboard (SPEC §3.7).
const content = computed(() => {
  switch (props.kind) {
    case 'no_videos':
      return {
        title: 'Нет доступных видео',
        text: `Все видео жанра «${props.genreName}» сейчас недоступны для встраивания. Таймер продолжает работать.`,
        action: 'Выбрать другой жанр',
      }
    case 'offline':
      return {
        title: 'Нет подключения к интернету',
        text: 'Таймер работает. Музыка включится сама, когда сеть вернётся.',
        action: 'Повторить',
      }
    default:
      return {
        title: 'Плеер не может загрузить видео',
        text: `Ошибка YouTube ${props.code ?? ''}. Таймер продолжает работать.`,
        action: 'Повторить',
      }
  }
})

function onAction() {
  if (props.kind === 'no_videos') emit('chooseGenre')
  else emit('retry')
}
</script>

<template>
  <div class="card" role="alert">
    <svg
      v-if="kind === 'offline'"
      aria-hidden="true"
      width="28"
      height="28"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M2 2l20 20" />
      <path d="M8.5 16.5a5 5 0 0 1 7 0" />
      <path d="M5 12.9a10 10 0 0 1 5.2-2.8" />
      <path d="M19 12.9a10 10 0 0 0-2.5-1.7" />
      <path d="M2 8.8a15 15 0 0 1 4.2-2.6" />
      <path d="M22 8.8a15 15 0 0 0-11.3-3.7" />
      <circle cx="12" cy="20" r="0.8" fill="currentColor" />
    </svg>
    <svg
      v-else
      aria-hidden="true"
      width="28"
      height="28"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <circle cx="12" cy="12" r="9.5" />
      <path d="M12 7.5v5.5" />
      <circle cx="12" cy="16.5" r="0.8" fill="currentColor" />
    </svg>
    <div class="title">{{ content.title }}</div>
    <div class="text">{{ content.text }}</div>
    <button class="action" type="button" @click="onAction">{{ content.action }}</button>
  </div>
</template>

<style scoped>
.card {
  width: 100%;
  height: 100%;
  padding: 20px 28px;
  border-radius: var(--radius-l);
  background: #12141b;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #e6e8ee;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  text-align: center;
}

.title {
  font-size: 14px;
  font-weight: 600;
  color: #f3f4f8;
}

.text {
  font-size: 12px;
  line-height: 1.45;
  color: #a7acbb;
}

.action {
  margin-top: 6px;
  height: 32px;
  padding: 0 14px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.action:hover {
  background: rgba(255, 255, 255, 0.18);
}
</style>
