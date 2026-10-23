// The phase-end sound (docs/SPEC.md §3.1): a short two-note chime synthesized
// with Web Audio, so there is no sound file to ship or license. Its loudness
// is fixed and independent of the music volume.
import { Cause, type Transition } from '../../bindings/focuswidget/internal/timer/models'

const NOTES_HZ = [659.25, 880] // E5, A5
const NOTE_GAP_S = 0.18
const NOTE_LENGTH_S = 0.6
const PEAK_GAIN = 0.18

export type AudioContextLike = Pick<
  AudioContext,
  'currentTime' | 'destination' | 'createOscillator' | 'createGain' | 'resume'
>

export function createChime(makeContext: () => AudioContextLike = () => new AudioContext()) {
  let ctx: AudioContextLike | null = null
  let played = 0
  return {
    play() {
      ctx ??= makeContext()
      void ctx.resume()
      const t0 = ctx.currentTime
      NOTES_HZ.forEach((hz, i) => {
        const start = t0 + i * NOTE_GAP_S
        const osc = ctx!.createOscillator()
        const gain = ctx!.createGain()
        osc.type = 'sine'
        osc.frequency.value = hz
        gain.gain.setValueAtTime(0, start)
        gain.gain.linearRampToValueAtTime(PEAK_GAIN, start + 0.02)
        gain.gain.exponentialRampToValueAtTime(0.0001, start + NOTE_LENGTH_S)
        osc.connect(gain).connect(ctx!.destination)
        osc.start(start)
        osc.stop(start + NOTE_LENGTH_S + 0.05)
      })
      played++
    },
    /** How many chimes were played (for the dev checks). */
    played: () => played,
  }
}

export type Chime = ReturnType<typeof createChime>

export interface PhaseSoundDeps {
  chime: Pick<Chime, 'play'>
  sound: boolean
  onTransition(cb: (tr: Transition) => void): () => void
  onSoundSetting(cb: (on: boolean) => void): () => void
}

/**
 * Chimes once per phase that runs out. Go emits exactly one expire
 * transition per phase, even after the PC slept, so the chime never repeats.
 */
export function startPhaseSound(deps: PhaseSoundDeps) {
  let sound = deps.sound
  const offT = deps.onTransition((tr) => {
    if (tr.cause === Cause.CauseExpire && sound) deps.chime.play()
  })
  const offS = deps.onSoundSetting((on) => (sound = on))
  return () => {
    offT()
    offS()
  }
}
