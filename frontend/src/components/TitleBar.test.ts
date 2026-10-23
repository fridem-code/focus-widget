import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TitleBar from './TitleBar.vue'

const quit = vi.fn()
vi.mock('@wailsio/runtime', () => ({ Application: { Quit: () => quit() } }))

describe('TitleBar', () => {
  it('renders the title', () => {
    const wrapper = mount(TitleBar, { props: { title: 'Focus Widget' } })
    expect(wrapper.text()).toContain('Focus Widget')
  })

  it('quits the app on close', async () => {
    const wrapper = mount(TitleBar, { props: { title: 'Focus Widget' } })
    await wrapper.get('button[aria-label="Закрыть"]').trigger('click')
    expect(quit).toHaveBeenCalledOnce()
  })
})
