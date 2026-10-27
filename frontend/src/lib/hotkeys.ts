// Recording a global shortcut on the settings screen (T26). Shortcuts are
// written as Go's internal/hotkey reads them: "Ctrl+Alt+Shift+P".

/** The actions, in the order the settings list them, keyed as in the settings. */
export const HOTKEY_ACTIONS = [
  { key: 'startPause', patch: 'hotkeyStartPause', label: 'Старт / пауза' },
  { key: 'showHide', patch: 'hotkeyShowHide', label: 'Показать / скрыть' },
  { key: 'nextTrack', patch: 'hotkeyNextTrack', label: 'Следующий трек' },
] as const

export type HotkeyAction = (typeof HOTKEY_ACTIONS)[number]['key']

/** The part of a KeyboardEvent a shortcut is made of. */
export interface KeyPress {
  code: string
  ctrlKey: boolean
  altKey: boolean
  shiftKey: boolean
  metaKey: boolean
}

export type Recorded =
  | { kind: 'combo'; combo: string }
  | { kind: 'cancel' } // Esc
  | { kind: 'clear' } // Backspace or Delete: turn the shortcut off
  | { kind: 'wait' } // only modifiers so far
  | { kind: 'invalid'; reason: string }

const ARROWS: Record<string, string> = {
  ArrowUp: 'Up',
  ArrowDown: 'Down',
  ArrowLeft: 'Left',
  ArrowRight: 'Right',
}

/**
 * The key's name by its position on the keyboard (KeyboardEvent.code), so a
 * shortcut does not depend on the layout: KeyP is "P" in the Russian layout
 * too. Null for keys a shortcut can't use.
 */
export function keyName(code: string): string | null {
  let m: RegExpMatchArray | null
  if ((m = code.match(/^Key([A-Z])$/))) return m[1]!
  if ((m = code.match(/^Digit([0-9])$/))) return m[1]!
  if (/^F([1-9]|1[0-2])$/.test(code)) return code
  if (code === 'Space') return 'Space'
  return ARROWS[code] ?? null
}

const MODIFIER_CODES = /^(Control|Alt|Shift|Meta|OS)(Left|Right)?$/

/** What a key press means while a shortcut is being recorded. */
export function recordKey(e: KeyPress): Recorded {
  const mods = e.ctrlKey || e.altKey || e.shiftKey || e.metaKey
  if (MODIFIER_CODES.test(e.code)) return { kind: 'wait' }
  if (!mods && e.code === 'Escape') return { kind: 'cancel' }
  if (!mods && (e.code === 'Backspace' || e.code === 'Delete')) return { kind: 'clear' }
  const key = keyName(e.code)
  if (!key) return { kind: 'invalid', reason: 'Эту клавишу назначить нельзя' }
  if (!e.ctrlKey && !e.altKey && !e.metaKey) {
    return { kind: 'invalid', reason: 'Нужен Ctrl или Alt' }
  }
  const parts = [
    e.ctrlKey && 'Ctrl',
    e.altKey && 'Alt',
    e.shiftKey && 'Shift',
    e.metaKey && 'Win',
    key,
  ].filter(Boolean)
  return { kind: 'combo', combo: parts.join('+') }
}

/** «Ctrl + Alt + P» for display. */
export function showCombo(combo: string) {
  return combo.split('+').join(' + ')
}
