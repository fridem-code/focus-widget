// Dev-only handle for poking the Go services from the WebView2 devtools console:
//   await __fw.TimerService.Start()
//   __fw.log  // events received so far
import { Events } from '@wailsio/runtime'
import * as AppEvents from '../bindings/events'
import { SettingsService, TimerService } from '../bindings/focuswidget/services'

interface DebugHandle {
  TimerService: typeof TimerService
  SettingsService: typeof SettingsService
  log: { name: string; data: unknown; at: number }[]
}

declare global {
  interface Window {
    __fw?: DebugHandle
  }
}

export function installDebugHandle() {
  const handle: DebugHandle = { TimerService, SettingsService, log: [] }
  for (const name of [AppEvents.TimerTransition, AppEvents.TimerState, AppEvents.SettingsChanged]) {
    Events.On(name, (ev) => handle.log.push({ name, data: ev.data, at: Date.now() }))
  }
  window.__fw = handle
}
