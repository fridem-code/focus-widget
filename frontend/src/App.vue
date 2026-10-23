<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { MusicService, SettingsService } from '../bindings/focuswidget/services'
import PlayerPanel from './components/PlayerPanel.vue'
import TimerControls from './components/TimerControls.vue'
import TimerView from './components/TimerView.vue'
import TitleBar from './components/TitleBar.vue'
import { useMusic } from './composables/useMusic'
import { useTimer } from './composables/useTimer'

const { state, start, pause, resume, skip, resetPhase } = useTimer()
const music = useMusic()

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
})
onUnmounted(() => music.dispose())

// play() runs synchronously inside the click handler, before the round trip
// to Go (SPEC §5). The rest of the timer → music rules arrive with T8.
function onStart() {
  music.play()
  void start()
}
</script>

<template>
  <div class="widget" :data-phase="state?.phase ?? 'focus'">
    <TitleBar />
    <!-- T7 replaces nextGenre with opening the genre list -->
    <PlayerPanel :music="music" @choose-genre="music.nextGenre()" />
    <template v-if="state">
      <TimerView :state="state" />
      <!-- T7: music row goes here -->
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
/* The glass fills the whole window: Windows draws the corners and shadow. */
.widget {
  height: 100%;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--glass);
  border: 1px solid var(--glass-border);
}
</style>
