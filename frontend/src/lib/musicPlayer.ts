// Music controller over the YouTube player (docs/SPEC.md §3.2, §3.4, §3.7):
// lazy creation, source/genre switching, error skipping and offline recovery.
// Timer-driven behaviour (pause on break, …) is T8's musicPolicy, not here.
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
  const isOnline = deps.isOnline ?? (() => navigator.onLine)
  const target = deps.target ?? window

  const state = reactive({
    configured: false,
    /** The iframe exists (it is created on the first play, SPEC §3.4). */
    created: false,
    ready: false,
    /** The user wants music (play/Start), even while it is still loading. */
    wanted: false,
    playing: false,
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

  function loadCurrent() {
    if (!player || !state.ready) return
    const src = source()
    if (!src) return setError('no_videos')
    playlistErrors = 0
    playlistConfigured = false
    state.playlist = null
    state.title = src.title ?? ''
    if (src.type === SourceType.SourcePlaylist) {
      const opts = { list: src.id, listType: 'playlist' as const }
      if (state.wanted) player.loadPlaylist(opts)
      else player.cuePlaylist(opts)
    } else if (state.wanted) {
      player.loadVideoById(src.id)
    } else {
      player.cueVideoById(src.id)
    }
  }

  /** Move to the next working source of the genre, or report that none is left. */
  function advance(code: number | null = null) {
    const g = genre()
    const next = g ? nextPlayableSource(g, state.sourceIndex, broken) : null
    if (next === null) return setError('no_videos', code)
    state.sourceIndex = next
    loadCurrent()
  }

  function applyVolume() {
    if (!player) return
    player.setVolume(state.duck === null ? state.volume : Math.min(state.duck, state.volume))
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
    if (!player) return
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
        loadCurrent()
      }
    }
  }

  function onError(code: number) {
    if (!player) return
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

  const onOffline = () => {
    if (state.created || state.wanted) setError('offline')
  }
  const onOnline = () => {
    if (state.error === 'offline') retry()
  }
  target.addEventListener('offline', onOffline)
  target.addEventListener('online', onOnline)

  function retry() {
    if (state.error === 'no_videos') return // only another genre helps
    clearError()
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
      state.title = source()?.title ?? ''
      state.volume = cfg.volume
      state.muted = cfg.muted
      state.configured = true
      if (state.wanted) void ensurePlayer()
    },

    /** The element the iframe goes into (PlayerPanel's container). */
    attach(el: HTMLElement) {
      host = el
      if (state.wanted) void ensurePlayer()
    },

    play() {
      state.wanted = true
      if (state.error === 'offline' || state.error === 'config') return retry()
      if (!player) return void ensurePlayer()
      if (state.ready && !state.error) player.playVideo()
    },

    pause() {
      state.wanted = false
      player?.pauseVideo()
    },

    nextTrack() {
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
      state.title = source()?.title ?? ''
      state.playlist = null
      if (player) loadCurrent()
      else if (state.wanted) void ensurePlayer()
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
      if (state.ready) applyVolume()
    },

    setMuted(muted: boolean) {
      state.muted = muted
      if (state.ready) applyVolume()
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
      if (state.ready) applyVolume()
    },

    retry,

    dispose() {
      target.removeEventListener('offline', onOffline)
      target.removeEventListener('online', onOnline)
      player?.destroy()
      player = null
    },
  }
  return api
}

export type MusicPlayer = ReturnType<typeof createMusicPlayer>
