import { Events } from '@wailsio/runtime'
import { SettingsChanged, TimerTransition } from '../../bindings/events'
import type { Settings } from '../../bindings/focuswidget/internal/config/models'
import type { MusicPlayer } from '../lib/musicPlayer'
import type { PolicySettings } from '../lib/musicPolicy'
import { startMusicPolicy } from '../lib/policyRunner'

const policySettings = (s: Settings): PolicySettings => ({
  breakBehavior: s.music.breakBehavior,
  duckVolume: s.music.duckVolume,
})

/** Connects the music policy to Go's timer:transition and settings:changed events. */
export function connectMusicPolicy(music: MusicPlayer, settings: Settings) {
  return startMusicPolicy({
    music,
    settings: policySettings(settings),
    onTransition: (cb) => Events.On(TimerTransition, (ev) => cb(ev.data)),
    onSettings: (cb) => Events.On(SettingsChanged, (ev) => cb(policySettings(ev.data))),
  })
}
