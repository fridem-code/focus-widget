// Texts for the statistics screen, the header and the label (T23–T25).
import type { Day } from '../../bindings/focuswidget/internal/stats/models'
import { plural } from './settingsForm'

/** «помидора» for 3: the noun agreeing with a count. */
export function pomodoroWord(n: number) {
  return plural(n, 'помидор', 'помидора', 'помидоров')
}

/** «3 помидора». */
export function pomodoros(n: number) {
  return `${n} ${pomodoroWord(n)}`
}

/** «45 мин», «1 ч», «2 ч 05 мин». */
export function formatMinutes(total: number) {
  const h = Math.floor(total / 60)
  const m = total % 60
  if (h === 0) return `${m} мин`
  return m === 0 ? `${h} ч` : `${h} ч ${String(m).padStart(2, '0')} мин`
}

const WEEKDAYS = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб']

/** «Пн» for "2026-10-26": the date is a calendar day, not a moment. */
export function weekday(date: string) {
  const [y, m, d] = date.split('-').map(Number)
  return WEEKDAYS[new Date(y!, m! - 1, d!).getDay()]!
}

/** «26 октября» for the bar tooltips. */
export function dayName(date: string) {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y!, m! - 1, d!).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
}

/** The bar's height as a share of the week's best day; 0 for an empty week. */
export function barShare(day: Day, week: Day[]) {
  const top = Math.max(0, ...week.map((d) => d.minutes))
  return top === 0 ? 0 : day.minutes / top
}

/** «5 дней подряд», or what to say before a streak starts. */
export function streakText(days: number) {
  if (days === 0) return 'Серия начнётся с первого помидора'
  return `${days} ${plural(days, 'день', 'дня', 'дней')} подряд`
}

/** «3/8» with a goal, «3» without; for the header button. */
export function progressText(done: number, goal: number) {
  return goal > 0 ? `${done}/${goal}` : String(done)
}

/** The label as it is saved: trimmed, one line, at most `max` characters. */
export function cleanLabel(text: string, max = 60) {
  return [...text.replace(/\s+/g, ' ').trim()].slice(0, max).join('')
}

/** «14:25» for a session's end, in local time. */
export function clockTime(iso: string) {
  return new Date(iso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}
