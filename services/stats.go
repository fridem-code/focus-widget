package services

import (
	"time"

	"focuswidget/internal/stats"
	"focuswidget/internal/timer"
)

// StatsService records every focus phase that runs to its end in the session
// log and sums the log up for the statistics screen (T23). After each new
// session it sends the fresh summary as a stats:changed event.
type StatsService struct {
	log      *stats.Log
	settings *SettingsService
	emit     Emitter
	now      func() time.Time
}

// NewStatsService subscribes to the timer. A skipped focus is not recorded:
// only time actually spent counts. onError reports a session the file did
// not take (it still counts until the app quits).
func NewStatsService(t *timer.Timer, log *stats.Log, settings *SettingsService, emit Emitter,
	now func() time.Time, onError func(error),
) *StatsService {
	s := &StatsService{log: log, settings: settings, emit: emit, now: now}
	t.OnTransition(func(tr timer.Transition) {
		if tr.Cause != timer.CauseExpire || tr.From.Phase != timer.Focus {
			return
		}
		session := stats.Session{
			End:     s.now(),
			Minutes: int((tr.From.DurationMs + 30_000) / 60_000),
			Label:   settings.Get().Timer.Label,
		}
		// In the listener, not on a goroutine of its own: the toast for this
		// phase end counts the session (PhaseNotifier subscribes later).
		if err := log.Add(session); err != nil && onError != nil {
			onError(err)
		}
		s.emit.Emit(EventStatsChanged, s.Summary())
	})
	return s
}

// Summary sums the log up as of now.
func (s *StatsService) Summary() stats.Summary {
	return stats.Summarize(s.log.Sessions(), s.now())
}

// FocusToday counts today's sessions, for the toast and the tray.
func (s *StatsService) FocusToday() int {
	return s.Summary().Today.Focus
}
