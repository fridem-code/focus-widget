import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { SourceType } from '../../bindings/focuswidget/internal/catalog/models'
import { createMusicPlayer } from './musicPlayer'
import { persistMusic, VOLUME_SAVE_DELAY_MS } from './persistMusic'

const catalog = {
  genres: [
    { id: 'a', name: 'A', sources: [{ type: SourceType.SourceVideo, id: 'aaaaaaaaaaa' }] },
    {
      id: 'b',
      name: 'B',
      sources: [
        { type: SourceType.SourceVideo, id: 'bbbbbbbbbbb' },
        { type: SourceType.SourceVideo, id: 'ccccccccccc' },
      ],
    },
  ],
}

function setup() {
  const music = createMusicPlayer({
    factory: vi.fn() as never,
    target: new EventTarget() as Window,
  })
  music.configure({
    catalog,
    genreId: 'a',
    sourceIndex: 0,
    volume: 50,
    muted: false,
    shuffle: true,
  })
  const store = {
    Select: vi.fn(() => Promise.resolve()),
    SaveVolume: vi.fn(() => Promise.resolve()),
  }
  const stop = persistMusic(music, store)
  return { music, store, stop }
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('persistMusic', () => {
  it('does not write back the restored values', async () => {
    const { store } = setup()
    await nextTick()
    vi.advanceTimersByTime(VOLUME_SAVE_DELAY_MS * 2)
    expect(store.Select).not.toHaveBeenCalled()
    expect(store.SaveVolume).not.toHaveBeenCalled()
  })

  it('saves the genre and source right away', async () => {
    const { music, store } = setup()
    music.selectGenre('b')
    await nextTick()
    expect(store.Select).toHaveBeenCalledWith('b', 0)
  })

  it('debounces volume and saves mute with it', async () => {
    const { music, store } = setup()
    for (const v of [40, 30, 20]) {
      music.setVolume(v)
      await nextTick()
    }
    music.setMuted(true)
    await nextTick()
    expect(store.SaveVolume).not.toHaveBeenCalled()
    vi.advanceTimersByTime(VOLUME_SAVE_DELAY_MS)
    expect(store.SaveVolume).toHaveBeenCalledOnce()
    expect(store.SaveVolume).toHaveBeenCalledWith(20, true)
  })

  it('logs a rejected save', async () => {
    const { music, store } = setup()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    store.Select.mockRejectedValueOnce(new Error('unknown genre'))
    music.selectGenre('b')
    await nextTick()
    await vi.runAllTimersAsync()
    expect(warn).toHaveBeenCalledOnce()
    warn.mockRestore()
  })

  it('stops watching', async () => {
    const { music, store, stop } = setup()
    stop()
    music.selectGenre('b')
    music.setVolume(10)
    await nextTick()
    vi.advanceTimersByTime(VOLUME_SAVE_DELAY_MS)
    expect(store.Select).not.toHaveBeenCalled()
    expect(store.SaveVolume).not.toHaveBeenCalled()
  })
})
