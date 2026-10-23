// Prints window.__state and the player's state from the widget page over the
// DevTools protocol. Usage: node state.mjs <port>
const port = process.argv[2] ?? '9334'
const pages = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).filter(
  (p) => p.type === 'page' && p.url.includes('widget.html'),
)
if (!pages.length) throw new Error('widget page not found')
const ws = new WebSocket(pages[0].webSocketDebuggerUrl)
await new Promise((resolve, reject) => {
  ws.onopen = resolve
  ws.onerror = reject
})
const result = await new Promise((resolve) => {
  ws.onmessage = (m) => resolve(JSON.parse(m.data))
  ws.send(
    JSON.stringify({
      id: 1,
      method: 'Runtime.evaluate',
      params: {
        returnByValue: true,
        expression: `JSON.stringify({ ...window.__state,
          playerState: window.__player?.getPlayerState?.(),
          currentTime: Math.round(window.__player?.getCurrentTime?.() ?? 0),
          muted: window.__player?.isMuted?.() })`,
      },
    }),
  )
})
console.log('page state:', result.result?.result?.value)
ws.close()
