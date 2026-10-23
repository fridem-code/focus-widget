import { watch, type WatchStopHandle } from 'vue'
import type { MusicPlayer } from './musicPlayer'

export interface MusicStore {
  Select(genreId: string, sourceIndex: number): Promise<void>
  SaveVolume(volume: number, muted: boolean): Promise<void>
}

/** Volume drags fire many input events; save once the slider settles. */
export const VOLUME_SAVE_DELAY_MS = 500

/**
 * Saves genre, source, volume and mute through Go whenever they change
 * (SPEC §3.2). Call after music.configure() so the restored values are not
 * written straight back. A rejected save is logged; playback is unaffected.
 */
export function persistMusic(music: MusicPlayer, store: MusicStore): WatchStopHandle {
  const log = (err: unknown) => console.warn('saving music settings failed:', err)
  let volumeTimer: ReturnType<typeof setTimeout> | undefined

  const stopSource = watch(
    () => [music.state.genreId, music.state.sourceIndex] as const,
    ([genreId, sourceIndex]) => {
      store.Select(genreId, sourceIndex).catch(log)
    },
  )
  const stopVolume = watch(
    () => [music.state.volume, music.state.muted] as const,
    ([volume, muted]) => {
      clearTimeout(volumeTimer)
      volumeTimer = setTimeout(
        () => store.SaveVolume(volume, muted).catch(log),
        VOLUME_SAVE_DELAY_MS,
      )
    },
  )
  return () => {
    stopSource()
    stopVolume()
    clearTimeout(volumeTimer)
  }
}
