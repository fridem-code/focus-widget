// Plays the user's music files (T22, docs/SPEC.md §3.2) through one <audio>
// element. Go serves each file at /local-music/<number>. Like generated noise
// it is not the YouTube player, so it may play while the widget is hidden.

/** The part of HTMLAudioElement used here. */
export type AudioLike = Pick<
  HTMLAudioElement,
  'src' | 'volume' | 'play' | 'pause' | 'removeAttribute' | 'load' | 'addEventListener'
>

export interface FileHandlers {
  /** The track played to its end. */
  ended(): void
  /** The track could not be played (unreadable, unsupported, gone). */
  failed(): void
}

export function createFilePlayer(makeAudio: () => AudioLike = () => new Audio()) {
  let audio: AudioLike | null = null
  let current: string | null = null
  let handlers: FileHandlers = { ended: () => {}, failed: () => {} }
  let volume = 0.5

  function element() {
    if (!audio) {
      audio = makeAudio()
      audio.volume = level(volume)
      audio.addEventListener('ended', () => handlers.ended())
      audio.addEventListener('error', () => {
        if (current) handlers.failed()
      })
    }
    return audio
  }

  return {
    on(h: FileHandlers) {
      handlers = h
    },

    /** Plays `url` from its start, or resumes it if it is the current one. */
    play(url: string) {
      const a = element()
      if (current !== url) {
        current = url
        a.src = url
      }
      // A rejected play() (e.g. a file that can't be decoded) also fires
      // the element's error event, which reports it.
      a.play().catch(() => {})
    },

    /** Pauses; play() with the same url resumes where it stopped. */
    pause() {
      audio?.pause()
    },

    /** Stops and forgets the track. */
    stop() {
      if (!audio || current === null) return
      audio.pause()
      current = null
      audio.removeAttribute('src')
      audio.load()
    },

    /** 0…1 on the same scale as the player's 0…100 volume. */
    setVolume(v: number) {
      volume = Math.min(1, Math.max(0, v))
      if (audio) audio.volume = level(volume)
    },

    /** The url playing or paused, null when stopped. */
    current: () => current,
  }
}

/** Perceived loudness grows roughly with the square of the slider (as for the noise). */
function level(volume: number) {
  return volume * volume
}

export type FilePlayer = ReturnType<typeof createFilePlayer>
