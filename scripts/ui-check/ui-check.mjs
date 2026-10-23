// Drives the running Focus Widget over CDP: clicks the real buttons, checks the
// DOM, and takes a screenshot after each step. Usage: node ui-check.mjs <port> <outDir>
import { execFileSync } from 'node:child_process'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const [port = '9333', outDir = '.'] = process.argv.slice(2)
const here = dirname(fileURLToPath(import.meta.url))
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

let pages = []
for (let i = 0; i < 80 && !pages.length; i++) {
  try {
    pages = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).filter((p) => p.type === 'page')
  } catch {}
  if (!pages.length) await sleep(250)
}
if (!pages.length) throw new Error('app page not found')

const ws = new WebSocket(pages[0].webSocketDebuggerUrl)
await new Promise((res, rej) => ((ws.onopen = res), (ws.onerror = rej)))
let seq = 0
const pending = new Map()
ws.onmessage = (m) => {
  const msg = JSON.parse(m.data)
  pending.get(msg.id)?.(msg)
}
const send = (method, params = {}) =>
  new Promise((resolve) => {
    const id = ++seq
    pending.set(id, resolve)
    ws.send(JSON.stringify({ id, method, params }))
  })
const evaluate = async (expression) => {
  const r = await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
  if (r.result?.exceptionDetails) throw new Error(JSON.stringify(r.result.exceptionDetails))
  return r.result?.result?.value
}

for (let i = 0; i < 40 && !(await evaluate('!!window.__fw && !!document.querySelector("button.main")')); i++) {
  await sleep(250)
}

const dom = () =>
  evaluate(`JSON.stringify({
    phase: document.querySelector('.widget')?.dataset.phase,
    pill: document.querySelector('.pill')?.textContent.trim(),
    time: document.querySelector('[role=timer]')?.textContent.trim(),
    progress: document.querySelector('[role=progressbar]')?.getAttribute('aria-valuenow'),
    main: document.querySelector('button.main')?.textContent.trim(),
    resetDisabled: document.querySelector('button[aria-label="Сбросить фазу"]')?.disabled,
    skipDisabled: document.querySelector('button[aria-label="Пропустить фазу"]')?.disabled,
    font: getComputedStyle(document.querySelector('[role=timer]')).fontFamily,
    fontLoaded: document.fonts.check('600 76px "Inter Variable"'),
    glass: getComputedStyle(document.querySelector('.widget')).backgroundColor,
    bodyBg: getComputedStyle(document.body).backgroundColor,
  })`)
const click = (selector) => evaluate(`document.querySelector('${selector}').click()`)
const shot = (name) => {
  execFileSync('powershell', ['-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', join(here, 'shot.ps1'), '-Out', join(outDir, name + '.png')])
  return name
}
const step = async (name) => console.log(name.padEnd(10), await dom(), '→', shot(name))

await evaluate('window.__fw.TimerService.Reset()') // clean idle focus
await sleep(400)
await step('idle')

await click('button.main') // Старт
await sleep(2300)
await step('running')

await click('button.main') // Пауза
await sleep(400)
await step('paused')

await click('button[aria-label="Сбросить фазу"]')
await sleep(400)
await step('reset')

await click('button[aria-label="Пропустить фазу"]') // -> short break, idle
await sleep(400)
await click('button.main') // start the break
await sleep(1300)
await step('break')

await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'light' }] })
await sleep(400)
await step('light')
await send('Emulation.setEmulatedMedia', { features: [] })

await evaluate('window.__fw.TimerService.Reset()')
const events = await evaluate('JSON.stringify(window.__fw.log.filter(e => e.name === "timer:transition").map(e => e.data.cause))')
console.log('transitions:', events)
ws.close()
