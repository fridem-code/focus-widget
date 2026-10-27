import { describe, expect, it } from 'vitest'
import { keyName, recordKey, showCombo, type KeyPress } from './hotkeys'

const press = (code: string, mods: Partial<KeyPress> = {}): KeyPress => ({
  code,
  ctrlKey: false,
  altKey: false,
  shiftKey: false,
  metaKey: false,
  ...mods,
})

describe('keyName', () => {
  it('names keys by position, so the layout does not matter', () => {
    expect(keyName('KeyP')).toBe('P')
    expect(keyName('Digit7')).toBe('7')
    expect(keyName('F12')).toBe('F12')
    expect(keyName('Space')).toBe('Space')
    expect(keyName('ArrowUp')).toBe('Up')
  })

  it('refuses keys a shortcut cannot use', () => {
    for (const code of ['F13', 'Tab', 'Enter', 'Minus', 'Numpad1', 'Backquote']) {
      expect(keyName(code)).toBeNull()
    }
  })
})

describe('recordKey', () => {
  it('builds the shortcut in a fixed modifier order', () => {
    expect(recordKey(press('KeyP', { shiftKey: true, altKey: true, ctrlKey: true }))).toEqual({
      kind: 'combo',
      combo: 'Ctrl+Alt+Shift+P',
    })
    expect(recordKey(press('F9', { altKey: true }))).toEqual({ kind: 'combo', combo: 'Alt+F9' })
  })

  it('waits while only modifiers are down', () => {
    expect(recordKey(press('ControlLeft', { ctrlKey: true }))).toEqual({ kind: 'wait' })
    expect(recordKey(press('AltRight', { altKey: true, ctrlKey: true }))).toEqual({ kind: 'wait' })
  })

  it('cancels on Esc and clears on Backspace or Delete', () => {
    expect(recordKey(press('Escape'))).toEqual({ kind: 'cancel' })
    expect(recordKey(press('Backspace'))).toEqual({ kind: 'clear' })
    expect(recordKey(press('Delete'))).toEqual({ kind: 'clear' })
  })

  it('needs Ctrl or Alt, and a usable key', () => {
    expect(recordKey(press('KeyP'))).toEqual({ kind: 'invalid', reason: 'Нужен Ctrl или Alt' })
    expect(recordKey(press('KeyP', { shiftKey: true }))).toMatchObject({ kind: 'invalid' })
    expect(recordKey(press('Tab', { ctrlKey: true }))).toEqual({
      kind: 'invalid',
      reason: 'Эту клавишу назначить нельзя',
    })
  })
})

it('shows a shortcut with spaced pluses', () => {
  expect(showCombo('Ctrl+Alt+P')).toBe('Ctrl + Alt + P')
})
