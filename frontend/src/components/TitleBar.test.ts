import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TitleBar from './TitleBar.vue'

const quit = vi.fn()
vi.mock('@wailsio/runtime', () => ({ Application: { Quit: () => quit() } }))

describe('TitleBar', () => {
  it('shows the app name', () => {
    expect(mount(TitleBar).text()).toContain('Focus')
  })

  it('quits the app on close', async () => {
    const wrapper = mount(TitleBar)
    await wrapper.get('button[aria-label="Закрыть"]').trigger('click')
    expect(quit).toHaveBeenCalledOnce()
  })

  it('keeps settings and compact mode disabled until they are implemented', () => {
    const wrapper = mount(TitleBar)
    expect(wrapper.get('button[aria-label="Настройки"]').attributes('disabled')).toBeDefined()
    expect(
      wrapper.get('button[aria-label="Свернуть в компактный режим"]').attributes('disabled'),
    ).toBeDefined()
  })
})
