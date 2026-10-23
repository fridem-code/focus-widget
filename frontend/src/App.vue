<script setup lang="ts">
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { WindowMode as WindowModeEvent } from '../bindings/events'
import { WindowMode } from '../bindings/focuswidget/internal/config/models'
import { Phase } from '../bindings/focuswidget/internal/timer/models'
import { MusicService, SettingsService, WindowService } from '../bindings/focuswidget/services'
import CompactBar from './components/CompactBar.vue'
import MusicRow from './components/MusicRow.vue'
import PlayerPanel from './components/PlayerPanel.vue'
import TimerControls from './components/TimerControls.vue'
import TimerView from './components/TimerView.vue'
import TitleBar from './components/TitleBar.vue'
import { useMusic } from './composables/useMusic'
import { connectMusicPolicy } from './composables/useMusicPolicy'
import { useTimer } from './composables/useTimer'
import { useWindowActive } from './composables/useWindowActive'
import { persistMusic } from './lib/persistMusic'
import type { MusicPolicyRunner } from './lib/policyRunner'
import { createWindowMode, type WindowModeController } from './lib/windowMode'

const { state, start, pause, resume, skip, resetPhase } = useTimer()
const music = useMusic()
const active = useWindowActive()
const genreMenuOpen = ref(false)
const ready = ref(false)
const windowMode = shallowRef<WindowModeController>()
const compact = computed(() => windowMode.value?.mode.value === 'compact')
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
  const runner = connectMusicPolicy(music, settings)
  policy = runner
  windowMode.value = createWindowMode({
    initial: settings.window.mode === WindowMode.WindowCompact ? 'compact' : 'expanded',
    setMode: (m) =>
      WindowService.SetMode(m === 'compact' ? WindowMode.WindowCompact : WindowMode.WindowExpanded),
    onModeEvent: (cb) =>
      Events.On(WindowModeEvent, (ev) =>
        cb(ev.data === WindowMode.WindowCompact ? 'compact' : 'expanded'),
      ),
    beforeCompact: () => state.value && runner.mode('compact', state.value),
    afterExpand: () => state.value && runner.mode('expanded', state.value),
  })
  // Rendered only now, so a window saved in compact mode never shows the
  // expanded layout first.
  ready.value = true
})
onUnmounted(() => {
  windowMode.value?.stop()
  policy?.stop()
  stopPersist?.()
  music.dispose()
})

// Starting a focus phase plays synchronously inside the click handler, before
// the round trip to Go (SPEC §5); the policy then confirms it on timer:transition.
// A break may want silence and compact mode always does, so those are left to
// the policy.
function onStart() {
  if (!compact.value && state.value?.phase === Phase.Focus) music.play()
  void start()
}
</script>

<template>
  <div
    v-if="ready"
    class="widget"
    :class="{ compact }"
    :data-phase="state?.phase ?? 'focus'"
    :data-inactive="active ? undefined : ''"
  >
    <TitleBar v-if="!compact" @compact="windowMode?.compact()" />
    <!-- Hidden, not unmounted, in compact mode: the stream must not reload (SPEC §3.4). -->
    <PlayerPanel v-show="!compact" :music="music" @choose-genre="genreMenuOpen = true" />
    <template v-if="state && !compact">
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
    <CompactBar
      v-if="state && compact"
      :state="state"
      @start="onStart"
      @pause="pause"
      @resume="resume"
      @skip="skip"
      @expand="windowMode?.expand()"
    />
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

.widget.compact {
  padding: 0;
}
</style>
