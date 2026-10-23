import { describe, expect, it, vi } from 'vitest'
import { createWindowMode, type Mode } from './windowMode'

function setup(initial: Mode = 'expanded') {
  const log: string[] = []
  let emit!: (m: Mode) => void
  const setMode = vi.fn(async (m: Mode) => {
    log.push(`resize ${m}`)
  })
  const wm = createWindowMode({
    initial,
    setMode,
    onModeEvent: (cb) => {
      emit = cb
      return () => log.push('unsubscribed')
    },
    beforeCompact: () => log.push('pause music'),
    afterExpand: () => log.push('maybe resume music'),
  })
  return { wm, log, setMode, emit: (m: Mode) => emit(m) }
}

describe('createWindowMode', () => {
  it('pauses before shrinking and resumes after growing', async () => {
    const { wm, log } = setup()
    await wm.compact()
    expect(wm.mode.value).toBe('compact')
    await wm.expand()
    expect(wm.mode.value).toBe('expanded')
    expect(log).toEqual(['pause music', 'resize compact', 'resize expanded', 'maybe resume music'])
  })

  it('ignores switching to the current mode', async () => {
    const { wm, setMode, log } = setup('compact')
    await wm.compact()
    expect(setMode).not.toHaveBeenCalled()
    expect(log).toEqual([])
  })

  it('follows window:mode from Go (the tray), once per change', () => {
    const { wm, emit, log } = setup()
    emit('compact')
    emit('compact')
    expect(wm.mode.value).toBe('compact')
    emit('expanded')
    expect(log).toEqual(['pause music', 'maybe resume music'])
  })

  it('the echo of our own switch does not run the hooks twice', async () => {
    const { wm, emit, log } = setup()
    const p = wm.compact()
    emit('compact') // Go announces the switch before the call returns
    await p
    expect(log).toEqual(['pause music', 'resize compact'])
  })

  it('goes back to expanded if the resize fails', async () => {
    const { wm, setMode, log } = setup()
    setMode.mockRejectedValueOnce(new Error('boom'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    await wm.compact()
    expect(wm.mode.value).toBe('expanded')
    expect(log).toEqual(['pause music', 'maybe resume music'])
    warn.mockRestore()
  })

  it('stays compact if growing fails', async () => {
    const { wm, setMode } = setup('compact')
    setMode.mockRejectedValueOnce(new Error('boom'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    await wm.expand()
    expect(wm.mode.value).toBe('compact')
    warn.mockRestore()
  })

  it('stop unsubscribes', () => {
    const { wm, log } = setup()
    wm.stop()
    expect(log).toEqual(['unsubscribed'])
  })
})
