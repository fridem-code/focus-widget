import { Events } from '@wailsio/runtime'
import { SettingsChanged, TimerTransition } from '../../bindings/events'
import type { Settings } from '../../bindings/focuswidget/internal/config/models'
import { createChime, startPhaseSound, type Chime } from '../lib/chime'

let chime: Chime | null = null

/** The app's chime (one AudioContext for the window's lifetime). */
export function useChime(): Chime {
  chime ??= createChime()
  return chime
}

/** Chimes when a phase runs out, following the notify.sound setting. */
export function connectPhaseSound(settings: Settings) {
  return startPhaseSound({
    chime: useChime(),
    sound: settings.notify.sound,
    onTransition: (cb) => Events.On(TimerTransition, (ev) => cb(ev.data)),
    onSoundSetting: (cb) => Events.On(SettingsChanged, (ev) => cb(ev.data.notify.sound)),
  })
}
