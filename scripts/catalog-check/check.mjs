// Opens the dev-only /?catalog-check page in the running app over the DevTools
// protocol and plays each source through the real IFrame player.
// Usage: node check.mjs <port> [candidates.json]
//   candidates.json: [{ "genre": "lofi", "type": "video|playlist|channel_live", "id": "...", "title": "..." }]
//   without it the app's own catalog is checked.
import { readFileSync } from 'node:fs'

const [port = '9333', candidatesFile] = process.argv.slice(2)
const candidates = candidatesFile ? JSON.parse(readFileSync(candidatesFile, 'utf8')) : null
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

let page
for (let i = 0; i < 80 && !page; i++) {
  try {
    page = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).find((p) => p.type === 'page')
  } catch {}
  if (!page) await sleep(250)
}
if (!page) throw new Error('app page not found')

const ws = new WebSocket(page.webSocketDebuggerUrl)
await new Promise((res, rej) => ((ws.onopen = res), (ws.onerror = rej)))
let seq = 0
const pending = new Map()
ws.onmessage = (m) => {
  const msg = JSON.parse(m.data)
  if (msg.method === 'Runtime.exceptionThrown') {
    console.error('page error:', msg.params.exceptionDetails?.exception?.description ?? msg.params.exceptionDetails?.text)
  }
  pending.get(msg.id)?.(msg)
}
ws.onclose = () => {
  console.error('DevTools connection closed (did the app exit?)')
  process.exit(1)
}
const send = (method, params = {}) =>
  new Promise((resolve) => {
    const id = ++seq
    pending.set(id, resolve)
    ws.send(JSON.stringify({ id, method, params }))
  })
const evaluate = async (expression) =>
  (await send('Runtime.evaluate', { expression, returnByValue: true })).result?.result?.value

const origin = new URL(page.url).origin
await send('Runtime.enable')
await send('Page.navigate', { url: `${origin}/?catalog-check` })
for (let i = 0; i < 80 && !(await evaluate('typeof window.__runCatalogCheck === "function"')); i++) {
  await sleep(250)
}
await evaluate(`window.__runCatalogCheck(${JSON.stringify(candidates)})`) // runs in the background

let printed = 0
for (;;) {
  const state = await evaluate('JSON.stringify(window.__catalogCheck ?? null)')
  const check = state ? JSON.parse(state) : null
  for (const r of check?.results.slice(printed) ?? []) {
    const extra = [r.error, r.playlistLength && `${r.playlistLength} videos`, r.videoId, r.videoTitle]
      .filter(Boolean)
      .join(' · ')
    console.log(`${r.ok ? 'OK  ' : 'FAIL'}  ${r.genre.padEnd(9)} ${r.type.padEnd(12)} ${r.id.padEnd(36)} ${extra}`)
    printed++
  }
  if (check?.done) break
  await sleep(1000)
}
await send('Page.navigate', { url: `${origin}/` })
ws.onclose = null
ws.close()
