// Music controller (docs/SPEC.md §3.2, §3.4, §3.7) over two kinds of source:
// the YouTube player (lazy creation, source/genre switching, error skipping,
// offline recovery) and noise generated in the page (T17).
//
// `wanted` is whether the user wants music; the timer policy (T8) sets it.
// Whether it is heard is decided here: YouTube only plays while its player is
// shown (SPEC §2: not in compact mode, in the tray or behind the settings), so
// a hidden player waits and resumes once shown; generated noise has no player
// and plays either way.
import { reactive, readonly } from 'vue'
import { SourceType, type Catalog } from '../../bindings/focuswidget/internal/catalog/models'
import {
  clampSource,
  classifyError,
  nextGenre,
  nextPlayableSource,
  resolveGenre,
  type PlayerErrorKind,
  type PlaylistPosition,
} from './music'
import { createNoise, type NoiseKind, type NoisePlayer } from './noise'
import { createYouTubePlayer, YTState, type PlayerFactory, type YTPlayer } from './youtube'

export interface MusicConfig {
  catalog: Catalog
  genreId: string
  sourceIndex: number
  volume: number
  muted: boolean
  shuffle: boolean
}

export interface MusicDeps {
  factory?: PlayerFactory
  noise?: NoisePlayer
  isOnline?: () => boolean
  target?: Pick<Window, 'addEventListener' | 'removeEventListener'>
}

/**
 * Inside a playlist, a broken video is skipped with nextVideo(); after this
 * many errors in a row the whole playlist counts as broken.
 */
const MAX_PLAYLIST_ERRORS = 5

export function createMusicPlayer(deps: MusicDeps = {}) {
  const factory = deps.factory ?? createYouTubePlayer
  const noise = deps.noise ?? createNoise()
  const isOnline = deps.isOnline ?? (() => navigator.onLine)
  const target = deps.target ?? window

  const state = reactive({
    configured: false,
    /** The iframe exists (it is created on the first YouTube play, SPEC §3.4). */
    created: false,
    ready: false,
    /** The user wants music (play/Start), even while it is loading or hidden. */
    wanted: false,
    playing: false,
    /** The YouTube player can be seen; a hidden one must not play (SPEC §2). */
    shown: true,
    /** The current source is noise generated in the page, not YouTube. */
    generated: false,
    error: null as PlayerErrorKind | null,
    errorCode: null as number | null,
    title: '',
    genreId: '',
    genreName: '',
    sourceIndex: 0,
    playlist: null as PlaylistPosition | null,
    volume: 50,
    muted: false,
    /** Temporary lower volume on a break ("Тише"); never saved, `volume` stays the user's. */
    duck: null as number | null,
  })

  let catalog: Catalog = { genres: [] }
  let shuffle = true
  let host: HTMLElement | null = null
  let player: YTPlayer | null = null
  let creating = false
  let broken = new Set<number>() // sources of the current genre that failed this session
  let playlistErrors = 0
  let playlistConfigured = false

  const genre = () => resolveGenre(catalog, state.genreId)
  const source = () => genre()?.sources?.[state.sourceIndex]

  /** Call after the genre or source changed. */
  function sourceChanged() {
    const src = source()
    state.generated = src?.type === SourceType.SourceNoise
    state.title = src?.title ?? ''
    state.playlist = null
  }

  function setError(kind: PlayerErrorKind, code: number | null = null) {
    state.error = kind
    state.errorCode = code
    state.playing = false
    if (kind !== 'no_videos') player?.pauseVideo()
  }

  function clearError() {
    state.error = null
    state.errorCode = null
  }

  /** Starts, switches or stops the generated noise to match `wanted`. */
  function syncNoise() {
    if (state.generated && state.wanted) {
      noise.play(source()!.id as NoiseKind)
      state.playing = true
    } else {
      noise.stop()
      if (state.generated) state.playing = false
    }
  }

  async function ensurePlayer() {
    if (player || creating || !host || !state.configured) return
    if (!isOnline()) return setError('offline')
    creating = true
    // YouTube replaces the element it is given, so hand it a child we own
    // rather than an element Vue renders.
    const el = document.createElement('div')
    host.replaceChildren(el)
    try {
      player = await factory(el, { onReady, onStateChange, onError })
      state.created = true
    } catch {
      setError('offline')
    } finally {
      creating = false
    }
  }

  /** Should the YouTube player be playing right now? */
  const youTubeAudible = () => state.wanted && state.shown && !state.generated

  function loadCurrent() {
    if (state.generated) {
      player?.pauseVideo()
      return syncNoise()
    }
    if (noise.playing()) {
      noise.stop()
      state.playing = false // until YouTube reports that it plays
    }
    if (!player || !state.ready) return
    const src = source()
    if (!src) return setError('no_videos')
    playlistErrors = 0
    playlistConfigured = false
    state.playlist = null
    state.title = src.title ?? ''
    const autoplay = youTubeAudible()
    if (src.type === SourceType.SourcePlaylist) {
      const opts = { list: src.id, listType: 'playlist' as const }
      if (autoplay) player.loadPlaylist(opts)
      else player.cuePlaylist(opts)
    } else if (autoplay) {
      player.loadVideoById(src.id)
    } else {
      player.cueVideoById(src.id)
    }
  }

  /** Plays YouTube now: creates the player, recovers from an error or resumes. */
  function playYouTube() {
    if (state.error === 'offline' || state.error === 'config') return retry()
    if (!player) return void ensurePlayer()
    if (state.ready && !state.error) player.playVideo()
  }

  /** Move to the next working source of the genre, or report that none is left. */
  function advance(code: number | null = null) {
    const g = genre()
    const next = g ? nextPlayableSource(g, state.sourceIndex, broken) : null
    if (next === null) return setError('no_videos', code)
    state.sourceIndex = next
    sourceChanged()
    loadCurrent()
  }

  /** The volume heard: the user's, capped by a duck, or silence when muted. */
  const effectiveVolume = () =>
    state.duck === null ? state.volume : Math.min(state.duck, state.volume)

  function applyVolume() {
    noise.setVolume(state.muted ? 0 : effectiveVolume() / 100)
    if (!player || !state.ready) return
    player.setVolume(effectiveVolume())
    if (state.muted) player.mute()
    else player.unMute()
  }

  function updateTrackInfo() {
    if (!player) return
    const title = player.getVideoData()?.title
    if (title) state.title = title
    if (source()?.type === SourceType.SourcePlaylist) {
      const list = player.getPlaylist()
      if (list?.length) {
        state.playlist = { index: Math.max(0, player.getPlaylistIndex()), total: list.length }
      }
    }
  }

  function onReady() {
    state.ready = true
    applyVolume()
    loadCurrent()
  }

  function onStateChange(s: number) {
    // While noise plays, the paused iframe's events are not about the music.
    if (!player || state.generated) return
    if (s === YTState.Playing) {
      state.playing = true
      playlistErrors = 0
      if (state.error !== 'config') clearError()
      if (source()?.type === SourceType.SourcePlaylist && !playlistConfigured) {
        playlistConfigured = true
        player.setLoop(true)
        player.setShuffle(shuffle)
      }
    } else if (s !== YTState.Buffering) {
      state.playing = false
    }
    updateTrackInfo()
    // A single video or stream ended: continue with the next source. Playlists
    // loop by themselves (setLoop above).
    if (s === YTState.Ended && source()?.type === SourceType.SourceVideo && state.wanted) {
      const g = genre()
      const next = g ? nextPlayableSource(g, state.sourceIndex, broken) : null
      if (next !== null) {
        state.sourceIndex = next
        sourceChanged()
        loadCurrent()
      }
    }
  }

  function onError(code: number) {
    if (!player || state.generated) return
    if (classifyError(code) === 'config') return setError('config', code)
    if (source()?.type === SourceType.SourcePlaylist) {
      const total = player.getPlaylist()?.length ?? 0
      playlistErrors++
      if (total > 1 && playlistErrors < Math.min(total, MAX_PLAYLIST_ERRORS)) {
        player.nextVideo()
        return
      }
    }
    broken.add(state.sourceIndex)
    advance(code)
  }

  // The network only matters to YouTube.
  const onOffline = () => {
    if (!state.generated && (state.created || state.wanted)) setError('offline')
  }
  const onOnline = () => {
    if (state.error === 'offline') retry()
  }
  target.addEventListener('offline', onOffline)
  target.addEventListener('online', onOnline)

  function retry() {
    if (state.generated || state.error === 'no_videos') return // only another genre helps
    clearError()
    if (!state.shown) return // plays once shown again
    if (!player) void ensurePlayer()
    else loadCurrent()
  }

  const api = {
    state: readonly(state),

    /** Provide catalog and saved choices (from Go); a play() requested earlier proceeds. */
    configure(cfg: MusicConfig) {
      catalog = cfg.catalog
      shuffle = cfg.shuffle
      const g = resolveGenre(catalog, cfg.genreId)
      state.genreId = g?.id ?? ''
      state.genreName = g?.name ?? ''
      state.sourceIndex = g ? clampSource(g, cfg.sourceIndex) : 0
      sourceChanged()
      state.volume = cfg.volume
      state.muted = cfg.muted
      state.configured = true
      applyVolume()
      if (state.generated) syncNoise()
      else if (youTubeAudible()) void ensurePlayer()
    },

    /** The element the iframe goes into (PlayerPanel's container). */
    attach(el: HTMLElement) {
      host = el
      if (youTubeAudible()) void ensurePlayer()
    },

    play() {
      state.wanted = true
      if (!state.configured) return // configure() starts it
      if (state.generated) return syncNoise()
      if (state.shown) playYouTube()
    },

    pause() {
      state.wanted = false
      player?.pauseVideo()
      syncNoise()
    },

    /**
     * Whether the YouTube player can be seen (App: not compact, not in the
     * tray, not behind the settings). Hiding pauses it before it disappears;
     * showing resumes it if music is wanted. Generated noise is not affected.
     */
    setShown(shown: boolean) {
      if (state.shown === shown) return
      state.shown = shown
      if (state.generated) return
      if (!shown) player?.pauseVideo()
      else if (state.wanted) playYouTube()
    },

    nextTrack() {
      if (state.generated) return advance()
      if (!player || !state.ready || state.error) return
      if (source()?.type === SourceType.SourcePlaylist) player.nextVideo()
      else advance()
    },

    selectGenre(id: string) {
      const g = resolveGenre(catalog, id)
      if (!g) return
      state.genreId = g.id
      state.genreName = g.name
      state.sourceIndex = 0
      broken = new Set()
      clearError()
      sourceChanged()
      loadCurrent() // also stops the noise when switching to YouTube
      if (!player && youTubeAudible()) void ensurePlayer()
    },

    /** The genre being played (reactive through state.genreId). */
    currentGenre: () => genre(),

    /** All genres of the catalog, for the genre list. */
    genres: () => catalog.genres ?? [],

    nextGenre() {
      const g = nextGenre(catalog, state.genreId)
      if (g) api.selectGenre(g.id)
    },

    setVolume(volume: number) {
      state.volume = Math.min(100, Math.max(0, Math.round(volume)))
      applyVolume()
    },

    setMuted(muted: boolean) {
      state.muted = muted
      applyVolume()
    },

    /** Shuffle for playlists: applies to the current playlist at once and to later ones. */
    setShuffle(on: boolean) {
      shuffle = on
      if (player && state.ready && source()?.type === SourceType.SourcePlaylist) {
        player.setShuffle(on)
      }
    },

    /**
     * Plays at most `volume` until called with null, without touching the
     * user's volume. Ducking never makes music louder than the user set it.
     */
    setDuck(volume: number | null) {
      state.duck = volume === null ? null : Math.min(100, Math.max(0, Math.round(volume)))
      applyVolume()
    },

    retry,

    dispose() {
      target.removeEventListener('offline', onOffline)
      target.removeEventListener('online', onOnline)
      noise.dispose()
      player?.destroy()
      player = null
    },
  }
  return api
}

export type MusicPlayer = ReturnType<typeof createMusicPlayer>
