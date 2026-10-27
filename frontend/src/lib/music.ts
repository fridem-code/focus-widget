// Pure music rules (docs/SPEC.md §3.2, §3.7): which source plays next, how a
// YouTube error is handled and what the music row shows.
import {
  SourceType,
  type Catalog,
  type Genre,
  type Source,
} from '../../bindings/focuswidget/internal/catalog/models'

/** `no_files`: "Мои файлы" has nothing it can play (empty or gone folder, T22). */
export type PlayerErrorKind = 'no_videos' | 'no_files' | 'offline' | 'config'

/** The genre of the user's music folder (Go: services.LocalGenreID). */
export const LOCAL_GENRE_ID = 'local'

export interface PlaylistPosition {
  index: number // 0-based
  total: number
}

/**
 * YouTube error codes: 153 means the embed itself is misconfigured (origin /
 * Referer) and no other source will fare better; everything else (2, 5, 100,
 * 101, 150 and unknown codes) means "this video can't play here", so skip it.
 */
export function classifyError(code: number): 'skip' | 'config' {
  return code === 153 ? 'config' : 'skip'
}

export function sources(genre: Genre): Source[] {
  return genre.sources ?? []
}

/** The genre with this id, else the first one; null for an empty catalog. */
export function resolveGenre(catalog: Catalog, id: string): Genre | null {
  const genres = catalog.genres ?? []
  return genres.find((g) => g.id === id) ?? genres[0] ?? null
}

/** The genre after this one, wrapping around. */
export function nextGenre(catalog: Catalog, id: string): Genre | null {
  const genres = catalog.genres ?? []
  if (!genres.length) return null
  const i = genres.findIndex((g) => g.id === id)
  return genres[(i + 1) % genres.length] ?? null
}

/**
 * The next source after `from` that is not known to be broken, wrapping
 * around and ending with `from` itself; null when every source is broken.
 */
export function nextPlayableSource(
  genre: Genre,
  from: number,
  broken: ReadonlySet<number>,
): number | null {
  const n = sources(genre).length
  for (let step = 1; step <= n; step++) {
    const i = (from + step) % n
    if (!broken.has(i)) return i
  }
  return null
}

/**
 * A random source other than `from` that is not known to be broken; `from`
 * itself when it is the only one left, null when every source is broken.
 */
export function randomPlayableSource(
  genre: Genre,
  from: number,
  broken: ReadonlySet<number>,
  random: () => number = Math.random,
): number | null {
  const n = sources(genre).length
  const others = [...Array(n).keys()].filter((i) => i !== from && !broken.has(i))
  if (others.length) return others[Math.floor(random() * others.length)]
  return n > 0 && from < n && !broken.has(from) ? from : null
}

/** Clamp a saved source index to the genre (the catalog may have changed). */
export function clampSource(genre: Genre, index: number): number {
  const n = sources(genre).length
  return index >= 0 && index < n ? index : 0
}

/**
 * Position for "Жанр · i / n": the video within a loaded playlist, or the
 * source within the genre for video sources; null while a playlist is loading.
 */
export function position(
  genre: Genre,
  sourceIndex: number,
  playlist: PlaylistPosition | null,
): PlaylistPosition | null {
  const src = sources(genre)[sourceIndex]
  if (!src) return null
  if (src.type === SourceType.SourcePlaylist) return playlist
  return { index: sourceIndex, total: sources(genre).length }
}

export function genreLabel(name: string, pos: PlaylistPosition | null): string {
  return pos ? `${name} · ${pos.index + 1} / ${pos.total}` : name
}
