import type { State, Transition } from '../../bindings/focuswidget/internal/timer/models'
import type { MusicPlayer } from './musicPlayer'
import {
  initialPolicyContext,
  musicPolicy,
  type MusicAction,
  type PolicyEvent,
  type PolicySettings,
} from './musicPolicy'

type Unsubscribe = () => void

export interface PolicyRunnerDeps {
  music: MusicPlayer
  settings: PolicySettings
  onTransition(cb: (tr: Transition) => void): Unsubscribe
  onSettings(cb: (s: PolicySettings) => void): Unsubscribe
}

/**
 * Feeds timer transitions, mode switches and manual music presses through
 * musicPolicy and carries out its actions on the player.
 */
export function startMusicPolicy(deps: PolicyRunnerDeps) {
  const { music } = deps
  let settings = deps.settings
  let context = initialPolicyContext

  const apply = (a: MusicAction) => {
    if (a.type === 'play') music.play()
    else if (a.type === 'pause') music.pause()
    else if (a.type === 'duck') music.setDuck(a.volume)
    else music.setDuck(null)
  }

  const handle = (event: PolicyEvent) => {
    const r = musicPolicy(event, settings, context, music.state.wanted)
    context = r.context
    r.actions.forEach(apply)
  }

  const offTransition = deps.onTransition((transition) =>
    handle({ kind: 'transition', transition }),
  )
  const offSettings = deps.onSettings((s) => {
    settings = s // takes effect from the next transition
  })

  return {
    /** The user pressed play/pause on the music row. */
    user: (action: 'play' | 'pause') => handle({ kind: 'user', action }),
    /** The window switched between expanded and compact (T9). */
    mode: (mode: 'expanded' | 'compact', timer: State) => handle({ kind: 'mode', mode, timer }),
    context: () => context,
    stop() {
      offTransition()
      offSettings()
    },
  }
}

export type MusicPolicyRunner = ReturnType<typeof startMusicPolicy>
