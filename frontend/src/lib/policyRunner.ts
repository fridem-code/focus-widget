import { Cause, type Transition } from '../../bindings/focuswidget/internal/timer/models'
import { FADE_OUT_MS, QUICK_FADE_MS, type MusicPlayer } from './musicPlayer'
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
 * Feeds timer transitions and manual music presses through
 * musicPolicy and carries out its actions on the player.
 */
export function startMusicPolicy(deps: PolicyRunnerDeps) {
  const { music } = deps
  let settings = deps.settings
  let context = initialPolicyContext

  // The timer's changes to the music fade (T18): a long fade-out when a focus
  // phase runs out, a quick one when the user pauses, resets or skips.
  const apply = (a: MusicAction, fadeOutMs: number) => {
    if (a.type === 'play') music.play({ fade: true })
    else if (a.type === 'pause') music.pause({ fadeMs: fadeOutMs })
    else if (a.type === 'duck') music.setDuck(a.volume, { glide: true })
    else music.setDuck(null, { glide: true })
  }

  const handle = (event: PolicyEvent) => {
    const r = musicPolicy(event, settings, context, music.state.wanted)
    context = r.context
    const ranOut = event.kind === 'transition' && event.transition.cause === Cause.CauseExpire
    r.actions.forEach((a) => apply(a, ranOut ? FADE_OUT_MS : QUICK_FADE_MS))
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
    context: () => context,
    stop() {
      offTransition()
      offSettings()
    },
  }
}

export type MusicPolicyRunner = ReturnType<typeof startMusicPolicy>
