// Dev-only catalog check (TASKS T6): plays every source through the real
// IFrame player on the app's own origin (http://wails.localhost) and reports
// which ones work. oEmbed can't be trusted for this: it answers 200 for dead
// streams (docs/spike-results.md §5).
//
// Opened as /?catalog-check by scripts/catalog-check/run.ps1, which calls
// window.__runCatalogCheck(candidates?) over the DevTools protocol.
import { MusicService } from '../../bindings/focuswidget/services'
import { loadYouTubeApi, PLAYER_HEIGHT, PLAYER_WIDTH } from '../lib/youtube'

export interface CheckSource {
  genre: string
  /** channel_live: `id` is a channel id, played via the live_stream embed. */
  type: 'video' | 'playlist' | 'channel_live'
  id: string
  title?: string
}

export interface CheckResult extends CheckSource {
  ok: boolean
  error?: number | string
  videoTitle?: string
  videoId?: string
  playlistLength?: number
  ms: number
}

const TIMEOUT_MS = 20_000

declare global {
  interface Window {
    __runCatalogCheck?: (candidates?: CheckSource[]) => Promise<CheckResult[]>
    __catalogCheck?: { done: boolean; results: CheckResult[] }
  }
}

/* eslint-disable @typescript-eslint/no-explicit-any -- raw IFrame API objects */
function checkOne(host: HTMLElement, src: CheckSource): Promise<CheckResult> {
  const started = performance.now()
  return new Promise((resolve) => {
    const el = document.createElement('div')
    host.replaceChildren(el)
    let lastTime = -1
    let settled = false
    const timers: {
      poll?: ReturnType<typeof setInterval>
      timeout?: ReturnType<typeof setTimeout>
    } = {}
    const finish = (r: Partial<CheckResult>) => {
      if (settled) return
      settled = true
      clearInterval(timers.poll)
      clearTimeout(timers.timeout)
      const data = player?.getVideoData?.() ?? {}
      const result: CheckResult = {
        ...src,
        ok: false,
        videoTitle: data.title,
        videoId: data.video_id,
        playlistLength: player?.getPlaylist?.()?.length,
        ms: Math.round(performance.now() - started),
        ...r,
      }
      player?.destroy?.()
      resolve(result)
    }

    const vars: Record<string, unknown> = {
      playsinline: 1,
      rel: 0,
      origin: location.origin,
      mute: 1,
    }
    const opts: Record<string, unknown> = {
      width: PLAYER_WIDTH,
      height: PLAYER_HEIGHT,
      playerVars: vars,
      events: {
        onReady: () => {
          player.mute()
          if (src.type === 'playlist') player.loadPlaylist({ list: src.id, listType: 'playlist' })
          else player.playVideo()
        },
        onError: (e: { data: number }) => finish({ error: e.data }),
      },
    }
    if (src.type === 'video') opts.videoId = src.id
    if (src.type === 'channel_live') {
      opts.videoId = 'live_stream'
      vars.channel = src.id
    }
    // The event handlers above only run after the constructor has returned.
    const player: any = new (window.YT as any).Player(el, opts)

    // Playing = state 1 and the playhead actually moving (an ad also counts).
    timers.poll = setInterval(() => {
      if (!player.getPlayerState) return
      const t = player.getCurrentTime()
      if (player.getPlayerState() === 1 && lastTime >= 0 && t > lastTime) finish({ ok: true })
      lastTime = t
    }, 1000)
    timers.timeout = setTimeout(() => finish({ error: 'timeout' }), TIMEOUT_MS)
  })
}
/* eslint-enable @typescript-eslint/no-explicit-any */

export function mountCatalogCheck() {
  document.body.innerHTML = ''
  document.body.style.cssText = 'margin:0;background:#121218;color:#ddd;font:12px monospace'
  const host = document.createElement('div')
  host.style.cssText = `width:${PLAYER_WIDTH}px;height:${PLAYER_HEIGHT}px;margin:8px auto`
  const out = document.createElement('pre')
  out.style.cssText = 'margin:8px;white-space:pre-wrap'
  document.body.append(host, out)

  window.__runCatalogCheck = async (candidates) => {
    await loadYouTubeApi()
    let list = candidates
    if (!list?.length) {
      const catalog = await MusicService.Genres()
      list = (catalog.genres ?? []).flatMap((g) =>
        (g.sources ?? [])
          .filter((s) => s.type !== 'noise') // generated in the page, nothing to check on YouTube
          .map((s) => ({ genre: g.id, type: s.type, id: s.id, title: s.title }) as CheckSource),
      )
    }
    window.__catalogCheck = { done: false, results: [] }
    for (const src of list) {
      const r = await checkOne(host, src)
      window.__catalogCheck.results.push(r)
      out.textContent += `${r.ok ? 'OK  ' : 'FAIL'} ${r.genre} ${r.type} ${r.id} ${r.error ?? ''} ${r.videoTitle ?? ''}\n`
    }
    window.__catalogCheck.done = true
    return window.__catalogCheck.results
  }
}
