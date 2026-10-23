import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { Phase, Status, type State } from '../../bindings/focuswidget/internal/timer/models'
import { useTimer } from './useTimer'

// vi.mock factories are hoisted above imports, so their state is too.
const { listeners, off, service } = vi.hoisted(() => ({
  listeners: new Map<string, (ev: { data: unknown }) => void>(),
  off: vi.fn(),
  service: {
    State: vi.fn<() => Promise<State>>(),
    Start: vi.fn<() => Promise<void>>(),
    Pause: vi.fn<() => Promise<void>>(),
    Resume: vi.fn<() => Promise<void>>(),
    Skip: vi.fn<() => Promise<void>>(),
    ResetPhase: vi.fn<() => Promise<void>>(),
  },
}))

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: (name: string, cb: (ev: { data: unknown }) => void) => {
      listeners.set(name, cb)
      return off
    },
  },
}))
vi.mock('../../bindings/focuswidget/services', () => ({ TimerService: service }))
vi.mock('../../bindings/events', () => ({ TimerState: 'timer:state' }))

const idle: State = {
  phase: Phase.Focus,
  status: Status.Idle,
  durationMs: 1_500_000,
  remainingMs: 1_500_000,
  completedFocus: 0,
  longBreakEvery: 4,
}

function mountTimer() {
  let api!: ReturnType<typeof useTimer>
  const wrapper = mount(
    defineComponent({
      setup() {
        api = useTimer()
        return () => h('div')
      },
    }),
  )
  return { wrapper, api }
}

const flush = () => new Promise((r) => setTimeout(r))

beforeEach(() => {
  listeners.clear()
  vi.clearAllMocks()
})

describe('useTimer', () => {
  it('loads the initial state and follows timer:state events', async () => {
    service.State.mockResolvedValue(idle)
    const { api } = mountTimer()
    await flush()
    expect(api.state.value).toEqual(idle)

    const running = { ...idle, status: Status.Running, remainingMs: 1_499_000 }
    listeners.get('timer:state')!({ data: running })
    expect(api.state.value).toEqual(running)
  })

  it('keeps an event that arrives before the initial fetch resolves', async () => {
    let resolve!: (s: State) => void
    service.State.mockReturnValue(new Promise((r) => (resolve = r)))
    const { api } = mountTimer()
    await flush()

    const newer = { ...idle, status: Status.Running }
    listeners.get('timer:state')!({ data: newer })
    resolve(idle) // stale snapshot
    await flush()
    expect(api.state.value).toEqual(newer)
  })

  it('logs rejected commands instead of throwing', async () => {
    service.State.mockResolvedValue(idle)
    service.Start.mockRejectedValue(new Error('start while running'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { api } = mountTimer()
    await expect(api.start()).resolves.toBeUndefined()
    expect(warn).toHaveBeenCalledOnce()
    warn.mockRestore()
  })

  it('unsubscribes on unmount', async () => {
    service.State.mockResolvedValue(idle)
    const { wrapper } = mountTimer()
    await flush()
    wrapper.unmount()
    expect(off).toHaveBeenCalledOnce()
  })
})
