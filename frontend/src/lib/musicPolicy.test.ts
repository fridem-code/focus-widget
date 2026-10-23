import { describe, expect, it } from 'vitest'
import { BreakBehavior } from '../../bindings/focuswidget/internal/config/models'
import {
  Cause,
  Phase,
  Status,
  type State,
  type Transition,
} from '../../bindings/focuswidget/internal/timer/models'
import {
  initialPolicyContext,
  musicPolicy,
  type MusicAction,
  type PolicyContext,
  type PolicyEvent,
} from './musicPolicy'

const st = (phase: Phase, status: Status): State => ({
  phase,
  status,
  durationMs: 1,
  remainingMs: 1,
  completedFocus: 0,
  longBreakEvery: 4,
})
const tr = (from: [Phase, Status], to: [Phase, Status], cause: Cause): PolicyEvent => ({
  kind: 'transition',
  transition: { from: st(...from), to: st(...to), cause } as Transition,
})

const F = Phase.Focus
const SB = Phase.ShortBreak
const LB = Phase.LongBreak
const { Idle, Running, Paused } = Status
const settings = (breakBehavior: BreakBehavior) => ({ breakBehavior, duckVolume: 20 })

const play: MusicAction = { type: 'play' }
const pause: MusicAction = { type: 'pause' }
const unduck: MusicAction = { type: 'unduck' }
const duck: MusicAction = { type: 'duck', volume: 20 }

function run(
  event: PolicyEvent,
  behavior = BreakBehavior.BreakPause,
  ctx: Partial<PolicyContext> = {},
  playing = true,
) {
  return musicPolicy(event, settings(behavior), { ...initialPolicyContext, ...ctx }, playing)
}

const behaviors = [
  [BreakBehavior.BreakPause, [pause]],
  [BreakBehavior.BreakDuck, [duck]],
  [BreakBehavior.BreakContinue, []],
] as const

describe('SPEC §3.3 table', () => {
  it('start / resume of focus plays at the user volume', () => {
    expect(run(tr([F, Idle], [F, Running], Cause.CauseStart)).actions).toEqual([unduck, play])
  })

  describe.each(behaviors)('focus -> break with %s', (behavior, expected) => {
    it.each([
      ['expire, break auto-started', tr([F, Running], [SB, Running], Cause.CauseExpire)],
      ['expire to a long break waiting for Start', tr([F, Running], [LB, Idle], Cause.CauseExpire)],
      ['skip', tr([F, Running], [SB, Idle], Cause.CauseSkip)],
      ['Start of a break that waited', tr([SB, Idle], [SB, Running], Cause.CauseStart)],
    ])('%s', (_, event) => {
      expect(run(event, behavior).actions).toEqual(expected)
    })
  })

  describe.each(behaviors.map(([b]) => b))('break -> focus with %s', (behavior) => {
    it('auto-started focus plays and restores the volume', () => {
      expect(run(tr([SB, Running], [F, Running], Cause.CauseExpire), behavior).actions).toEqual([
        unduck,
        play,
      ])
    })
    it('focus waiting for Start pauses', () => {
      expect(run(tr([LB, Running], [F, Idle], Cause.CauseExpire), behavior).actions).toEqual([
        unduck,
        pause,
      ])
    })
    it('skipping the break pauses until Start', () => {
      expect(run(tr([SB, Running], [F, Idle], Cause.CauseSkip), behavior).actions).toEqual([
        unduck,
        pause,
      ])
    })
  })

  it.each([
    ['reset phase', Cause.CauseResetPhase],
    ['reset cycle', Cause.CauseReset],
  ])('%s pauses and restores the volume', (_, cause) => {
    expect(run(tr([F, Running], [F, Idle], cause)).actions).toEqual([unduck, pause])
  })
})

describe('timer pause', () => {
  it('pauses playing music and brings it back on resume', () => {
    const paused = run(tr([F, Running], [F, Paused], Cause.CausePause))
    expect(paused.actions).toEqual([pause])
    expect(paused.context.resumeAfterTimerPause).toBe(true)

    const resumed = musicPolicy(
      tr([F, Paused], [F, Running], Cause.CauseResume),
      settings(BreakBehavior.BreakPause),
      paused.context,
      false, // the policy paused it
    )
    expect(resumed.actions).toEqual([play])
    expect(resumed.context.resumeAfterTimerPause).toBe(false)
  })

  it('does not start music on resume if it was not playing at pause', () => {
    const paused = run(tr([F, Running], [F, Paused], Cause.CausePause), undefined, {}, false)
    expect(paused.actions).toEqual([])
    const resumed = musicPolicy(
      tr([F, Paused], [F, Running], Cause.CauseResume),
      settings(BreakBehavior.BreakPause),
      paused.context,
      false,
    )
    expect(resumed.actions).toEqual([])
  })

  it('a reset while paused forgets the pending resume', () => {
    const r = run(tr([F, Paused], [F, Idle], Cause.CauseReset), undefined, {
      resumeAfterTimerPause: true,
    })
    expect(r.context.resumeAfterTimerPause).toBe(false)
  })
})

describe('manual pause is respected', () => {
  const paused = () => run({ kind: 'user', action: 'pause' }, undefined, {}, false).context

  it('records the manual pause without acting', () => {
    const r = run({ kind: 'user', action: 'pause' })
    expect(r.actions).toEqual([])
    expect(r.context.userPaused).toBe(true)
  })

  it('an auto-started focus keeps music off', () => {
    const r = musicPolicy(
      tr([SB, Running], [F, Running], Cause.CauseExpire),
      settings(BreakBehavior.BreakContinue),
      paused(),
      false,
    )
    expect(r.actions).toEqual([unduck])
  })

  it('an explicit Start plays again and clears it', () => {
    const r = musicPolicy(
      tr([F, Idle], [F, Running], Cause.CauseStart),
      settings(BreakBehavior.BreakPause),
      paused(),
      false,
    )
    expect(r.actions).toEqual([unduck, play])
    expect(r.context.userPaused).toBe(false)
  })

  it('pressing play clears it', () => {
    const r = musicPolicy(
      { kind: 'user', action: 'play' },
      settings(BreakBehavior.BreakPause),
      paused(),
      true,
    )
    expect(r.context.userPaused).toBe(false)
  })
})

describe('compact mode (SPEC §3.4)', () => {
  const running = st(F, Running)

  it('entering compact pauses playing music; expanding during a running phase resumes', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: running })
    expect(c.actions).toEqual([pause])
    expect(c.context).toMatchObject({ compact: true, resumeAfterCompact: true })

    const e = musicPolicy(
      { kind: 'mode', mode: 'expanded', timer: running },
      settings(BreakBehavior.BreakPause),
      c.context,
      false,
    )
    expect(e.actions).toEqual([play])
    expect(e.context).toMatchObject({ compact: false, resumeAfterCompact: false })
  })

  it('does not resume when the timer is no longer running', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: running }).context
    const e = musicPolicy(
      { kind: 'mode', mode: 'expanded', timer: st(F, Idle) },
      settings(BreakBehavior.BreakPause),
      c,
      false,
    )
    expect(e.actions).toEqual([])
  })

  it('never plays while compact, but remembers to', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: st(SB, Running) }, undefined, {}, false)
    const r = musicPolicy(
      tr([SB, Running], [F, Running], Cause.CauseExpire),
      settings(BreakBehavior.BreakPause),
      c.context,
      false,
    )
    expect(r.actions).toEqual([unduck])
    expect(r.context.resumeAfterCompact).toBe(true)
  })

  it('a break with pause behaviour while compact cancels the resume', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: running }).context
    const r = musicPolicy(
      tr([F, Running], [SB, Running], Cause.CauseExpire),
      settings(BreakBehavior.BreakPause),
      c,
      false,
    )
    expect(r.actions).toEqual([])
    expect(r.context.resumeAfterCompact).toBe(false)
  })

  it('timer pause and resume while compact work on the remembered state', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: running }).context
    const p = musicPolicy(
      tr([F, Running], [F, Paused], Cause.CausePause),
      settings(BreakBehavior.BreakPause),
      c,
      false,
    )
    expect(p.context).toMatchObject({ resumeAfterTimerPause: true, resumeAfterCompact: false })
    const r = musicPolicy(
      tr([F, Paused], [F, Running], Cause.CauseResume),
      settings(BreakBehavior.BreakPause),
      p.context,
      false,
    )
    expect(r.actions).toEqual([])
    expect(r.context.resumeAfterCompact).toBe(true)
  })

  it('ducking still applies while compact', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: running }).context
    const r = musicPolicy(
      tr([F, Running], [SB, Running], Cause.CauseExpire),
      settings(BreakBehavior.BreakDuck),
      c,
      false,
    )
    expect(r.actions).toEqual([duck])
    expect(r.context.resumeAfterCompact).toBe(true)
  })

  it('repeated mode events are ignored', () => {
    const c = run({ kind: 'mode', mode: 'compact', timer: running }).context
    expect(
      musicPolicy(
        { kind: 'mode', mode: 'compact', timer: running },
        settings(BreakBehavior.BreakPause),
        c,
        false,
      ).actions,
    ).toEqual([])
    expect(run({ kind: 'mode', mode: 'expanded', timer: running }).actions).toEqual([])
  })
})

it('does not mutate the given context', () => {
  const ctx = { ...initialPolicyContext }
  run(tr([F, Running], [F, Paused], Cause.CausePause), undefined, ctx)
  musicPolicy(
    tr([F, Running], [F, Paused], Cause.CausePause),
    settings(BreakBehavior.BreakPause),
    ctx,
    true,
  )
  expect(ctx).toEqual(initialPolicyContext)
})
