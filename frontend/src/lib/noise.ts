// Noise the app generates itself (T17, docs/SPEC.md §3.2): white, pink and
// brown noise and rain, synthesized with Web Audio like the phase chime. No
// YouTube and no network, so it may play while the player is hidden.

export type NoiseKind = 'white' | 'pink' | 'brown' | 'rain'

export const NOISE_KINDS: readonly NoiseKind[] = ['white', 'pink', 'brown', 'rain']

/** Loop length; the last CROSSFADE_S fades into the start so the loop has no seam. */
const LOOP_S = 8
const CROSSFADE_S = 0.25
/** Every kind is scaled to this RMS, so switching kinds keeps the loudness. */
const TARGET_RMS = 0.18
/** Volume changes glide over about this long instead of clicking. */
const GLIDE_S = 0.08

export type NoiseAudioContext = Pick<
  AudioContext,
  | 'sampleRate'
  | 'currentTime'
  | 'destination'
  | 'createBuffer'
  | 'createBufferSource'
  | 'createGain'
  | 'resume'
  | 'suspend'
  | 'close'
>

/**
 * `length` samples of a noise that loops without a seam. `random` returns
 * values in [0, 1) (Math.random; seeded in tests).
 */
export function noiseSamples(
  kind: NoiseKind,
  length: number,
  sampleRate: number,
  random: () => number = Math.random,
): Float32Array {
  const fade = Math.min(Math.floor(CROSSFADE_S * sampleRate), Math.floor(length / 2))
  const raw = generate(kind, length + fade, sampleRate, random)
  // The extra tail fades into the head: the loop end meets its start.
  const out = raw.slice(0, length)
  for (let i = 0; i < fade; i++) {
    const t = i / fade
    out[i] = raw[i] * t + raw[length + i] * (1 - t)
  }
  return normalise(out)
}

function generate(kind: NoiseKind, n: number, sampleRate: number, random: () => number) {
  const out = new Float32Array(n)
  const white = () => random() * 2 - 1
  if (kind === 'white') {
    for (let i = 0; i < n; i++) out[i] = white()
    return out
  }
  if (kind === 'brown') {
    // Integrated white noise with a leak, so it never drifts away from zero.
    let last = 0
    for (let i = 0; i < n; i++) {
      last = (last + 0.02 * white()) / 1.02
      out[i] = last
    }
    return out
  }
  // Pink: Paul Kellet's filter over white noise.
  let b0 = 0,
    b1 = 0,
    b2 = 0,
    b3 = 0,
    b4 = 0,
    b5 = 0,
    b6 = 0
  for (let i = 0; i < n; i++) {
    const w = white()
    b0 = 0.99886 * b0 + w * 0.0555179
    b1 = 0.99332 * b1 + w * 0.0750759
    b2 = 0.969 * b2 + w * 0.153852
    b3 = 0.8665 * b3 + w * 0.3104856
    b4 = 0.55 * b4 + w * 0.5329522
    b5 = -0.7616 * b5 - w * 0.016898
    out[i] = b0 + b1 + b2 + b3 + b4 + b5 + b6 + w * 0.5362
    b6 = w * 0.115926
  }
  if (kind === 'rain') addDrops(out, sampleRate, random)
  return out
}

/** Rain: soft pink noise with short decaying drops scattered over it. */
function addDrops(out: Float32Array, sampleRate: number, random: () => number) {
  const dropsPerSecond = 120
  const decay = Math.exp(-1 / (0.004 * sampleRate)) // a drop fades in ~4 ms
  let drop = 0
  for (let i = 0; i < out.length; i++) {
    if (random() < dropsPerSecond / sampleRate) drop += 0.4 + random() * 1.6
    out[i] = out[i] * 0.6 + drop * (random() * 2 - 1)
    drop *= decay
  }
}

function normalise(samples: Float32Array) {
  let sum = 0
  for (const s of samples) sum += s * s
  const rms = Math.sqrt(sum / samples.length) || 1
  const k = TARGET_RMS / rms
  for (let i = 0; i < samples.length; i++) samples[i] = Math.max(-1, Math.min(1, samples[i] * k))
  return samples
}

/**
 * A noise player: one looping buffer per kind (generated once), behind a gain
 * for the volume. The audio context sleeps while nothing plays.
 */
export function createNoise(makeContext: () => NoiseAudioContext = () => new AudioContext()) {
  let ctx: NoiseAudioContext | null = null
  let gain: GainNode | null = null
  let source: AudioBufferSourceNode | null = null
  let current: NoiseKind | null = null
  let volume = 0.5
  const buffers = new Map<NoiseKind, AudioBuffer>()

  function context() {
    if (!ctx) {
      ctx = makeContext()
      gain = ctx.createGain()
      gain.gain.value = level(volume)
      gain.connect(ctx.destination)
    }
    return ctx
  }

  function buffer(kind: NoiseKind) {
    const c = context()
    let b = buffers.get(kind)
    if (!b) {
      const length = Math.floor(LOOP_S * c.sampleRate)
      b = c.createBuffer(1, length, c.sampleRate)
      b.getChannelData(0).set(noiseSamples(kind, length, c.sampleRate))
      buffers.set(kind, b)
    }
    return b
  }

  function stopSource() {
    source?.stop()
    source?.disconnect()
    source = null
    current = null
  }

  return {
    /** Plays `kind` (switching from another kind at once). */
    play(kind: NoiseKind) {
      if (current === kind) return
      const c = context()
      stopSource()
      source = c.createBufferSource()
      source.buffer = buffer(kind)
      source.loop = true
      source.connect(gain!)
      source.start()
      current = kind
      void c.resume()
    },

    stop() {
      if (!source) return
      stopSource()
      void ctx?.suspend()
    },

    /** 0…1 on the same scale as the player's 0…100 volume. */
    setVolume(v: number) {
      volume = Math.min(1, Math.max(0, v))
      if (ctx && gain) gain.gain.setTargetAtTime(level(volume), ctx.currentTime, GLIDE_S / 3)
    },

    /** The kind playing, null when stopped. */
    playing: () => current,

    dispose() {
      stopSource()
      void ctx?.close()
      ctx = null
      gain = null
      buffers.clear()
    },
  }
}

/** Perceived loudness grows roughly with the square of the slider. */
function level(volume: number) {
  return volume * volume
}

export type NoisePlayer = ReturnType<typeof createNoise>
