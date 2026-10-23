package services

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

// TimerService runs the Pomodoro timer and forwards its state and transitions
// to the frontend as timer:state and timer:transition events.
type TimerService struct {
	timer  *timer.Timer
	cancel context.CancelFunc
}

func NewTimerService(t *timer.Timer, emit Emitter) *TimerService {
	t.OnTransition(func(tr timer.Transition) { emit.Emit(EventTimerTransition, tr) })
	t.OnState(func(st timer.State) { emit.Emit(EventTimerState, st) })
	return &TimerService{timer: t}
}

func (s *TimerService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	ctx, s.cancel = context.WithCancel(ctx)
	go s.timer.Run(ctx)
	return nil
}

func (s *TimerService) ServiceShutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// State returns the current snapshot, for the frontend's initial render.
func (s *TimerService) State() timer.State { return s.timer.State() }

// Start starts the current idle phase; it fails with a "not allowed" error
// unless the timer is idle.
func (s *TimerService) Start() error { return s.timer.Start() }

// Pause pauses the running phase; it fails unless the timer is running.
func (s *TimerService) Pause() error { return s.timer.Pause() }

// Resume continues a paused phase; it fails unless the timer is paused.
func (s *TimerService) Resume() error { return s.timer.Resume() }

// Skip ends the current phase without counting it and moves to the next one.
func (s *TimerService) Skip() { s.timer.Skip() }

// ResetPhase puts the current phase back to its full length, idle; the
// session count is kept («Сбросить фазу»).
func (s *TimerService) ResetPhase() { s.timer.ResetPhase() }

// Reset returns to an idle focus phase and starts a new cycle («Сбросить цикл»).
func (s *TimerService) Reset() { s.timer.Reset() }

// TimerConfig converts the persisted timer settings into the timer's config.
func TimerConfig(t config.TimerSettings) timer.Config {
	return timer.Config{
		Focus:           time.Duration(t.FocusMin) * time.Minute,
		ShortBreak:      time.Duration(t.ShortBreakMin) * time.Minute,
		LongBreak:       time.Duration(t.LongBreakMin) * time.Minute,
		LongBreakEvery:  t.LongBreakEvery,
		AutoStartBreaks: t.AutoStartBreaks,
		AutoStartFocus:  t.AutoStartFocus,
	}
}
