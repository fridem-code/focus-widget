import { describe, expect, it } from 'vitest'
import {
  barShare,
  cleanLabel,
  formatMinutes,
  pomodoros,
  progressText,
  streakText,
  weekday,
} from './stats'

describe('stats texts', () => {
  it('agrees «помидор» with the count', () => {
    expect(pomodoros(1)).toBe('1 помидор')
    expect(pomodoros(3)).toBe('3 помидора')
    expect(pomodoros(11)).toBe('11 помидоров')
    expect(pomodoros(0)).toBe('0 помидоров')
  })

  it('formats minutes as hours and minutes', () => {
    expect(formatMinutes(0)).toBe('0 мин')
    expect(formatMinutes(45)).toBe('45 мин')
    expect(formatMinutes(60)).toBe('1 ч')
    expect(formatMinutes(125)).toBe('2 ч 05 мин')
  })

  it('names the weekday of a calendar date', () => {
    expect(weekday('2026-10-26')).toBe('Пн')
    expect(weekday('2026-11-01')).toBe('Вс')
  })

  it('scales bars to the best day', () => {
    const week = [
      { date: 'a', focus: 1, minutes: 25 },
      { date: 'b', focus: 2, minutes: 50 },
    ]
    expect(barShare(week[0]!, week)).toBe(0.5)
    expect(barShare(week[1]!, week)).toBe(1)
    expect(barShare({ date: 'c', focus: 0, minutes: 0 }, [])).toBe(0)
  })

  it('words the streak and the progress', () => {
    expect(streakText(0)).toBe('Серия начнётся с первого помидора')
    expect(streakText(1)).toBe('1 день подряд')
    expect(streakText(5)).toBe('5 дней подряд')
    expect(progressText(3, 8)).toBe('3/8')
    expect(progressText(3, 0)).toBe('3')
  })

  it('cleans a label to one trimmed line of at most 60 characters', () => {
    expect(cleanLabel('  Отчёт \n по Q3 ')).toBe('Отчёт по Q3')
    expect([...cleanLabel('я'.repeat(70))]).toHaveLength(60)
  })
})
