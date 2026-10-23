import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Phase, Status, type State } from '../../bindings/focuswidget/internal/timer/models'
import TimerControls from './TimerControls.vue'
import TimerView from './TimerView.vue'

const MIN = 60_000
const base: State = {
  phase: Phase.Focus,
  status: Status.Idle,
  durationMs: 25 * MIN,
  remainingMs: 25 * MIN,
  completedFocus: 0,
  longBreakEvery: 4,
}

describe('TimerControls', () => {
  it.each<[Status, string, 'start' | 'pause' | 'resume']>([
    [Status.Idle, 'Старт', 'start'],
    [Status.Running, 'Пауза', 'pause'],
    [Status.Paused, 'Продолжить', 'resume'],
  ])('main button while %s says %s and emits %s', async (status, label, event) => {
    const wrapper = mount(TimerControls, {
      props: { state: { ...base, status, remainingMs: 20 * MIN } },
    })
    const main = wrapper.get('button.main')
    expect(main.text()).toBe(label)
    await main.trigger('click')
    for (const e of ['start', 'pause', 'resume'] as const) {
      expect(wrapper.emitted(e)?.length ?? 0).toBe(e === event ? 1 : 0)
    }
  })

  it('disables reset for an untouched idle phase but never disables skip', () => {
    const wrapper = mount(TimerControls, { props: { state: base } })
    expect(wrapper.get('button[aria-label="Сбросить фазу"]').attributes('disabled')).toBeDefined()
    expect(
      wrapper.get('button[aria-label="Пропустить фазу"]').attributes('disabled'),
    ).toBeUndefined()
  })

  it('emits skip and resetPhase', async () => {
    const wrapper = mount(TimerControls, { props: { state: { ...base, status: Status.Running } } })
    await wrapper.get('button[aria-label="Сбросить фазу"]').trigger('click')
    await wrapper.get('button[aria-label="Пропустить фазу"]').trigger('click')
    expect(wrapper.emitted('resetPhase')).toHaveLength(1)
    expect(wrapper.emitted('skip')).toHaveLength(1)
  })
})

describe('TimerView', () => {
  it('renders label, time and progress', () => {
    const wrapper = mount(TimerView, {
      props: {
        state: {
          ...base,
          status: Status.Running,
          completedFocus: 1,
          remainingMs: 18 * MIN + 42_000,
        },
      },
    })
    expect(wrapper.get('.pill').text()).toBe('Фокус · сессия 2 из 4')
    expect(wrapper.get('.pill').classes()).toContain('accent')
    expect(wrapper.get('[role="timer"]').text()).toBe('18:42')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('25')
  })

  it('dims the time and shows a neutral pill while paused', () => {
    const wrapper = mount(TimerView, { props: { state: { ...base, status: Status.Paused } } })
    expect(wrapper.get('.timer').classes()).toContain('paused')
    expect(wrapper.get('.pill').classes()).toContain('neutral')
    expect(wrapper.get('.pill').text()).toBe('Пауза · фокус 1 из 4')
  })
})
