// Thin typed wrapper over the YouTube IFrame Player API
// (https://developers.google.com/youtube/iframe_api_reference).

/** Player states from the IFrame API. */
export const YTState = {
  Unstarted: -1,
  Ended: 0,
  Playing: 1,
  Paused: 2,
  Buffering: 3,
  Cued: 5,
} as const

/** The subset of YT.Player the app uses; a fake implements it in tests. */
export interface YTPlayer {
  playVideo(): void
  pauseVideo(): void
  loadVideoById(id: string): void
  cueVideoById(id: string): void
  loadPlaylist(opts: { list: string; listType: 'playlist'; index?: number }): void
  cuePlaylist(opts: { list: string; listType: 'playlist'; index?: number }): void
  nextVideo(): void
  playVideoAt(index: number): void
  setShuffle(on: boolean): void
  setLoop(on: boolean): void
  setVolume(volume: number): void
  mute(): void
  unMute(): void
  getPlaylist(): string[] | null
  getPlaylistIndex(): number
  getVideoData(): { title?: string; video_id?: string }
  destroy(): void
}

export interface YTHandlers {
  onReady(): void
  onStateChange(state: number): void
  onError(code: number): void
}

/** Creates a player inside `host`; resolves once the IFrame API is loaded. */
export type PlayerFactory = (host: HTMLElement, handlers: YTHandlers) => Promise<YTPlayer>

export const PLAYER_WIDTH = 356
export const PLAYER_HEIGHT = 204
const API_URL = 'https://www.youtube.com/iframe_api'
const API_TIMEOUT_MS = 15_000

declare global {
  interface Window {
    YT?: { Player: new (host: HTMLElement, opts: unknown) => YTPlayer }
    onYouTubeIframeAPIReady?: () => void
  }
}

let apiPromise: Promise<void> | null = null

/**
 * Loads iframe_api once. A failed load (offline) is not cached, so a retry
 * after the network returns loads it again.
 */
export function loadYouTubeApi(): Promise<void> {
  if (window.YT?.Player) return Promise.resolve()
  apiPromise ??= new Promise<void>((resolve, reject) => {
    const script = document.createElement('script')
    const fail = (why: string) => {
      script.remove()
      apiPromise = null
      reject(new Error(why))
    }
    const timer = setTimeout(() => fail('iframe_api load timed out'), API_TIMEOUT_MS)
    window.onYouTubeIframeAPIReady = () => {
      clearTimeout(timer)
      resolve()
    }
    script.src = API_URL
    script.onerror = () => {
      clearTimeout(timer)
      fail('iframe_api failed to load')
    }
    document.head.appendChild(script)
  })
  return apiPromise
}

export const createYouTubePlayer: PlayerFactory = async (host, handlers) => {
  await loadYouTubeApi()
  const YT = window.YT!
  return new YT.Player(host, {
    width: PLAYER_WIDTH,
    height: PLAYER_HEIGHT,
    playerVars: { playsinline: 1, rel: 0, origin: location.origin },
    events: {
      onReady: () => handlers.onReady(),
      onStateChange: (e: { data: number }) => handlers.onStateChange(e.data),
      onError: (e: { data: number }) => handlers.onError(e.data),
    },
  })
}
