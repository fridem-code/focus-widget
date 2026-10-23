// Dev-only handle for poking the app from the WebView2 devtools console:
//   await __fw.TimerService.Start()
//   __fw.music.nextTrack(); __fw.music.state
//   __fw.log  // events received so far
import { Events } from '@wailsio/runtime'
import * as AppEvents from '../bindings/events'
import { MusicService, SettingsService, TimerService } from '../bindings/focuswidget/services'
import { useMusic } from './composables/useMusic'
import type { MusicPlayer } from './lib/musicPlayer'

interface DebugHandle {
  TimerService: typeof TimerService
  SettingsService: typeof SettingsService
  MusicService: typeof MusicService
  music: MusicPlayer
  log: { name: string; data: unknown; at: number }[]
}

declare global {
  interface Window {
    __fw?: DebugHandle
  }
}

export function installDebugHandle() {
  const handle: DebugHandle = {
    TimerService,
    SettingsService,
    MusicService,
    music: useMusic(),
    log: [],
  }
  for (const name of [AppEvents.TimerTransition, AppEvents.TimerState, AppEvents.SettingsChanged]) {
    Events.On(name, (ev) => handle.log.push({ name, data: ev.data, at: Date.now() }))
  }
  window.__fw = handle
}
