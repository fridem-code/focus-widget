import { ref } from 'vue'

export type Mode = 'expanded' | 'compact'

export interface WindowModeDeps {
  initial: Mode
  /** Resizes the window in Go; emits window:mode when done. */
  setMode(mode: Mode): Promise<void>
  /** window:mode from Go (our own switches and, from T10, the tray's). */
  onModeEvent(cb: (mode: Mode) => void): () => void
  /** Runs before the window shrinks: the policy pauses the music. */
  beforeCompact(): void
  /** Runs once the window has grown back: the policy may resume the music. */
  afterExpand(): void
}

/**
 * Switches between expanded and compact mode in an order that never leaves
 * the player playing while hidden (SPEC §3.4): pause, then shrink; grow, then
 * resume.
 */
export function createWindowMode(deps: WindowModeDeps) {
  const mode = ref<Mode>(deps.initial)

  const enterCompact = () => {
    if (mode.value === 'compact') return
    deps.beforeCompact()
    mode.value = 'compact'
  }
  const enterExpanded = () => {
    if (mode.value === 'expanded') return
    mode.value = 'expanded'
    deps.afterExpand()
  }

  const off = deps.onModeEvent((m) => (m === 'compact' ? enterCompact() : enterExpanded()))

  return {
    mode,
    async compact() {
      if (mode.value === 'compact') return
      enterCompact()
      try {
        await deps.setMode('compact')
      } catch (err) {
        console.warn('switching to compact mode failed:', err)
        enterExpanded()
      }
    },
    async expand() {
      if (mode.value === 'expanded') return
      try {
        await deps.setMode('expanded')
      } catch (err) {
        console.warn('switching to expanded mode failed:', err)
        return
      }
      enterExpanded()
    },
    stop: off,
  }
}

export type WindowModeController = ReturnType<typeof createWindowMode>
