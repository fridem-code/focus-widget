<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Phase } from '../bindings/focuswidget/internal/timer/models'
import { MusicService, SettingsService } from '../bindings/focuswidget/services'
import MusicRow from './components/MusicRow.vue'
import PlayerPanel from './components/PlayerPanel.vue'
import TimerControls from './components/TimerControls.vue'
import TimerView from './components/TimerView.vue'
import TitleBar from './components/TitleBar.vue'
import { useMusic } from './composables/useMusic'
import { connectMusicPolicy } from './composables/useMusicPolicy'
import { useTimer } from './composables/useTimer'
import { persistMusic } from './lib/persistMusic'
import type { MusicPolicyRunner } from './lib/policyRunner'

const { state, start, pause, resume, skip, resetPhase } = useTimer()
const music = useMusic()
const genreMenuOpen = ref(false)
let policy: MusicPolicyRunner | undefined
let stopPersist: (() => void) | undefined

onMounted(async () => {
  const [catalog, settings] = await Promise.all([MusicService.Genres(), SettingsService.Get()])
  music.configure({
    catalog,
    genreId: settings.music.genreId,
    sourceIndex: settings.music.sourceIndex,
    volume: settings.music.volume,
    muted: settings.music.muted,
    shuffle: settings.music.shuffle,
  })
  stopPersist = persistMusic(music, MusicService)
  policy = connectMusicPolicy(music, settings)
})
onUnmounted(() => {
  policy?.stop()
  stopPersist?.()
  music.dispose()
})

// Starting a focus phase plays synchronously inside the click handler, before
// the round trip to Go (SPEC §5); the policy then confirms it on timer:transition.
// A break may want silence, so it is left to the policy.
function onStart() {
  if (state.value?.phase === Phase.Focus) music.play()
  void start()
}
</script>

<template>
  <div class="widget" :data-phase="state?.phase ?? 'focus'">
    <TitleBar />
    <PlayerPanel :music="music" @choose-genre="genreMenuOpen = true" />
    <template v-if="state">
      <TimerView :state="state" />
      <MusicRow
        v-if="music.state.configured"
        v-model:genre-open="genreMenuOpen"
        :music="music"
        @user="policy?.user($event)"
      />
      <TimerControls
        :state="state"
        @start="onStart"
        @pause="pause"
        @resume="resume"
        @skip="skip"
        @reset-phase="resetPhase"
      />
    </template>
  </div>
</template>

<style scoped>
/* The glass fills the whole window: Windows draws the corners and shadow.
   Positioned so the music popovers can place themselves against it. */
.widget {
  position: relative;
  height: 100%;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--glass);
  border: 1px solid var(--glass-border);
}
</style>
