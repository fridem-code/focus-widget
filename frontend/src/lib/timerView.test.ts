import { describe, expect, it } from 'vitest'
import { Phase, Status, type State } from '../../bindings/focuswidget/internal/timer/models'
import {
  canResetPhase,
  formatTime,
  mainAction,
  phaseLabel,
  pillTone,
  progress,
  sessionNumber,
} from './timerView'

const MIN = 60_000

function state(over: Partial<State> = {}): State {
  return {
    phase: Phase.Focus,
    status: Status.Idle,
    durationMs: 25 * MIN,
    remainingMs: 25 * MIN,
    completedFocus: 0,
    longBreakEvery: 4,
    ...over,
  }
}

describe('formatTime', () => {
  it.each([
    [25 * MIN, '25:00'],
    [18 * MIN + 42_000, '18:42'],
    [1, '00:01'], // rounds up: 0:00 only when the phase is over
    [999, '00:01'],
    [0, '00:00'],
    [-500, '00:00'],
    [180 * MIN, '180:00'],
  ])('%i ms -> %s', (ms, want) => {
    expect(formatTime(ms)).toBe(want)
  })
})

describe('sessionNumber', () => {
  it('is completed + 1', () => {
    expect(sessionNumber(state({ completedFocus: 1 }))).toBe(2)
  })
  it('never exceeds the cycle length (LongBreakEvery lowered mid-cycle)', () => {
    expect(sessionNumber(state({ completedFocus: 5, longBreakEvery: 4 }))).toBe(4)
  })
})

describe('phaseLabel', () => {
  const running = Status.Running
  it.each<[string, Partial<State>, string, string]>([
    ['idle focus', {}, 'Готов к фокусу · сессия 1 из 4', 'Фокус · 1/4'],
    [
      'running focus',
      { status: running, completedFocus: 1 },
      'Фокус · сессия 2 из 4',
      'Фокус · 2/4',
    ],
    [
      'paused focus',
      { status: Status.Paused, completedFocus: 1 },
      'Пауза · фокус 2 из 4',
      'Пауза · фокус 2/4',
    ],
    [
      'short break',
      { phase: Phase.ShortBreak, status: running, durationMs: 5 * MIN },
      'Короткий перерыв',
      'Перерыв · 5 мин',
    ],
    [
      'long break',
      { phase: Phase.LongBreak, status: running, durationMs: 15 * MIN },
      'Длинный перерыв',
      'Длинный перерыв · 15 мин',
    ],
    [
      'paused break',
      { phase: Phase.ShortBreak, status: Status.Paused },
      'Пауза · перерыв',
      'Пауза · перерыв',
    ],
  ])('%s', (_, over, expanded, compact) => {
    expect(phaseLabel(state(over), 'expanded')).toBe(expanded)
    expect(phaseLabel(state(over), 'compact')).toBe(compact)
  })
})

describe('progress', () => {
  it('goes from 0 to 1 as the phase elapses', () => {
    expect(progress(state())).toBe(0)
    expect(progress(state({ remainingMs: 18.75 * MIN }))).toBe(0.25)
    expect(progress(state({ remainingMs: 0 }))).toBe(1)
  })
  it('is safe for a zero duration', () => {
    expect(progress(state({ durationMs: 0, remainingMs: 0 }))).toBe(0)
  })
})

describe('controls', () => {
  it('main action follows the status', () => {
    expect(mainAction(state())).toBe('start')
    expect(mainAction(state({ status: Status.Running }))).toBe('pause')
    expect(mainAction(state({ status: Status.Paused }))).toBe('resume')
  })

  it('reset phase is disabled only for an untouched idle phase', () => {
    expect(canResetPhase(state())).toBe(false)
    expect(canResetPhase(state({ status: Status.Running }))).toBe(true)
    expect(canResetPhase(state({ status: Status.Paused, remainingMs: 10 * MIN }))).toBe(true)
  })

  it('pill is coloured only while running', () => {
    expect(pillTone(state())).toBe('neutral')
    expect(pillTone(state({ status: Status.Running }))).toBe('accent')
    expect(pillTone(state({ status: Status.Paused }))).toBe('neutral')
  })
})
