import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TitleBar from './TitleBar.vue'

const hide = vi.fn()
vi.mock('../../bindings/focuswidget/services', () => ({ WindowService: { Hide: () => hide() } }))

describe('TitleBar', () => {
  it('shows the app name', () => {
    expect(mount(TitleBar).text()).toContain('Focus')
  })

  it('hides to the tray instead of quitting', async () => {
    const wrapper = mount(TitleBar)
    await wrapper.get('button[aria-label="Скрыть в трей"]').trigger('click')
    expect(hide).toHaveBeenCalledOnce()
  })

  it('opens the settings', async () => {
    const wrapper = mount(TitleBar)
    await wrapper.get('button[aria-label="Настройки"]').trigger('click')
    expect(wrapper.emitted('settings')).toHaveLength(1)
  })

  it('asks for compact mode', async () => {
    const wrapper = mount(TitleBar)
    await wrapper.get('button[aria-label="Свернуть в компактный режим"]').trigger('click')
    expect(wrapper.emitted('compact')).toHaveLength(1)
  })
})
