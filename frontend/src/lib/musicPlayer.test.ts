import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SourceType, type Catalog } from '../../bindings/focuswidget/internal/catalog/models'
import { createMusicPlayer, type MusicConfig } from './musicPlayer'
import { YTState, type YTHandlers, type YTPlayer } from './youtube'

class FakePlayer implements YTPlayer {
  calls: string[] = []
  playlist: string[] | null = null
  playlistIndex = 0
  title = ''
  private log = (c: string) => this.calls.push(c)
  playVideo = () => this.log('play')
  pauseVideo = () => this.log('pause')
  loadVideoById = (id: string) => this.log(`loadVideo ${id}`)
  cueVideoById = (id: string) => this.log(`cueVideo ${id}`)
  loadPlaylist = (o: { list: string }) => this.log(`loadPlaylist ${o.list}`)
  cuePlaylist = (o: { list: string }) => this.log(`cuePlaylist ${o.list}`)
  nextVideo = () => this.log('next')
  playVideoAt = (i: number) => this.log(`playAt ${i}`)
  setShuffle = (on: boolean) => this.log(`shuffle ${on}`)
  setLoop = (on: boolean) => this.log(`loop ${on}`)
  setVolume = (v: number) => this.log(`volume ${v}`)
  mute = () => this.log('mute')
  unMute = () => this.log('unmute')
  getPlaylist = () => this.playlist
  getPlaylistIndex = () => this.playlistIndex
  getVideoData = () => ({ title: this.title })
  destroy = () => this.log('destroy')
}

const video = (id: string) => ({ type: SourceType.SourceVideo, id })
const catalog: Catalog = {
  genres: [
    { id: 'noise', name: 'Шум', sources: [video('vid-aaaaaaa'), video('vid-bbbbbbb')] },
    {
      id: 'lofi',
      name: 'Lo-fi',
      sources: [{ type: SourceType.SourcePlaylist, id: 'PL-lofi' }, video('vid-ccccccc')],
    },
  ],
}
const config = (over: Partial<MusicConfig> = {}): MusicConfig => ({
  catalog,
  genreId: 'noise',
  sourceIndex: 0,
  volume: 40,
  muted: false,
  shuffle: true,
  ...over,
})

let fake: FakePlayer
let handlers: YTHandlers
let online: boolean
let factory: ReturnType<typeof vi.fn>
let target: EventTarget

function setup(cfg = config()) {
  const music = createMusicPlayer({
    factory: factory as never,
    isOnline: () => online,
    target: target as Window,
  })
  music.configure(cfg)
  music.attach(document.createElement('div'))
  return music
}

const flush = () => new Promise((r) => setTimeout(r))

async function started(cfg = config()) {
  const music = setup(cfg)
  music.play()
  await flush()
  handlers.onReady()
  return music
}

beforeEach(() => {
  fake = new FakePlayer()
  online = true
  target = new EventTarget()
  factory = vi.fn(async (_el: HTMLElement, h: YTHandlers) => {
    handlers = h
    return fake
  })
})

describe('creation', () => {
  it('creates the iframe only on the first play, then autoplays the saved source', async () => {
    const music = setup(config({ sourceIndex: 1 }))
    await flush()
    expect(factory).not.toHaveBeenCalled()
    expect(music.state.created).toBe(false)

    music.play()
    music.play() // a double click must not create a second player
    await flush()
    expect(factory).toHaveBeenCalledOnce()
    handlers.onReady()
    expect(fake.calls).toEqual(['volume 40', 'unmute', 'loadVideo vid-bbbbbbb'])
    expect(music.state.created).toBe(true)
  })

  it('waits for configure() when play comes first', async () => {
    const music = createMusicPlayer({
      factory: factory as never,
      isOnline: () => true,
      target: target as Window,
    })
    music.attach(document.createElement('div'))
    music.play()
    await flush()
    expect(factory).not.toHaveBeenCalled()
    music.configure(config())
    await flush()
    expect(factory).toHaveBeenCalledOnce()
  })

  it('clamps a stale saved source index and falls back to the first genre', async () => {
    const music = await started(config({ genreId: 'gone', sourceIndex: 9 }))
    expect(music.state.genreId).toBe('noise')
    expect(fake.calls).toContain('loadVideo vid-aaaaaaa')
  })
})

describe('playback', () => {
  it('reports playing, title and playlist position; configures loop and shuffle once', async () => {
    const music = await started(config({ genreId: 'lofi' }))
    expect(fake.calls).toContain('loadPlaylist PL-lofi')
    fake.playlist = Array.from({ length: 12 }, (_, i) => `v${i}`)
    fake.playlistIndex = 2
    fake.title = 'Lofi mix'
    handlers.onStateChange(YTState.Playing)
    handlers.onStateChange(YTState.Playing)
    expect(music.state.playing).toBe(true)
    expect(music.state.title).toBe('Lofi mix')
    expect(music.state.playlist).toEqual({ index: 2, total: 12 })
    expect(fake.calls.filter((c) => c === 'loop true' || c === 'shuffle true')).toEqual([
      'loop true',
      'shuffle true',
    ])

    handlers.onStateChange(YTState.Paused)
    expect(music.state.playing).toBe(false)
  })

  it('next track: nextVideo in a playlist, next source for videos', async () => {
    const music = await started(config({ genreId: 'lofi' }))
    music.nextTrack()
    expect(fake.calls.at(-1)).toBe('next')

    music.selectGenre('noise')
    music.nextTrack()
    expect(fake.calls.at(-1)).toBe('loadVideo vid-bbbbbbb')
    expect(music.state.sourceIndex).toBe(1)
  })

  it('continues with the next source when a single video ends', async () => {
    const music = await started()
    handlers.onStateChange(YTState.Ended)
    expect(music.state.sourceIndex).toBe(1)
    expect(fake.calls.at(-1)).toBe('loadVideo vid-bbbbbbb')
  })

  it('cues instead of loading after an explicit pause', async () => {
    const music = await started()
    music.pause()
    music.nextTrack()
    expect(fake.calls.at(-1)).toBe('cueVideo vid-bbbbbbb')
  })

  it('changes shuffle on the current playlist only', async () => {
    const music = await started(config({ genreId: 'lofi', shuffle: true }))
    music.setShuffle(false)
    expect(fake.calls.at(-1)).toBe('shuffle false')
    music.selectGenre('noise') // video sources: nothing to shuffle
    music.setShuffle(true)
    expect(fake.calls.at(-1)).toBe('loadVideo vid-aaaaaaa')
  })

  it('applies volume and mute', async () => {
    const music = await started()
    music.setVolume(130)
    music.setMuted(true)
    expect(music.state.volume).toBe(100)
    expect(fake.calls.slice(-4)).toEqual(['volume 100', 'unmute', 'volume 100', 'mute'])
  })
})

describe('ducking', () => {
  it('lowers the player volume without changing the saved volume, and restores it', async () => {
    const music = await started()
    music.setDuck(20)
    expect(fake.calls.at(-2)).toBe('volume 20')
    expect(music.state.volume).toBe(40)

    music.setVolume(60) // the user moves the slider during a ducked break
    expect(fake.calls.at(-2)).toBe('volume 20')
    music.setDuck(null)
    expect(fake.calls.at(-2)).toBe('volume 60')
  })

  it('never raises the volume above the user setting', async () => {
    const music = await started(config({ volume: 10 }))
    music.setDuck(20)
    expect(fake.calls.at(-2)).toBe('volume 10')
  })

  it('applies a duck set before the player is ready', async () => {
    const music = setup()
    music.setDuck(20)
    music.play()
    await flush()
    handlers.onReady()
    expect(fake.calls[0]).toBe('volume 20')
  })
})

describe('errors', () => {
  it('skips a broken video and reports no_videos when the genre is exhausted', async () => {
    const music = await started()
    handlers.onError(150)
    expect(music.state.sourceIndex).toBe(1)
    expect(fake.calls.at(-1)).toBe('loadVideo vid-bbbbbbb')

    handlers.onError(101)
    expect(music.state.error).toBe('no_videos')
    expect(music.state.errorCode).toBe(101)
    expect(music.state.playing).toBe(false)

    music.retry() // retrying the same genre does not help
    expect(music.state.error).toBe('no_videos')
    music.selectGenre('lofi') // another genre does
    expect(music.state.error).toBeNull()
    expect(fake.calls.at(-1)).toBe('loadPlaylist PL-lofi')
  })

  it('skips broken videos inside a playlist before giving up on it', async () => {
    const music = await started(config({ genreId: 'lofi' }))
    fake.playlist = ['a', 'b', 'c', 'd', 'e', 'f', 'g']
    for (let i = 0; i < 4; i++) handlers.onError(150)
    expect(fake.calls.filter((c) => c === 'next')).toHaveLength(4)
    expect(music.state.sourceIndex).toBe(0)

    handlers.onError(150) // fifth in a row: the playlist counts as broken
    expect(music.state.sourceIndex).toBe(1)
    expect(fake.calls.at(-1)).toBe('loadVideo vid-ccccccc')
  })

  it('a playing video resets the playlist error streak', async () => {
    const music = await started(config({ genreId: 'lofi' }))
    fake.playlist = ['a', 'b', 'c', 'd', 'e', 'f', 'g']
    for (let i = 0; i < 4; i++) handlers.onError(150)
    handlers.onStateChange(YTState.Playing)
    handlers.onError(150)
    expect(music.state.sourceIndex).toBe(0)
  })

  it('treats 153 as a configuration error without skipping', async () => {
    const music = await started()
    handlers.onError(153)
    expect(music.state.error).toBe('config')
    expect(music.state.sourceIndex).toBe(0)
    music.retry()
    expect(music.state.error).toBeNull()
    expect(fake.calls.at(-1)).toBe('loadVideo vid-aaaaaaa')
  })

  it('offline at start: no player; the online event brings music back', async () => {
    online = false
    const music = setup()
    music.play()
    await flush()
    expect(music.state.error).toBe('offline')
    expect(factory).not.toHaveBeenCalled()

    online = true
    target.dispatchEvent(new Event('online'))
    await flush()
    expect(factory).toHaveBeenCalledOnce()
    handlers.onReady()
    expect(music.state.error).toBeNull()
    expect(fake.calls.at(-1)).toBe('loadVideo vid-aaaaaaa')
  })

  it('iframe_api failing to load counts as offline', async () => {
    factory.mockRejectedValueOnce(new Error('iframe_api failed to load'))
    const music = setup()
    music.play()
    await flush()
    expect(music.state.error).toBe('offline')
    expect(music.state.created).toBe(false)
  })

  it('losing the network while playing pauses and reloads on reconnect', async () => {
    const music = await started()
    target.dispatchEvent(new Event('offline'))
    expect(music.state.error).toBe('offline')
    expect(fake.calls.at(-1)).toBe('pause')
    target.dispatchEvent(new Event('online'))
    expect(music.state.error).toBeNull()
    expect(fake.calls.at(-1)).toBe('loadVideo vid-aaaaaaa')
  })
})

it('dispose removes listeners and destroys the player', async () => {
  const music = await started()
  music.dispose()
  target.dispatchEvent(new Event('offline'))
  expect(music.state.error).toBeNull()
  expect(fake.calls.at(-1)).toBe('destroy')
})
