import { describe, expect, it, vi } from 'vitest'
import {
  Cause,
  Phase,
  Status,
  type State,
  type Transition,
} from '../../bindings/focuswidget/internal/timer/models'
import { createChime, startPhaseSound, type AudioContextLike } from './chime'

function fakeContext() {
  const oscillators: { frequency: { value: number }; started: number; stopped: number }[] = []
  const param = () => ({
    setValueAtTime: vi.fn(),
    linearRampToValueAtTime: vi.fn(),
    exponentialRampToValueAtTime: vi.fn(),
  })
  const ctx = {
    currentTime: 10,
    destination: {},
    resume: vi.fn(() => Promise.resolve()),
    createGain: () => ({ gain: param(), connect: (d: unknown) => d }),
    createOscillator: () => {
      const o = {
        type: '',
        frequency: { value: 0 },
        started: -1,
        stopped: -1,
        connect: (g: unknown) => g,
        start(t: number) {
          o.started = t
        },
        stop(t: number) {
          o.stopped = t
        },
      }
      oscillators.push(o)
      return o
    },
  }
  return { ctx: ctx as unknown as AudioContextLike, oscillators, resume: ctx.resume }
}

describe('createChime', () => {
  it('plays two rising notes and reuses one audio context', () => {
    const { ctx, oscillators, resume } = fakeContext()
    const make = vi.fn(() => ctx)
    const chime = createChime(make)
    chime.play()
    chime.play()
    expect(make).toHaveBeenCalledOnce()
    expect(resume).toHaveBeenCalledTimes(2)
    expect(chime.played()).toBe(2)
    const [a, b] = oscillators
    expect([a!.frequency.value, b!.frequency.value]).toEqual([659.25, 880])
    expect(b!.started).toBeGreaterThan(a!.started)
    expect(a!.stopped).toBeGreaterThan(a!.started)
  })
})

const st = (phase: Phase, status: Status): State => ({
  phase,
  status,
  durationMs: 1,
  remainingMs: 1,
  completedFocus: 0,
  longBreakEvery: 4,
})

describe('startPhaseSound', () => {
  function setup(sound = true) {
    const chime = { play: vi.fn() }
    let emitT!: (tr: Transition) => void
    let emitS!: (on: boolean) => void
    const stop = startPhaseSound({
      chime,
      sound,
      onTransition: (cb) => ((emitT = cb), () => {}),
      onSoundSetting: (cb) => ((emitS = cb), () => {}),
    })
    const transition = (cause: Cause) =>
      emitT({
        from: st(Phase.Focus, Status.Running),
        to: st(Phase.ShortBreak, Status.Running),
        cause,
      })
    return { chime, transition, setSound: (on: boolean) => emitS(on), stop }
  }

  it('chimes only when a phase runs out', () => {
    const { chime, transition } = setup()
    for (const c of [
      Cause.CauseStart,
      Cause.CausePause,
      Cause.CauseResume,
      Cause.CauseSkip,
      Cause.CauseResetPhase,
      Cause.CauseReset,
    ]) {
      transition(c)
    }
    expect(chime.play).not.toHaveBeenCalled()
    transition(Cause.CauseExpire)
    expect(chime.play).toHaveBeenCalledOnce()
  })

  it('follows the sound setting', () => {
    const { chime, transition, setSound } = setup(false)
    transition(Cause.CauseExpire)
    expect(chime.play).not.toHaveBeenCalled()
    setSound(true)
    transition(Cause.CauseExpire)
    expect(chime.play).toHaveBeenCalledOnce()
  })
})
