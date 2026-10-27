import { describe, expect, it } from 'vitest'
import { createFilePlayer, type AudioLike } from './filePlayer'

class FakeAudio {
  src = ''
  volume = 1
  log: string[] = []
  listeners: Record<string, () => void> = {}
  rejectPlay = false
  play = () => {
    this.log.push(`play ${this.src}`)
    return this.rejectPlay ? Promise.reject(new Error('no')) : Promise.resolve()
  }
  pause = () => void this.log.push('pause')
  removeAttribute = (name: string) => {
    if (name === 'src') this.src = ''
  }
  load = () => void this.log.push('load')
  addEventListener = (type: string, fn: () => void) => {
    this.listeners[type] = fn
  }
  fire(type: string) {
    this.listeners[type]?.()
  }
}

function setup() {
  const audio = new FakeAudio()
  let made = 0
  const files = createFilePlayer(() => (made++, audio as unknown as AudioLike))
  const events: string[] = []
  files.on({ ended: () => events.push('ended'), failed: () => events.push('failed') })
  return { audio, files, events, made: () => made }
}

describe('createFilePlayer', () => {
  it('creates the element on the first play and plays the url', () => {
    const { audio, files, made } = setup()
    expect(made()).toBe(0)
    files.play('/local-music/3')
    expect(made()).toBe(1)
    expect(audio.src).toBe('/local-music/3')
    expect(files.current()).toBe('/local-music/3')
  })

  it('resumes the same track and switches to another', () => {
    const { audio, files } = setup()
    files.play('/local-music/1')
    files.pause()
    audio.src = '/local-music/1#t=42' // the element keeps its own position
    files.play('/local-music/1')
    expect(audio.src).toBe('/local-music/1#t=42')
    files.play('/local-music/2')
    expect(audio.src).toBe('/local-music/2')
  })

  it('stops and forgets the track', () => {
    const { audio, files } = setup()
    files.play('/local-music/1')
    files.stop()
    expect(files.current()).toBeNull()
    expect(audio.src).toBe('')
    expect(audio.log.slice(-2)).toEqual(['pause', 'load'])
  })

  it('reports the end and failures of the current track only', () => {
    const { audio, files, events } = setup()
    files.play('/local-music/1')
    audio.fire('ended')
    audio.fire('error')
    files.stop()
    audio.fire('error') // clearing src fires error too: not a failure
    expect(events).toEqual(['ended', 'failed'])
  })

  it('sets the volume on a squared scale, also before the element exists', () => {
    const { audio, files } = setup()
    files.setVolume(0.5)
    files.play('/local-music/1')
    expect(audio.volume).toBeCloseTo(0.25)
    files.setVolume(2)
    expect(audio.volume).toBe(1)
  })

  it('swallows a rejected play', async () => {
    const { audio, files } = setup()
    audio.rejectPlay = true
    files.play('/local-music/1')
    await new Promise((r) => setTimeout(r))
  })
})
