// Validation for the settings screen (docs/SPEC.md §3.6). The ranges mirror
// internal/config so invalid input never reaches Go.

export interface NumberSpec {
  min: number
  max: number
}

export const NUMBER_SPECS = {
  focusMin: { min: 1, max: 180 },
  shortBreakMin: { min: 1, max: 180 },
  longBreakMin: { min: 1, max: 180 },
  longBreakEvery: { min: 2, max: 10 },
  duckVolume: { min: 0, max: 100 },
  dailyGoal: { min: 0, max: 24 },
} as const satisfies Record<string, NumberSpec>

export type NumberField = keyof typeof NUMBER_SPECS

export type Parsed = { ok: true; value: number } | { ok: false; error: string }

/** Parses what the user typed: a whole number within the field's range. */
export function parseNumber(raw: string, spec: NumberSpec): Parsed {
  const text = raw.trim()
  if (!/^-?\d+$/.test(text)) return { ok: false, error: 'Введите целое число' }
  const value = Number(text)
  if (value < spec.min || value > spec.max) {
    return { ok: false, error: `От ${spec.min} до ${spec.max}` }
  }
  return { ok: true, value }
}

/** Russian plural for the unit after a number: 1 сессия, 2 сессии, 5 сессий. */
export function plural(n: number, one: string, few: string, many: string): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return one
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return few
  return many
}
