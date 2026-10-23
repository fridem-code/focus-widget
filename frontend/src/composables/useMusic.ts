import { createMusicPlayer, type MusicPlayer } from '../lib/musicPlayer'

let instance: MusicPlayer | null = null

/** The app's single music player (one YouTube iframe for the window's lifetime). */
export function useMusic(): MusicPlayer {
  instance ??= createMusicPlayer()
  return instance
}
