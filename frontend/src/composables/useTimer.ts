import { Events } from '@wailsio/runtime'
import { onMounted, onUnmounted, shallowRef } from 'vue'
import { TimerState } from '../../bindings/events'
import { TimerService } from '../../bindings/focuswidget/services'
import type { State } from '../../bindings/focuswidget/internal/timer/models'

/**
 * Live timer state from Go plus its commands. A command the timer rejects
 * (e.g. a double-click on Start) is logged, not thrown: the next timer:state
 * event brings the UI back in line.
 */
export function useTimer() {
  const state = shallowRef<State | null>(null)
  let off: (() => void) | undefined

  onMounted(async () => {
    // Subscribe before fetching so no change between the two is lost; an event
    // that arrives first is newer than the fetched snapshot.
    off = Events.On(TimerState, (ev) => {
      state.value = ev.data
    })
    const initial = await TimerService.State()
    state.value ??= initial
  })
  onUnmounted(() => off?.())

  const run = (command: () => Promise<void>) => () =>
    command().catch((err: unknown) => console.warn('timer command rejected:', err))

  return {
    state,
    start: run(TimerService.Start),
    pause: run(TimerService.Pause),
    resume: run(TimerService.Resume),
    skip: run(TimerService.Skip),
    resetPhase: run(TimerService.ResetPhase),
  }
}
