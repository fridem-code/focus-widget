import { describe, expect, it } from 'vitest'
import {
  SourceType,
  type Catalog,
  type Genre,
} from '../../bindings/focuswidget/internal/catalog/models'
import {
  clampSource,
  classifyError,
  genreLabel,
  nextGenre,
  nextPlayableSource,
  position,
  resolveGenre,
} from './music'

const video = (id: string) => ({ type: SourceType.SourceVideo, id })
const genre: Genre = {
  id: 'noise',
  name: 'Белый шум',
  sources: [video('aaaaaaaaaaa'), video('bbbbbbbbbbb'), video('ccccccccccc')],
}
const mixed: Genre = {
  id: 'lofi',
  name: 'Lo-fi',
  sources: [{ type: SourceType.SourcePlaylist, id: 'PLxxxxxxxxxx' }, video('ddddddddddd')],
}
const catalog: Catalog = { genres: [mixed, genre] }

describe('classifyError', () => {
  it.each([2, 5, 100, 101, 150, 999])('%i skips the source', (code) => {
    expect(classifyError(code)).toBe('skip')
  })
  it('153 is a configuration error', () => {
    expect(classifyError(153)).toBe('config')
  })
})

describe('genres', () => {
  it('resolves by id and falls back to the first genre', () => {
    expect(resolveGenre(catalog, 'noise')).toBe(genre)
    expect(resolveGenre(catalog, 'gone')).toBe(mixed)
    expect(resolveGenre({ genres: null }, 'x')).toBeNull()
  })

  it('next genre wraps around', () => {
    expect(nextGenre(catalog, 'lofi')).toBe(genre)
    expect(nextGenre(catalog, 'noise')).toBe(mixed)
    expect(nextGenre(catalog, 'gone')).toBe(mixed)
  })
})

describe('nextPlayableSource', () => {
  it('moves forward and wraps', () => {
    expect(nextPlayableSource(genre, 0, new Set())).toBe(1)
    expect(nextPlayableSource(genre, 2, new Set())).toBe(0)
  })
  it('skips broken sources', () => {
    expect(nextPlayableSource(genre, 0, new Set([1]))).toBe(2)
  })
  it('returns the current one last', () => {
    expect(nextPlayableSource(genre, 0, new Set([1, 2]))).toBe(0)
  })
  it('is null when everything is broken', () => {
    expect(nextPlayableSource(genre, 0, new Set([0, 1, 2]))).toBeNull()
  })
})

describe('position and label', () => {
  it('uses the playlist position for playlist sources', () => {
    expect(position(mixed, 0, null)).toBeNull()
    expect(position(mixed, 0, { index: 2, total: 12 })).toEqual({ index: 2, total: 12 })
  })
  it('uses the source position for video sources', () => {
    expect(position(genre, 1, null)).toEqual({ index: 1, total: 3 })
  })
  it('formats the genre label', () => {
    expect(genreLabel('Lo-fi', { index: 2, total: 12 })).toBe('Lo-fi · 3 / 12')
    expect(genreLabel('Lo-fi', null)).toBe('Lo-fi')
  })
  it('clamps a stale source index', () => {
    expect(clampSource(genre, 2)).toBe(2)
    expect(clampSource(genre, 7)).toBe(0)
    expect(clampSource(genre, -1)).toBe(0)
  })
})
