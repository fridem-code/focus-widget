<script setup lang="ts">
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import {
  HotkeysStatus,
  StatsChanged,
  UiCommand,
  WindowMode as WindowModeEvent,
  WindowVisible as WindowVisibleEvent,
} from '../bindings/events'
import { WindowMode } from '../bindings/focuswidget/internal/config/models'
import type { Summary } from '../bindings/focuswidget/internal/stats/models'
import { Phase } from '../bindings/focuswidget/internal/timer/models'
import {
  AutostartService,
  HotkeyService,
  MusicService,
  SettingsService,
  StatsService,
  TimerService,
  WindowService,
} from '../bindings/focuswidget/services'
import CompactBar from './components/CompactBar.vue'
import MusicRow from './components/MusicRow.vue'
import PlayerPanel from './components/PlayerPanel.vue'
import SettingsPanel from './components/SettingsPanel.vue'
import StatsPanel from './components/StatsPanel.vue'
import TimerControls from './components/TimerControls.vue'
import TimerView from './components/TimerView.vue'
import TitleBar from './components/TitleBar.vue'
import { useMusic } from './composables/useMusic'
import { connectMusicPolicy } from './composables/useMusicPolicy'
import { connectPhaseSound } from './composables/usePhaseSound'
import {
  applyTheme,
  createSettingsStore,
  syncGlass,
  themeIsDark,
  type SettingsStore,
} from './composables/useSettings'
import { useTimer } from './composables/useTimer'
import { useWindowActive } from './composables/useWindowActive'
import { LOCAL_GENRE_ID } from './lib/music'
import { persistMusic } from './lib/persistMusic'
import type { MusicPolicyRunner } from './lib/policyRunner'
import { createWindowMode, type WindowModeController } from './lib/windowMode'

const { state, start, pause, resume, skip, resetPhase } = useTimer()
const music = useMusic()
const active = useWindowActive()
const genreMenuOpen = ref(false)
const ready = ref(false)
const store = shallowRef<SettingsStore>()
const windowMode = shallowRef<WindowModeController>()
const compact = computed(() => windowMode.value?.mode.value === 'compact')
/** False while hidden in the tray. */
const visible = ref(true)
const screen = ref<'timer' | 'settings' | 'stats'>('timer')
/**
 * Whether the YouTube player can be seen. It can't in compact mode, in the
 * tray or behind the settings and statistics screens, and an unseen player must not play
 * (SPEC §2): the music player holds YouTube back while it is false. Generated
 * noise plays regardless.
 */
const playerShown = computed(() => visible.value && !compact.value && screen.value === 'timer')
let policy: MusicPolicyRunner | undefined
let stopPersist: (() => void) | undefined
const offEvents: (() => void)[] = []
// Starting with Windows, for the settings screen (T19).
const autostart = {
  enabled: () => AutostartService.Enabled(),
  set: (on: boolean) => AutostartService.Set(on),
}
// The music folder of "Мои файлы" (T22) and the folder the catalog was last
// built for, so the settings:changed after a choice does not rescan it again.
let catalogFolder = ''
async function reloadCatalog(folder: string) {
  catalogFolder = folder
  music.setCatalog(await MusicService.Genres())
}
// The statistics (T23): fetched at start, pushed by Go after each finished
// focus, and fetched again when the window comes back and at midnight, when
// "today" starts over.
const summary = ref<Summary | null>(null)
async function refreshStats() {
  summary.value = await StatsService.Summary()
}
let midnightTimer: ReturnType<typeof setTimeout> | undefined
function refreshAtMidnight() {
  const next = new Date()
  next.setHours(24, 0, 1, 0)
  midnightTimer = setTimeout(() => {
    void refreshStats()
    refreshAtMidnight()
  }, next.getTime() - Date.now())
}
// Global shortcuts that don't work (T26), and freeing them while the settings
// screen records a new one.
const hotkeyStatus = ref<Partial<Record<string, string>>>({})
const hotkeys = {
  get status() {
    return hotkeyStatus.value
  },
  suspend: (on: boolean) => void HotkeyService.Suspend(on),
}
function setLabel(label: string) {
  void store.value?.patch({ label }).then((err) => {
    if (err) console.warn('label not saved:', err)
  })
}
const musicFolder = {
  // Choosing a folder also switches to its music.
  async choose() {
    const folder = await MusicService.ChooseFolder()
    if (folder) {
      await reloadCatalog(folder)
      music.selectGenre(LOCAL_GENRE_ID)
    }
    return folder
  },
  clear: () => MusicService.ClearFolder(),
}

onMounted(async () => {
  const [catalog, settings, stats, status] = await Promise.all([
    MusicService.Genres(),
    SettingsService.Get(),
    StatsService.Summary(),
    HotkeyService.Status(),
  ])
  summary.value = stats
  hotkeyStatus.value = status ?? {}
  refreshAtMidnight()
  const settingsStore = createSettingsStore(settings)
  store.value = settingsStore
  offEvents.push(settingsStore.stop)
  // The theme and the window glass tinted for it follow the setting and, for
  // "system", Windows.
  const systemDark = matchMedia('(prefers-color-scheme: dark)')
  const syncTheme = () => {
    const theme = settingsStore.settings.value.window.theme
    applyTheme(theme)
    void syncGlass(themeIsDark(theme, systemDark.matches), WindowService.SetGlass)
  }
  syncTheme()
  watch(() => settingsStore.settings.value.window.theme, syncTheme)
  systemDark.addEventListener('change', syncTheme)
  offEvents.push(() => systemDark.removeEventListener('change', syncTheme))
  watch(
    () => settingsStore.settings.value.music.shuffle,
    (on) => music.setShuffle(on),
  )
  catalogFolder = settings.music.localFolder
  watch(
    () => settingsStore.settings.value.music.localFolder,
    (folder) => {
      if (folder !== catalogFolder) void reloadCatalog(folder)
    },
  )

  music.configure({
    catalog,
    genreId: settings.music.genreId,
    sourceIndex: settings.music.sourceIndex,
    volume: settings.music.volume,
    muted: settings.music.muted,
    shuffle: settings.music.shuffle,
  })
  stopPersist = persistMusic(music, MusicService)
  offEvents.push(connectPhaseSound(settings))
  policy = connectMusicPolicy(music, settings)
  // Immediate: a widget saved in compact mode starts with the player hidden.
  watch(playerShown, (shown) => music.setShown(shown), { immediate: true })

  const wm = createWindowMode({
    initial: settings.window.mode === WindowMode.WindowCompact ? 'compact' : 'expanded',
    setMode: (m) =>
      WindowService.SetMode(m === 'compact' ? WindowMode.WindowCompact : WindowMode.WindowExpanded),
    onModeEvent: (cb) =>
      Events.On(WindowModeEvent, (ev) =>
        cb(ev.data === WindowMode.WindowCompact ? 'compact' : 'expanded'),
      ),
    // Hold YouTube back before the window shrinks; resuming after it grows is
    // playerShown's job.
    beforeCompact: () => {
      screen.value = 'timer'
      music.setShown(false)
    },
    afterExpand: () => {},
  })
  windowMode.value = wm

  offEvents.push(
    Events.On(WindowVisibleEvent, (ev) => {
      visible.value = ev.data
      if (ev.data) void refreshStats()
    }),
    Events.On(StatsChanged, (ev) => {
      summary.value = ev.data
    }),
    Events.On(HotkeysStatus, (ev) => {
      hotkeyStatus.value = ev.data ?? {}
    }),
    // Tray commands the frontend carries out: it owns the player and the
    // order of pausing and resizing.
    Events.On(UiCommand, (ev) => {
      if (ev.data === 'compact') void wm.compact()
      else if (ev.data === 'expand') void wm.expand()
      else if (ev.data === 'next-genre') music.nextGenre()
      else if (ev.data === 'next-track') music.nextTrack()
    }),
  )
  // Rendered only now, so a window saved in compact mode never shows the
  // expanded layout first.
  ready.value = true
})
onUnmounted(() => {
  clearTimeout(midnightTimer)
  offEvents.forEach((off) => off())
  windowMode.value?.stop()
  policy?.stop()
  stopPersist?.()
  music.dispose()
})

// Starting a focus phase plays synchronously inside the click handler, before
// the round trip to Go (SPEC §5); the policy then confirms it on timer:transition.
// A break may want silence, so that is left to the policy; a hidden YouTube
// player waits in the music player itself.
function onStart() {
  if (state.value?.phase === Phase.Focus) music.play({ fade: true })
  void start()
}

function openStats() {
  screen.value = 'stats'
  void refreshStats()
}

function setAlwaysOnTop(on: boolean) {
  WindowService.SetAlwaysOnTop(on).catch((err: unknown) =>
    console.warn('always-on-top failed:', err),
  )
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
    <TitleBar
      v-if="!compact && screen === 'timer'"
      :label="store?.settings.value.timer.label ?? ''"
      :done="summary?.today.focus ?? 0"
      :goal="store?.settings.value.timer.dailyGoal ?? 0"
      @compact="windowMode?.compact()"
      @settings="screen = 'settings'"
      @stats="openStats"
      @label="setLabel"
    />
    <!-- Hidden, not unmounted: the stream must not reload (SPEC §3.4). -->
    <PlayerPanel
      v-show="!compact && screen === 'timer'"
      :music="music"
      @choose-genre="genreMenuOpen = true"
    />
    <template v-if="state && !compact && screen === 'timer'">
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
    <SettingsPanel
      v-if="store && !compact && screen === 'settings'"
      :settings="store.settings.value"
      :genres="music.genres()"
      :genre-id="music.state.genreId"
      :save="store.patch"
      :autostart="autostart"
      :music-folder="musicFolder"
      :hotkeys="hotkeys"
      @back="screen = 'timer'"
      @hide="WindowService.Hide()"
      @genre="music.selectGenre($event)"
      @always-on-top="setAlwaysOnTop"
      @reset-cycle="TimerService.Reset()"
    />
    <StatsPanel
      v-if="!compact && screen === 'stats'"
      :summary="summary"
      :goal="store?.settings.value.timer.dailyGoal ?? 0"
      @back="screen = 'timer'"
      @hide="WindowService.Hide()"
    />
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
