import { describe, expect, it, vi } from 'vitest'
import { BreakBehavior } from '../../bindings/focuswidget/internal/config/models'
import {
  Cause,
  Phase,
  Status,
  type State,
  type Transition,
} from '../../bindings/focuswidget/internal/timer/models'
import type { PolicySettings } from './musicPolicy'
import type { MusicPlayer } from './musicPlayer'
import { startMusicPolicy } from './policyRunner'

const st = (phase: Phase, status: Status): State => ({
  phase,
  status,
  durationMs: 1,
  remainingMs: 1,
  completedFocus: 0,
  longBreakEvery: 4,
})

function setup(behavior = BreakBehavior.BreakDuck) {
  const music = {
    state: { wanted: false },
    play: vi.fn(() => (music.state.wanted = true)),
    pause: vi.fn(() => (music.state.wanted = false)),
    setDuck: vi.fn(),
  }
  let emitTransition!: (tr: Transition) => void
  let emitSettings!: (s: PolicySettings) => void
  const offT = vi.fn()
  const offS = vi.fn()
  const runner = startMusicPolicy({
    music: music as unknown as MusicPlayer,
    settings: { breakBehavior: behavior, duckVolume: 20 },
    onTransition: (cb) => ((emitTransition = cb), offT),
    onSettings: (cb) => ((emitSettings = cb), offS),
  })
  const transition = (from: [Phase, Status], to: [Phase, Status], cause: Cause) =>
    emitTransition({ from: st(...from), to: st(...to), cause })
  return {
    music,
    runner,
    transition,
    emitSettings: (s: PolicySettings) => emitSettings(s),
    offT,
    offS,
  }
}

describe('startMusicPolicy', () => {
  it('runs a full cycle: focus plays, break ducks, focus restores', () => {
    const { music, transition } = setup()
    transition([Phase.Focus, Status.Idle], [Phase.Focus, Status.Running], Cause.CauseStart)
    expect(music.setDuck).toHaveBeenLastCalledWith(null)
    expect(music.play).toHaveBeenCalledOnce()

    transition([Phase.Focus, Status.Running], [Phase.ShortBreak, Status.Running], Cause.CauseExpire)
    expect(music.setDuck).toHaveBeenLastCalledWith(20)
    expect(music.pause).not.toHaveBeenCalled()

    transition([Phase.ShortBreak, Status.Running], [Phase.Focus, Status.Running], Cause.CauseExpire)
    expect(music.setDuck).toHaveBeenLastCalledWith(null)
    expect(music.play).toHaveBeenCalledTimes(2)
  })

  it('passes the play intent and keeps the context between events', () => {
    const { music, runner, transition } = setup()
    transition([Phase.Focus, Status.Idle], [Phase.Focus, Status.Running], Cause.CauseStart)
    transition([Phase.Focus, Status.Running], [Phase.Focus, Status.Paused], Cause.CausePause)
    expect(music.pause).toHaveBeenCalledOnce()
    expect(runner.context().resumeAfterTimerPause).toBe(true)
    transition([Phase.Focus, Status.Paused], [Phase.Focus, Status.Running], Cause.CauseResume)
    expect(music.play).toHaveBeenCalledTimes(2)
  })

  it('a manual pause keeps the next auto-started focus silent', () => {
    const { music, runner, transition } = setup(BreakBehavior.BreakContinue)
    runner.user('pause')
    transition([Phase.ShortBreak, Status.Running], [Phase.Focus, Status.Running], Cause.CauseExpire)
    expect(music.play).not.toHaveBeenCalled()
  })

  it('new settings apply from the next transition', () => {
    const { music, transition, emitSettings } = setup(BreakBehavior.BreakDuck)
    emitSettings({ breakBehavior: BreakBehavior.BreakPause, duckVolume: 20 })
    music.state.wanted = true
    transition([Phase.Focus, Status.Running], [Phase.ShortBreak, Status.Running], Cause.CauseExpire)
    expect(music.pause).toHaveBeenCalledOnce()
    expect(music.setDuck).not.toHaveBeenCalledWith(20)
  })

  it('stop unsubscribes', () => {
    const { runner, offT, offS } = setup()
    runner.stop()
    expect(offT).toHaveBeenCalledOnce()
    expect(offS).toHaveBeenCalledOnce()
  })
})
