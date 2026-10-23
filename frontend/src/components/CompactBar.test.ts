import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Phase, Status, type State } from '../../bindings/focuswidget/internal/timer/models'
import CompactBar from './CompactBar.vue'

const MIN = 60_000
const state = (over: Partial<State> = {}): State => ({
  phase: Phase.Focus,
  status: Status.Running,
  durationMs: 25 * MIN,
  remainingMs: 18 * MIN + 42_000,
  completedFocus: 1,
  longBreakEvery: 4,
  ...over,
})

describe('CompactBar (design artboard "Компактный режим")', () => {
  it('focus running: label, time, music-held icon, pause button, progress', () => {
    const w = mount(CompactBar, { props: { state: state() } })
    expect(w.get('.label').text()).toBe('Фокус · 2/4')
    expect(w.get('[role=timer]').text()).toBe('18:42')
    expect(w.find('[aria-label="Музыка на паузе"]').exists()).toBe(true)
    expect(w.get('button.main').attributes('aria-label')).toBe('Пауза')
    expect(w.get('[role=progressbar]').attributes('aria-valuenow')).toBe('25')
  })

  it('paused: dimmed, resume button, no music icon', () => {
    const w = mount(CompactBar, { props: { state: state({ status: Status.Paused }) } })
    expect(w.get('.bar').classes()).toContain('paused')
    expect(w.get('.label').text()).toBe('Пауза · фокус 2/4')
    expect(w.find('[aria-label="Музыка на паузе"]').exists()).toBe(false)
    expect(w.get('button.main').attributes('aria-label')).toBe('Продолжить')
  })

  it('short break shows its length', () => {
    const w = mount(CompactBar, {
      props: {
        state: state({ phase: Phase.ShortBreak, durationMs: 5 * MIN, remainingMs: 197_000 }),
      },
    })
    expect(w.get('.label').text()).toBe('Перерыв · 5 мин')
    expect(w.get('[role=timer]').text()).toBe('03:17')
  })

  it.each([
    [Status.Idle, 'start'],
    [Status.Running, 'pause'],
    [Status.Paused, 'resume'],
  ] as const)('main button while %s emits %s', async (status, event) => {
    const w = mount(CompactBar, { props: { state: state({ status }) } })
    await w.get('button.main').trigger('click')
    expect(w.emitted(event)).toHaveLength(1)
  })

  it('skip and expand', async () => {
    const w = mount(CompactBar, { props: { state: state() } })
    await w.get('button[aria-label="Пропустить фазу"]').trigger('click')
    await w.get('button[aria-label="Развернуть"]').trigger('click')
    expect(w.emitted('skip')).toHaveLength(1)
    expect(w.emitted('expand')).toHaveLength(1)
  })
})
