// How the timer drives the music (docs/SPEC.md §3.3) and the compact-mode rule
// (§3.4), as a pure function: event + settings + context -> actions + context.
import { BreakBehavior } from '../../bindings/focuswidget/internal/config/models'
import {
  Cause,
  Phase,
  Status,
  type State,
  type Transition,
} from '../../bindings/focuswidget/internal/timer/models'

export type PolicyEvent =
  | { kind: 'transition'; transition: Transition }
  /** The window switched mode (T9); `timer` is the timer state at that moment. */
  | { kind: 'mode'; mode: 'expanded' | 'compact'; timer: State }
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
  compact: boolean
  /** In compact mode: music should play once the window expands again. */
  resumeAfterCompact: boolean
}

export type MusicAction =
  { type: 'play' } | { type: 'pause' } | { type: 'duck'; volume: number } | { type: 'unduck' }

export const initialPolicyContext: PolicyContext = {
  userPaused: false,
  resumeAfterTimerPause: false,
  compact: false,
  resumeAfterCompact: false,
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
 * is currently wanted (the player's play intent). In compact mode the player
 * is always paused (SPEC §3.4): play/pause decisions are remembered in
 * `resumeAfterCompact` and carried out when the window expands.
 */
export function musicPolicy(
  event: PolicyEvent,
  settings: PolicySettings,
  context: PolicyContext,
  musicPlaying: boolean,
): { actions: MusicAction[]; context: PolicyContext } {
  const ctx = { ...context }
  // While compact, "playing" means "would be playing once expanded".
  const playing = ctx.compact ? ctx.resumeAfterCompact : musicPlaying
  let actions: MusicAction[]

  if (event.kind === 'user') {
    ctx.userPaused = event.action === 'pause'
    return { actions: [], context: ctx } // the row already played/paused
  }

  if (event.kind === 'mode') {
    if (event.mode === 'compact') {
      if (ctx.compact) return { actions: [], context: ctx }
      ctx.compact = true
      ctx.resumeAfterCompact = musicPlaying
      return { actions: musicPlaying ? [pause] : [], context: ctx }
    }
    if (!ctx.compact) return { actions: [], context: ctx }
    ctx.compact = false
    const resume = ctx.resumeAfterCompact && event.timer.status === Status.Running
    ctx.resumeAfterCompact = false
    return { actions: resume ? [play] : [], context: ctx }
  }

  actions = onTransition(event.transition, settings, ctx, playing)
  if (ctx.compact) {
    // Keep the player paused; remember what should happen on expand.
    for (const a of actions) {
      if (a.type === 'play') ctx.resumeAfterCompact = true
      if (a.type === 'pause') ctx.resumeAfterCompact = false
    }
    actions = actions.filter((a) => a.type !== 'play' && a.type !== 'pause')
  }
  return { actions, context: ctx }
}
