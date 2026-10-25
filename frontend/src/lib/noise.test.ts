import { describe, expect, it } from 'vitest'
import { createNoise, NOISE_KINDS, noiseSamples, type NoiseAudioContext } from './noise'

// A deterministic [0, 1) generator, so the samples are the same every run.
function seeded(seed = 1) {
  let s = seed
  return () => {
    s = (s * 1664525 + 1013904223) % 4294967296
    return s / 4294967296
  }
}

const RATE = 8000
const rms = (a: Float32Array) => Math.sqrt(a.reduce((s, x) => s + x * x, 0) / a.length)

describe('noiseSamples', () => {
  it.each(NOISE_KINDS)('%s has the target loudness and stays in range', (kind) => {
    const s = noiseSamples(kind, RATE * 2, RATE, seeded())
    expect(s).toHaveLength(RATE * 2)
    expect(rms(s)).toBeGreaterThan(0.15)
    expect(rms(s)).toBeLessThan(0.2)
    expect(Math.max(...s.map(Math.abs))).toBeLessThanOrEqual(1)
  })

  it.each(NOISE_KINDS)('%s loops without a jump at the seam', (kind) => {
    const s = noiseSamples(kind, RATE * 2, RATE, seeded(7))
    const seam = Math.abs(s[s.length - 1] - s[0])
    // No bigger than the typical step between neighbouring samples.
    let steps = 0
    for (let i = 1; i < s.length; i++) steps += Math.abs(s[i] - s[i - 1])
    expect(seam).toBeLessThan((steps / (s.length - 1)) * 6)
  })

  it('brown noise is darker than white: smaller steps between samples', () => {
    const step = (a: Float32Array) => {
      let d = 0
      for (let i = 1; i < a.length; i++) d += Math.abs(a[i] - a[i - 1])
      return d / (a.length - 1)
    }
    const white = noiseSamples('white', RATE, RATE, seeded(3))
    const pink = noiseSamples('pink', RATE, RATE, seeded(3))
    const brown = noiseSamples('brown', RATE, RATE, seeded(3))
    expect(step(pink)).toBeLessThan(step(white))
    expect(step(brown)).toBeLessThan(step(pink))
  })
})

class FakeParam {
  value = 0
  targets: number[] = []
  setTargetAtTime(v: number) {
    this.targets.push(v)
    this.value = v
  }
}

function fakeContext() {
  const log: string[] = []
  const gain = { gain: new FakeParam(), connect: () => {} }
  let buffers = 0
  const ctx = {
    sampleRate: RATE,
    currentTime: 0,
    destination: {},
    createBuffer: (_c: number, length: number) => {
      buffers++
      const data = new Float32Array(length)
      return { getChannelData: () => data }
    },
    createBufferSource: () => {
      const src = {
        buffer: null as unknown,
        loop: false,
        connect: () => {},
        disconnect: () => {},
        start: () => log.push(`start loop=${src.loop}`),
        stop: () => log.push('stop'),
      }
      return src
    },
    createGain: () => gain,
    resume: async () => void log.push('resume'),
    suspend: async () => void log.push('suspend'),
    close: async () => void log.push('close'),
  }
  return { ctx: ctx as unknown as NoiseAudioContext, log, gain, buffers: () => buffers }
}

describe('createNoise', () => {
  it('creates the audio context only on the first play and loops the noise', () => {
    const f = fakeContext()
    let made = 0
    const noise = createNoise(() => (made++, f.ctx))
    expect(made).toBe(0)
    noise.play('pink')
    expect(made).toBe(1)
    expect(noise.playing()).toBe('pink')
    expect(f.log).toEqual(['start loop=true', 'resume'])
  })

  it('switches kinds, generates each buffer once and ignores a repeated play', () => {
    const f = fakeContext()
    const noise = createNoise(() => f.ctx)
    noise.play('white')
    noise.play('white')
    noise.play('rain')
    noise.play('white')
    expect(f.buffers()).toBe(2)
    expect(f.log.filter((l) => l.startsWith('start'))).toHaveLength(3)
    expect(noise.playing()).toBe('white')
  })

  it('stops and lets the context sleep', () => {
    const f = fakeContext()
    const noise = createNoise(() => f.ctx)
    noise.stop() // nothing playing: no context needed
    noise.play('brown')
    noise.stop()
    expect(noise.playing()).toBeNull()
    expect(f.log.slice(-2)).toEqual(['stop', 'suspend'])
  })

  it('glides the volume on a squared scale', () => {
    const f = fakeContext()
    const noise = createNoise(() => f.ctx)
    noise.setVolume(0.8) // before the context exists: used when it is created
    noise.play('white')
    expect(f.gain.gain.value).toBeCloseTo(0.64)
    noise.setVolume(0.5)
    expect(f.gain.gain.targets).toEqual([0.25])
    noise.setVolume(2)
    expect(f.gain.gain.value).toBe(1)
  })
})
