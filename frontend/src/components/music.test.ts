import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { SourceType, type Catalog } from '../../bindings/focuswidget/internal/catalog/models'
import { createMusicPlayer } from '../lib/musicPlayer'
import type { YTHandlers, YTPlayer } from '../lib/youtube'
import MusicRow from './MusicRow.vue'
import PlayerError from './PlayerError.vue'

const catalog: Catalog = {
  genres: [
    {
      id: 'noise',
      name: 'Белый шум',
      sources: [
        { type: SourceType.SourceVideo, id: 'aaaaaaaaaaa', title: 'White noise' },
        { type: SourceType.SourceVideo, id: 'bbbbbbbbbbb', title: 'Brown noise' },
      ],
    },
    {
      id: 'lofi',
      name: 'Lo-fi',
      sources: [{ type: SourceType.SourcePlaylist, id: 'PLlofi12345' }],
    },
    {
      id: 'classical',
      name: 'Классика',
      sources: [{ type: SourceType.SourcePlaylist, id: 'PLclassic12' }],
    },
  ],
}

let handlers: YTHandlers
// Every player method is a no-op. `then` must stay undefined, or `await`
// would take the proxy for a promise.
const fakePlayer = () =>
  new Proxy({} as YTPlayer, {
    get: (_, key) => {
      if (key === 'then') return undefined
      if (key === 'getPlaylist') return () => null
      if (key === 'getVideoData') return () => ({})
      return () => {}
    },
  })

function setup() {
  const music = createMusicPlayer({
    factory: async (_el, h) => {
      handlers = h
      return fakePlayer()
    },
    isOnline: () => true,
    target: new EventTarget() as Window,
  })
  music.configure({
    catalog,
    genreId: 'noise',
    sourceIndex: 0,
    volume: 60,
    muted: false,
    shuffle: true,
  })
  music.attach(document.createElement('div'))
  const wrapper = mount(MusicRow, { props: { music }, attachTo: document.body })
  return { music, wrapper }
}

const btn = (w: VueWrapper, label: string) => w.get(`button[aria-label="${label}"]`)
const flush = () => new Promise((r) => setTimeout(r))
let current: VueWrapper | undefined
afterEach(() => current?.unmount())

describe('MusicRow', () => {
  it('shows the saved source before the player exists', () => {
    const { wrapper } = setup()
    current = wrapper
    expect(wrapper.get('.title').text()).toBe('White noise')
    expect(wrapper.get('.genre').text()).toBe('Белый шум · 1 / 2')
    expect(btn(wrapper, 'Включить музыку').attributes('disabled')).toBeUndefined()
    expect(btn(wrapper, 'Следующее видео').attributes('disabled')).toBeDefined()
    expect(wrapper.get('svg.note').attributes('aria-label')).toBe('Музыка не играет')
  })

  it('play creates the player and flips to pause; next becomes available when ready', async () => {
    const { wrapper } = setup()
    current = wrapper
    await btn(wrapper, 'Включить музыку').trigger('click')
    expect(wrapper.find('button[aria-label="Пауза музыки"]').exists()).toBe(true)
    await btn(wrapper, 'Пауза музыки').trigger('click')
    await btn(wrapper, 'Включить музыку').trigger('click')
    expect(wrapper.emitted('user')).toEqual([['play'], ['pause'], ['play']])
    await flush()
    handlers.onReady()
    await nextTick()
    expect(btn(wrapper, 'Следующее видео').attributes('disabled')).toBeUndefined()
  })

  it('genre list: current genre checked and focused, choosing one switches and closes', async () => {
    const { music, wrapper } = setup()
    current = wrapper
    await wrapper.get('.genre').trigger('click')
    await flush()
    const items = wrapper.findAll('[role=menuitemradio]')
    expect(items.map((i) => i.text())).toEqual(['Белый шум', 'Lo-fi', 'Классика'])
    expect(items[0]!.attributes('aria-checked')).toBe('true')
    expect(document.activeElement).toBe(items[0]!.element)

    await items[2]!.trigger('click')
    expect(music.state.genreId).toBe('classical')
    expect(wrapper.find('[role=menu]').exists()).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('.genre').element)
  })

  it('genre list: arrows move focus, Escape closes and returns focus', async () => {
    const { wrapper } = setup()
    current = wrapper
    await wrapper.get('.genre').trigger('click')
    await flush()
    const menu = wrapper.get('[role=menu]')
    await menu.trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement?.textContent).toContain('Lo-fi')
    await menu.trigger('keydown', { key: 'ArrowUp' })
    await menu.trigger('keydown', { key: 'ArrowUp' })
    expect(document.activeElement?.textContent).toContain('Классика') // wraps

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(wrapper.find('[role=menu]').exists()).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('.genre').element)
  })

  it('closes on a press outside the row', async () => {
    const { wrapper } = setup()
    current = wrapper
    await wrapper.get('.genre').trigger('click')
    document.body.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await nextTick()
    expect(wrapper.find('[role=menu]').exists()).toBe(false)
  })

  it('volume popover changes volume and mute; only one popover at a time', async () => {
    const { music, wrapper } = setup()
    current = wrapper
    await wrapper.get('.genre').trigger('click')
    await btn(wrapper, 'Громкость').trigger('click')
    expect(wrapper.find('[role=menu]').exists()).toBe(false)

    const slider = wrapper.get('input[type=range]')
    expect((slider.element as HTMLInputElement).value).toBe('60')
    await slider.setValue('25')
    expect(music.state.volume).toBe(25)
    await btn(wrapper, 'Выключить звук').trigger('click')
    expect(music.state.muted).toBe(true)
    expect(wrapper.find('button[aria-label="Включить звук"]').exists()).toBe(true)
  })

  it('shows the error state in the row', async () => {
    const { music, wrapper } = setup()
    current = wrapper
    music.play()
    await flush()
    handlers.onReady()
    handlers.onError(150)
    handlers.onError(150)
    await nextTick()
    expect(music.state.error).toBe('no_videos')
    expect(wrapper.get('.title').text()).toBe('Музыка недоступна')
    expect(btn(wrapper, 'Включить музыку').attributes('disabled')).toBeDefined()
  })
})

describe('PlayerError', () => {
  it.each([
    ['no_videos', 'Нет доступных видео', 'Выбрать другой жанр', 'chooseGenre'],
    ['offline', 'Нет подключения к интернету', 'Повторить', 'retry'],
    ['config', 'Плеер не может загрузить видео', 'Повторить', 'retry'],
  ] as const)('%s', async (kind, title, action, event) => {
    const wrapper = mount(PlayerError, { props: { kind, code: 153, genreName: 'Lo-fi' } })
    expect(wrapper.get('.title').text()).toBe(title)
    await wrapper.get('button').trigger('click')
    expect(wrapper.get('button').text()).toBe(action)
    expect(wrapper.emitted(event)).toHaveLength(1)
  })

  it('names the genre for no_videos', () => {
    const wrapper = mount(PlayerError, {
      props: { kind: 'no_videos', code: 150, genreName: 'Lo-fi' },
    })
    expect(wrapper.get('.text').text()).toContain('«Lo-fi»')
  })
})
