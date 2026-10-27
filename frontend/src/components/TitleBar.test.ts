import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TitleBar from './TitleBar.vue'

const hide = vi.fn()
vi.mock('../../bindings/focuswidget/services', () => ({ WindowService: { Hide: () => hide() } }))

const mountBar = (props: Partial<{ label: string; done: number; goal: number }> = {}) =>
  mount(TitleBar, { props: { label: '', done: 0, goal: 8, ...props }, attachTo: document.body })

describe('TitleBar', () => {
  it('asks what the user is working on until there is a label', async () => {
    const wrapper = mountBar()
    expect(wrapper.get('.label').text()).toBe('Над чем работаю?')
    await wrapper.setProps({ label: 'Отчёт по Q3' })
    expect(wrapper.get('.label').text()).toBe('Отчёт по Q3')
  })

  it('edits the label: Enter saves it cleaned, Esc keeps the old one', async () => {
    const wrapper = mountBar({ label: 'Отчёт' })
    await wrapper.get('.label').trigger('click')
    const input = wrapper.get('input')
    await input.setValue('  Письма  клиентам ')
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('label')).toEqual([['Письма клиентам']])

    await wrapper.get('.label').trigger('click')
    await wrapper.get('input').setValue('что-то ещё')
    await wrapper.get('input').trigger('keydown', { key: 'Escape' })
    expect(wrapper.emitted('label')).toHaveLength(1)
    expect(wrapper.find('input').exists()).toBe(false)
    wrapper.unmount()
  })

  it('does not save an unchanged label', async () => {
    const wrapper = mountBar({ label: 'Отчёт' })
    await wrapper.get('.label').trigger('click')
    await wrapper.get('input').trigger('blur')
    expect(wrapper.emitted('label')).toBeUndefined()
    wrapper.unmount()
  })

  it('shows the day against the goal and opens the statistics', async () => {
    const wrapper = mountBar({ done: 3, goal: 8 })
    const button = wrapper.get('button.progress')
    expect(button.text()).toBe('3/8')
    expect(button.attributes('aria-label')).toBe('Статистика · сегодня 3 из 8')
    expect(button.classes()).not.toContain('reached')
    await button.trigger('click')
    expect(wrapper.emitted('stats')).toHaveLength(1)

    await wrapper.setProps({ done: 8 })
    expect(wrapper.get('button.progress').classes()).toContain('reached')
    await wrapper.setProps({ done: 2, goal: 0 })
    expect(wrapper.get('button.progress').text()).toBe('2')
    expect(wrapper.get('button.progress').attributes('aria-label')).toBe(
      'Статистика · сегодня 2 помидора',
    )
    wrapper.unmount()
  })

  it('hides to the tray instead of quitting', async () => {
    const wrapper = mountBar()
    await wrapper.get('button[aria-label="Скрыть в трей"]').trigger('click')
    expect(hide).toHaveBeenCalledOnce()
  })

  it('opens the settings', async () => {
    const wrapper = mountBar()
    await wrapper.get('button[aria-label="Настройки"]').trigger('click')
    expect(wrapper.emitted('settings')).toHaveLength(1)
  })

  it('asks for compact mode', async () => {
    const wrapper = mountBar()
    await wrapper.get('button[aria-label="Свернуть в компактный режим"]').trigger('click')
    expect(wrapper.emitted('compact')).toHaveLength(1)
  })
})
