import { describe, expect, it } from 'vitest'
import { NUMBER_SPECS, parseNumber, plural } from './settingsForm'

describe('parseNumber', () => {
  const focus = NUMBER_SPECS.focusMin

  it.each([
    ['25', 25],
    [' 90 ', 90],
    ['1', 1],
    ['180', 180],
  ])('accepts %j', (raw, value) => {
    expect(parseNumber(raw, focus)).toEqual({ ok: true, value })
  })

  it.each(['', 'abc', '2.5', '1e2', '25 мин'])('rejects %j as not a whole number', (raw) => {
    expect(parseNumber(raw, focus)).toEqual({ ok: false, error: 'Введите целое число' })
  })

  it.each(['0', '181', '-5'])('rejects %j as out of range', (raw) => {
    expect(parseNumber(raw, focus)).toEqual({ ok: false, error: 'От 1 до 180' })
  })

  it('uses each field range', () => {
    expect(parseNumber('1', NUMBER_SPECS.longBreakEvery)).toEqual({
      ok: false,
      error: 'От 2 до 10',
    })
    expect(parseNumber('0', NUMBER_SPECS.duckVolume)).toEqual({ ok: true, value: 0 })
  })
})

describe('plural', () => {
  it.each([
    [1, 'сессия'],
    [2, 'сессии'],
    [4, 'сессии'],
    [5, 'сессий'],
    [11, 'сессий'],
    [12, 'сессий'],
    [21, 'сессия'],
    [22, 'сессии'],
  ])('%i %s', (n, word) => {
    expect(plural(n, 'сессия', 'сессии', 'сессий')).toBe(word)
  })
})
