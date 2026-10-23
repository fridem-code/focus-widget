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
import { applyTheme } from '../composables/useSettings'
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
  },
  music: {
    volume: 60,
    muted: false,
    shuffle: true,
    breakBehavior: BreakBehavior.BreakPause,
    duckVolume: 20,
    genreId: 'lofi',
    sourceIndex: 0,
  },
  window: {
    mode: WindowMode.WindowExpanded,
    alwaysOnTop: true,
    theme: Theme.ThemeSystem,
    x: null,
    y: null,
  },
  notify: { sound: true, toast: true },
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
  const wrapper = mount(SettingsPanel, {
    props: { settings: initial, genres, genreId: 'lofi', save },
  })
  return { wrapper, saved, save }
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

  it('shows «сессий» in the right form', async () => {
    const { wrapper } = mountPanel()
    expect(wrapper.findAll('.unit').map((u) => u.text())).toContain('сессии')
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
