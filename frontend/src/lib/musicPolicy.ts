// How the timer drives the music (docs/SPEC.md §3.3), as a pure function:
// event + settings + context -> actions + context. It decides whether music is
// wanted; whether a hidden YouTube player may sound is the player's rule
// (musicPlayer setShown, SPEC §2).
import { BreakBehavior } from '../../bindings/focuswidget/internal/config/models'
import {
  Cause,
  Phase,
  Status,
  type Transition,
} from '../../bindings/focuswidget/internal/timer/models'

export type PolicyEvent =
  | { kind: 'transition'; transition: Transition }
  /** The user pressed play/pause on the music row. */
  | { kind: 'user'; action: 'play' | 'pause' }

export interface PolicySettings {
  breakBehavior: BreakBehavior
  duckVolume: number
}

export interface PolicyContext {
  /** Music was paused by hand: automation won't start it until Start/Play. */
  userPaused: boolean
  /** Music was playing when the timer was paused, so Resume brings it back. */
  resumeAfterTimerPause: boolean
}

export type MusicAction =
  { type: 'play' } | { type: 'pause' } | { type: 'duck'; volume: number } | { type: 'unduck' }

export const initialPolicyContext: PolicyContext = {
  userPaused: false,
  resumeAfterTimerPause: false,
}

const play: MusicAction = { type: 'play' }
const pause: MusicAction = { type: 'pause' }
const unduck: MusicAction = { type: 'unduck' }

function breakActions(s: PolicySettings): MusicAction[] {
  switch (s.breakBehavior) {
    case BreakBehavior.BreakDuck:
      return [{ type: 'duck', volume: s.duckVolume }]
    case BreakBehavior.BreakContinue:
      return []
    default:
      return [pause]
  }
}

function onTransition(
  tr: Transition,
  s: PolicySettings,
  ctx: PolicyContext,
  playing: boolean,
): MusicAction[] {
  const toFocus = tr.to.phase === Phase.Focus
  switch (tr.cause) {
    case Cause.CauseStart:
      if (!toFocus) return breakActions(s) // a break that waited for Start
      ctx.userPaused = false // an explicit Start overrides a manual pause
      return [unduck, play]
    case Cause.CausePause:
      ctx.resumeAfterTimerPause = playing
      return playing ? [pause] : []
    case Cause.CauseResume: {
      const resume = ctx.resumeAfterTimerPause
      ctx.resumeAfterTimerPause = false
      return resume ? [play] : []
    }
    case Cause.CauseExpire:
    case Cause.CauseSkip:
      if (tr.from.phase === Phase.Focus && !toFocus) return breakActions(s)
      if (tr.from.phase !== Phase.Focus && toFocus) {
        if (tr.to.status !== Status.Running) return [unduck, pause] // waits for Start
        return ctx.userPaused ? [unduck] : [unduck, play]
      }
      return []
    case Cause.CauseResetPhase:
    case Cause.CauseReset:
      ctx.resumeAfterTimerPause = false
      return [unduck, pause]
    default:
      return []
  }
}

/**
 * Decides what the music does for an event. `musicPlaying` is whether music
 * is currently wanted (the player's play intent).
 */
export function musicPolicy(
  event: PolicyEvent,
  settings: PolicySettings,
  context: PolicyContext,
  musicPlaying: boolean,
): { actions: MusicAction[]; context: PolicyContext } {
  const ctx = { ...context }
  if (event.kind === 'user') {
    ctx.userPaused = event.action === 'pause'
    return { actions: [], context: ctx } // the row already played/paused
  }
  return { actions: onTransition(event.transition, settings, ctx, musicPlaying), context: ctx }
}
