import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import type { Genre } from '../../bindings/focuswidget/internal/catalog/models'
import {
  BreakBehavior,
  Theme,
  WindowMode,
  type Settings,
} from '../../bindings/focuswidget/internal/config/models'
import { applyTheme, syncGlass, themeIsDark } from '../composables/useSettings'
import { NUMBER_SPECS } from '../lib/settingsForm'
import SettingsPanel from './SettingsPanel.vue'
import SettingNumber from './settings/SettingNumber.vue'
import type { SettingsPatch } from '../../bindings/focuswidget/services/models'

// Only applyTheme is used from useSettings; keep the Wails runtime out.
vi.mock('@wailsio/runtime', () => ({ Events: { On: () => () => {} } }))
vi.mock('../../bindings/events', () => ({}))
vi.mock('../../bindings/focuswidget/services', () => ({ SettingsService: {} }))

const settings = (): Settings => ({
  version: 1,
  timer: {
    focusMin: 25,
    shortBreakMin: 5,
    longBreakMin: 15,
    longBreakEvery: 4,
    autoStartBreaks: false,
    autoStartFocus: false,
    pauseOnLock: true,
    label: '',
    dailyGoal: 8,
  },
  music: {
    volume: 60,
    muted: false,
    shuffle: true,
    breakBehavior: BreakBehavior.BreakPause,
    duckVolume: 20,
    genreId: 'lofi',
    sourceIndex: 0,
    localFolder: '',
  },
  window: {
    mode: WindowMode.WindowExpanded,
    alwaysOnTop: true,
    theme: Theme.ThemeSystem,
    x: null,
    y: null,
  },
  notify: { sound: true, toast: true },
  hotkeys: {
    startPause: 'Ctrl+Alt+Shift+P',
    showHide: 'Ctrl+Alt+Shift+F',
    nextTrack: 'Ctrl+Alt+Shift+N',
  },
})

const genres: Genre[] = [
  { id: 'noise', name: 'Белый шум', sources: [] },
  { id: 'lofi', name: 'Lo-fi', sources: [] },
]

function mountPanel(initial = settings(), result: string | null = null) {
  const saved: SettingsPatch[] = []
  const save = vi.fn(async (patch: SettingsPatch) => {
    saved.push(patch)
    return result
  })
  const autostart = {
    on: false,
    fail: false,
    enabled: vi.fn(async () => autostart.on),
    set: vi.fn(async (on: boolean) => {
      if (autostart.fail) throw new Error('access denied')
      autostart.on = on
    }),
  }
  const musicFolder = {
    choose: vi.fn(async () => 'D:\\Music\\Focus'),
    clear: vi.fn(async () => {}),
  }
  const hotkeys = {
    status: {} as Partial<Record<string, string>>,
    suspend: vi.fn((on: boolean) => void on),
  }
  const wrapper = mount(SettingsPanel, {
    props: { settings: initial, genres, genreId: 'lofi', save, autostart, musicFolder, hotkeys },
  })
  return { wrapper, saved, save, autostart, musicFolder, hotkeys }
}

const flush = () => new Promise((r) => setTimeout(r))

function field(wrapper: ReturnType<typeof mountPanel>['wrapper'], label: string) {
  const l = wrapper.findAll('label').find((x) => x.text() === label)
  if (!l) throw new Error(`no field ${label}`)
  return wrapper.get(`#${CSS.escape(l.attributes('for')!)}`)
}

function switchButton(wrapper: ReturnType<typeof mountPanel>['wrapper'], label: string) {
  const l = wrapper.findAll('span').find((x) => x.text() === label)
  if (!l) throw new Error(`no switch ${label}`)
  return wrapper.get(`button[role="switch"][aria-labelledby="${CSS.escape(l.attributes('id')!)}"]`)
}

// setValue fires input and then change, as typing and pressing Enter would.
describe('SettingNumber', () => {
  const mountNumber = () =>
    mount(SettingNumber, {
      props: { label: 'Фокус', unit: 'мин', value: 25, spec: NUMBER_SPECS.focusMin },
    })

  it('commits a valid number on change', async () => {
    const wrapper = mountNumber()
    const input = wrapper.get('input')
    await input.setValue('50')
    expect(wrapper.emitted('commit')).toEqual([[50]])
  })

  it('flags invalid input while typing and never commits it', async () => {
    const wrapper = mountNumber()
    const input = wrapper.get('input')
    await input.setValue('500')
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('[role="alert"]').text()).toBe('От 1 до 180')
    expect(wrapper.emitted('commit')).toBeUndefined()
  })

  it('does not commit an unchanged value', async () => {
    const wrapper = mountNumber()
    const input = wrapper.get('input')
    await input.setValue('25')
    expect(wrapper.emitted('commit')).toBeUndefined()
  })

  it('follows a value changed elsewhere', async () => {
    const wrapper = mountNumber()
    await wrapper.setProps({ value: 40 })
    expect(wrapper.get('input').element.value).toBe('40')
  })
})

describe('SettingsPanel', () => {
  it('saves a timer length', async () => {
    const { wrapper, saved } = mountPanel()
    const input = field(wrapper, 'Фокус')
    await input.setValue('45')
    await flush()
    expect(saved).toEqual([{ focusMin: 45 }])
  })

  it('does not save an invalid number', async () => {
    const { wrapper, save } = mountPanel()
    const input = field(wrapper, 'Короткий перерыв')
    await input.setValue('0')
    expect(save).not.toHaveBeenCalled()
  })

  it('toggles a switch', async () => {
    const { wrapper, saved } = mountPanel()
    await switchButton(wrapper, 'Автостарт перерывов').trigger('click')
    await switchButton(wrapper, 'Уведомление Windows').trigger('click')
    await flush()
    expect(saved).toEqual([{ autoStartBreaks: true }, { toast: false }])
  })

  it('shows the music folder by name and chooses or forgets it', async () => {
    const { wrapper, musicFolder } = mountPanel()
    expect(wrapper.get('.folder-name').text()).toBe('не выбрана')
    expect(wrapper.find('button[aria-label="Забыть папку"]').exists()).toBe(false)
    await wrapper.get('button.pick').trigger('click')
    expect(musicFolder.choose).toHaveBeenCalledOnce()

    const withFolder = settings()
    withFolder.music.localFolder = 'D:\\Music\\Focus'
    await wrapper.setProps({ settings: withFolder })
    expect(wrapper.get('.folder-name').text()).toBe('Focus')
    expect(wrapper.get('.folder-name').attributes('title')).toBe('D:\\Music\\Focus')
    await wrapper.get('button[aria-label="Забыть папку"]').trigger('click')
    expect(musicFolder.clear).toHaveBeenCalledOnce()
  })

  it('saves the daily goal, 0 included', async () => {
    const { wrapper, saved } = mountPanel()
    await field(wrapper, 'Цель: помидоров в день').setValue('0')
    await flush()
    expect(saved).toEqual([{ dailyGoal: 0 }])
    expect(wrapper.text()).toContain('0 — без цели')
  })

  it('records a shortcut, freeing ours meanwhile', async () => {
    const { wrapper, saved, hotkeys } = mountPanel()
    const button = () => wrapper.findAll('button.combo')[2]!
    expect(button().text()).toBe('Ctrl + Alt + Shift + N')
    await button().trigger('click')
    expect(hotkeys.suspend).toHaveBeenLastCalledWith(true)
    expect(button().text()).toBe('Нажмите сочетание…')

    await button().trigger('keydown', { code: 'KeyN' })
    expect(wrapper.text()).toContain('Нужен Ctrl или Alt')
    await button().trigger('keydown', { code: 'KeyP', ctrlKey: true, altKey: true, shiftKey: true })
    expect(wrapper.text()).toContain('Уже у «Старт / пауза»')
    await button().trigger('keydown', { code: 'F9', altKey: true })
    await flush()
    expect(saved).toEqual([{ hotkeyNextTrack: 'Alt+F9' }])
    expect(hotkeys.suspend).toHaveBeenLastCalledWith(false)
  })

  it('turns a shortcut off and shows one that does not work', async () => {
    const { wrapper, saved } = mountPanel()
    await wrapper.setProps({
      hotkeys: { status: { showHide: 'Занято другим приложением' }, suspend: () => {} },
    })
    expect(wrapper.get('.note.problem').text()).toBe('Занято другим приложением')
    await wrapper.get('button[aria-label="Выключить «Старт / пауза»"]').trigger('click')
    const combo = wrapper.findAll('button.combo')[1]!
    await combo.trigger('click')
    await combo.trigger('keydown', { code: 'Backspace' })
    await flush()
    expect(saved).toEqual([{ hotkeyStartPause: '' }, { hotkeyShowHide: '' }])
  })

  it('switches the pause on lock', async () => {
    const { wrapper, saved } = mountPanel()
    await switchButton(wrapper, 'Пауза при блокировке ПК').trigger('click')
    await flush()
    expect(saved).toEqual([{ pauseOnLock: false }])
  })

  it('saves the break behaviour', async () => {
    const { wrapper, saved } = mountPanel()
    await wrapper.get('[role="radio"][aria-checked="false"]').trigger('click')
    await flush()
    expect(saved).toEqual([{ breakBehavior: BreakBehavior.BreakDuck }])
  })

  it('shows the duck volume only for «Тише»', async () => {
    const { wrapper } = mountPanel()
    expect(wrapper.text()).not.toContain('Громкость «Тише»')
    const ducked = settings()
    ducked.music.breakBehavior = BreakBehavior.BreakDuck
    await wrapper.setProps({ settings: ducked })
    expect(wrapper.text()).toContain('Громкость «Тише»')
  })

  it('saves the theme', async () => {
    const { wrapper, saved } = mountPanel()
    const dark = wrapper.findAll('[role="radio"]').find((r) => r.text() === 'Тёмная')!
    await dark.trigger('click')
    await flush()
    expect(saved).toEqual([{ theme: Theme.ThemeDark }])
  })

  it('hands the genre and always-on-top to the app', async () => {
    const { wrapper, save } = mountPanel()
    await wrapper.get('#s-genre').setValue('noise')
    await switchButton(wrapper, 'Всегда поверх окон').trigger('click')
    expect(wrapper.emitted('genre')).toEqual([['noise']])
    expect(wrapper.emitted('alwaysOnTop')).toEqual([[false]])
    expect(save).not.toHaveBeenCalled()
  })

  it('resets the cycle, goes back and hides', async () => {
    const { wrapper } = mountPanel()
    await wrapper.get('button.reset').trigger('click')
    await wrapper.get('button[aria-label="Назад к таймеру"]').trigger('click')
    await wrapper.get('button[aria-label="Скрыть в трей"]').trigger('click')
    expect(wrapper.emitted('resetCycle')).toHaveLength(1)
    expect(wrapper.emitted('back')).toHaveLength(1)
    expect(wrapper.emitted('hide')).toHaveLength(1)
  })

  it('shows a rejected save', async () => {
    const { wrapper } = mountPanel(settings(), 'timer.focusMin: out of range')
    await switchButton(wrapper, 'Автостарт фокуса').trigger('click')
    await flush()
    await nextTick()
    expect(wrapper.get('.save-error').text()).toContain('out of range')
  })

  it('reads autostart from Windows and switches it', async () => {
    const { wrapper, autostart } = mountPanel()
    expect(wrapper.text()).not.toContain('Запускать вместе с Windows') // until known
    await flush()
    await nextTick()
    const sw = switchButton(wrapper, 'Запускать вместе с Windows')
    expect(sw.attributes('aria-checked')).toBe('false')
    await sw.trigger('click')
    await flush()
    expect(autostart.set).toHaveBeenCalledWith(true)
    expect(sw.attributes('aria-checked')).toBe('true')
  })

  it('shows a failed autostart switch', async () => {
    const { wrapper, autostart } = mountPanel()
    autostart.fail = true
    await flush()
    await nextTick()
    await switchButton(wrapper, 'Запускать вместе с Windows').trigger('click')
    await flush()
    await nextTick()
    expect(wrapper.get('.save-error').text()).toContain('access denied')
    expect(switchButton(wrapper, 'Запускать вместе с Windows').attributes('aria-checked')).toBe(
      'false',
    )
  })

  it('shows «сессий» in the right form', async () => {
    const { wrapper } = mountPanel()
    expect(wrapper.findAll('.unit').map((u) => u.text())).toContain('сессии')
  })
})

describe('window glass', () => {
  it('resolves the theme the page shows', () => {
    expect(themeIsDark(Theme.ThemeDark, false)).toBe(true)
    expect(themeIsDark(Theme.ThemeLight, true)).toBe(false)
    expect(themeIsDark(Theme.ThemeSystem, true)).toBe(true)
    expect(themeIsDark(Theme.ThemeSystem, false)).toBe(false)
  })

  it('marks the page only when the glass is on', async () => {
    const root = document.createElement('html')
    const asked: boolean[] = []
    await syncGlass(true, async (dark) => (asked.push(dark), true), root)
    expect(root.dataset.glass).toBe('on')
    await syncGlass(false, async () => false, root)
    expect(root.dataset.glass).toBeUndefined()
    await syncGlass(false, () => Promise.reject(new Error('no window')), root)
    expect(root.dataset.glass).toBeUndefined()
    expect(asked).toEqual([true])
  })
})

describe('applyTheme', () => {
  it('pins dark and light, and leaves system to the OS', () => {
    const root = document.createElement('html')
    applyTheme(Theme.ThemeDark, root)
    expect(root.dataset.theme).toBe('dark')
    applyTheme(Theme.ThemeSystem, root)
    expect(root.dataset.theme).toBeUndefined()
  })
})
