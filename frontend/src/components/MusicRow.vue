<script setup lang="ts">
import { computed, ref, useTemplateRef } from 'vue'
import { useDismiss } from '../composables/useDismiss'
import { genreLabel, position } from '../lib/music'
import type { MusicPlayer } from '../lib/musicPlayer'
import GenreMenu from './GenreMenu.vue'
import VolumePopover from './VolumePopover.vue'

const props = defineProps<{ music: MusicPlayer }>()
/** Open state of the genre list; App opens it from the "no videos" card too. */
const genreOpen = defineModel<boolean>('genreOpen', { default: false })
/** A manual play/pause, so the timer policy can respect it (SPEC §3.3). */
const emit = defineEmits<{ user: [action: 'play' | 'pause'] }>()
const volumeOpen = ref(false)

const s = props.music.state
const genre = computed(() => props.music.currentGenre())
const genres = computed(() => props.music.genres())
const wanted = computed(() => s.wanted && !s.error)

const title = computed(() => {
  if (s.error === 'offline') return 'Ожидание сети'
  if (s.error) return 'Музыка недоступна'
  return s.title || s.genreName
})
const label = computed(() =>
  genre.value ? genreLabel(s.genreName, position(genre.value, s.sourceIndex, s.playlist)) : '',
)

const row = useTemplateRef<HTMLElement>('row')
const genreTrigger = useTemplateRef<HTMLButtonElement>('genreTrigger')
const volumeTrigger = useTemplateRef<HTMLButtonElement>('volumeTrigger')
useDismiss(genreOpen, [row], () => genreTrigger.value?.focus())
useDismiss(volumeOpen, [row], () => volumeTrigger.value?.focus())

function toggleGenres() {
  volumeOpen.value = false
  genreOpen.value = !genreOpen.value
}

function toggleVolume() {
  genreOpen.value = false
  volumeOpen.value = !volumeOpen.value
}

function selectGenre(id: string) {
  props.music.selectGenre(id)
  genreOpen.value = false
  genreTrigger.value?.focus()
}

function togglePlay() {
  const action = wanted.value ? 'pause' : 'play'
  if (action === 'pause') props.music.pause()
  else props.music.play()
  emit('user', action)
}
</script>

<template>
  <div ref="row" class="row">
    <svg
      class="note"
      :class="{ on: s.playing }"
      role="img"
      :aria-label="s.playing ? 'Музыка играет' : 'Музыка не играет'"
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M9 18V5l12-2v13" />
      <circle cx="6" cy="18" r="3" />
      <circle cx="18" cy="16" r="3" />
      <path v-if="!s.playing" d="M2 2l20 20" />
    </svg>

    <div class="text">
      <div class="title">{{ title }}</div>
      <button
        ref="genreTrigger"
        class="genre"
        :class="{ open: genreOpen }"
        type="button"
        aria-haspopup="menu"
        :aria-expanded="genreOpen"
        :aria-label="`Выбрать жанр: ${s.genreName}`"
        @click="toggleGenres"
      >
        {{ label }}
        <svg
          aria-hidden="true"
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path :d="genreOpen ? 'M6 15l6-6 6 6' : 'M6 9l6 6 6-6'" />
        </svg>
      </button>
    </div>

    <div class="buttons">
      <button
        class="icon-button"
        type="button"
        :aria-label="wanted ? 'Пауза музыки' : 'Включить музыку'"
        :disabled="s.error === 'no_videos'"
        @click="togglePlay"
      >
        <svg
          v-if="wanted"
          aria-hidden="true"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="currentColor"
        >
          <rect x="6" y="5" width="4" height="14" rx="1" />
          <rect x="14" y="5" width="4" height="14" rx="1" />
        </svg>
        <svg
          v-else
          aria-hidden="true"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="currentColor"
        >
          <path d="M7 4.5l13 7.5-13 7.5z" />
        </svg>
      </button>
      <button
        ref="volumeTrigger"
        class="icon-button"
        :class="{ active: volumeOpen }"
        type="button"
        aria-label="Громкость"
        :aria-expanded="volumeOpen"
        @click="toggleVolume"
      >
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
          <path d="M11 5L6 9H3v6h3l5 4z" />
          <template v-if="s.muted || s.volume === 0">
            <path d="M16 9l5 6M21 9l-5 6" />
          </template>
          <template v-else>
            <path d="M15.5 8.5a5 5 0 0 1 0 7" />
            <path d="M18.5 5.5a9 9 0 0 1 0 13" />
          </template>
        </svg>
      </button>
      <button
        class="icon-button"
        type="button"
        :aria-label="s.generated ? 'Следующий звук' : 'Следующее видео'"
        :disabled="!s.generated && (!s.ready || !!s.error)"
        @click="music.nextTrack()"
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
          stroke-linejoin="round"
        >
          <path d="M5 5l10 7-10 7z" fill="currentColor" />
          <path d="M19 5v14" />
        </svg>
      </button>
    </div>

    <GenreMenu v-if="genreOpen" :genres="genres" :current-id="s.genreId" @select="selectGenre" />
    <VolumePopover
      v-if="volumeOpen"
      :volume="s.volume"
      :muted="s.muted"
      @volume="music.setVolume($event)"
      @muted="music.setMuted($event)"
    />
  </div>
</template>

<style scoped>
/* Not positioned: the popovers place themselves relative to the widget. */
.row {
  margin-top: 2px;
  padding: 6px 6px 6px 12px;
  border-radius: var(--radius-l);
  background: var(--surface);
  display: flex;
  align-items: center;
  gap: 10px;
}

.note {
  flex-shrink: 0;
  color: var(--player-placeholder);
}

.note.on {
  color: var(--icon);
}

.text {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.title {
  font-weight: 500;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.genre {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--text-muted);
}

.genre:hover,
.genre.open {
  color: var(--text);
}

.buttons {
  flex-shrink: 0;
  display: flex;
  gap: 2px;
}

.buttons .icon-button {
  color: var(--icon-strong);
}

.buttons .icon-button:disabled {
  color: var(--disabled);
}

.icon-button.active {
  background: var(--surface-hover);
  color: var(--text);
}
</style>
