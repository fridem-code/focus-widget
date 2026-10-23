// Pure presentation rules for the timer (docs/SPEC.md §3.1): what the widget
// shows and which controls are available for a given timer state.
import { Phase, Status, type State } from '../../bindings/focuswidget/internal/timer/models'

export type ViewMode = 'expanded' | 'compact'
export type MainAction = 'start' | 'pause' | 'resume'

/** Remaining time as MM:SS, rounded up so 0:00 only shows when the phase is over. */
export function formatTime(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000))
  const minutes = Math.floor(total / 60)
  const seconds = total % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

/** Number of the current (or next) focus session within the cycle, 1-based. */
export function sessionNumber(s: State): number {
  return Math.min(s.completedFocus + 1, s.longBreakEvery)
}

export function phaseLabel(s: State, mode: ViewMode = 'expanded'): string {
  const n = sessionNumber(s)
  const m = s.longBreakEvery
  const focus = s.phase === Phase.Focus
  const compact = mode === 'compact'

  if (s.status === Status.Paused) {
    if (!focus) return 'Пауза · перерыв'
    return compact ? `Пауза · фокус ${n}/${m}` : `Пауза · фокус ${n} из ${m}`
  }
  if (focus) {
    if (compact) return `Фокус · ${n}/${m}`
    return s.status === Status.Idle
      ? `Готов к фокусу · сессия ${n} из ${m}`
      : `Фокус · сессия ${n} из ${m}`
  }
  const name = s.phase === Phase.LongBreak ? 'Длинный перерыв' : 'Короткий перерыв'
  if (!compact) return name
  const minutes = Math.round(s.durationMs / 60_000)
  return s.phase === Phase.LongBreak ? `${name} · ${minutes} мин` : `Перерыв · ${minutes} мин`
}

/** Share of the phase already elapsed, 0..1. */
export function progress(s: State): number {
  if (s.durationMs <= 0) return 0
  return Math.min(1, Math.max(0, 1 - s.remainingMs / s.durationMs))
}

export function mainAction(s: State): MainAction {
  if (s.status === Status.Running) return 'pause'
  if (s.status === Status.Paused) return 'resume'
  return 'start'
}

/** "Сбросить фазу" is pointless for a phase that has not started yet. */
export function canResetPhase(s: State): boolean {
  return !(s.status === Status.Idle && s.remainingMs === s.durationMs)
}

/** The phase pill uses the phase colour only while the phase is running. */
export function pillTone(s: State): 'accent' | 'neutral' {
  return s.status === Status.Running ? 'accent' : 'neutral'
}
