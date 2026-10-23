package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

type fakeSender struct {
	mu   sync.Mutex
	sent []notifications.NotificationOptions
	err  error
}

func (s *fakeSender) SendNotification(o notifications.NotificationOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, o)
	return s.err
}

func (s *fakeSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sent)
}

func tr(fromPhase timer.Phase, to timer.State, cause timer.Cause) timer.Transition {
	return timer.Transition{From: timer.State{Phase: fromPhase, Status: timer.Running}, To: to, Cause: cause}
}

func TestPhaseEndMessage(t *testing.T) {
	const min = 60_000
	tests := []struct {
		name        string
		tr          timer.Transition
		title, body string
	}{
		{"focus -> short break, auto-started",
			tr(timer.Focus, timer.State{Phase: timer.ShortBreak, Status: timer.Running, DurationMs: 5 * min}, timer.CauseExpire),
			"Фокус завершён", "Начался короткий перерыв, 5 мин"},
		{"focus -> long break, waiting",
			tr(timer.Focus, timer.State{Phase: timer.LongBreak, Status: timer.Idle, DurationMs: 15 * min}, timer.CauseExpire),
			"Фокус завершён", "Дальше длинный перерыв, 15 мин. Нажмите «Старт», когда будете готовы"},
		{"break -> focus, waiting",
			tr(timer.ShortBreak, timer.State{Phase: timer.Focus, Status: timer.Idle, DurationMs: 25 * min, CompletedFocus: 2, LongBreakEvery: 4}, timer.CauseExpire),
			"Перерыв окончен", "Дальше фокус, сессия 3 из 4. Нажмите «Старт», когда будете готовы"},
		{"long break -> focus, auto-started",
			tr(timer.LongBreak, timer.State{Phase: timer.Focus, Status: timer.Running, DurationMs: 25 * min, LongBreakEvery: 4}, timer.CauseExpire),
			"Перерыв окончен", "Начался фокус, сессия 1 из 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, body, ok := PhaseEndMessage(tt.tr)
			if !ok || title != tt.title || body != tt.body {
				t.Fatalf("got %q / %q / %v", title, body, ok)
			}
		})
	}

	for _, cause := range []timer.Cause{timer.CauseStart, timer.CausePause, timer.CauseResume, timer.CauseSkip, timer.CauseResetPhase, timer.CauseReset} {
		if _, _, ok := PhaseEndMessage(tr(timer.Focus, timer.State{Phase: timer.ShortBreak}, cause)); ok {
			t.Errorf("%s produced a toast", cause)
		}
	}
}

// stepClock drives a real timer (its Run loop) with manual time steps; each
// advance fires one tick.
type stepClock struct {
	mu   sync.Mutex
	now  time.Time
	tick chan time.Time
}

func newStepClock() *stepClock {
	return &stepClock{now: time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC), tick: make(chan time.Time)}
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *stepClock) NewTicker(time.Duration) timer.Ticker { return stepTicker{c.tick} }

// advance moves time on and waits until the timer's loop has taken the tick.
func (c *stepClock) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	c.mu.Unlock()
	c.tick <- now
}

type stepTicker struct{ c chan time.Time }

func (t stepTicker) C() <-chan time.Time { return t.c }
func (t stepTicker) Stop()               {}

func runTimer(t *testing.T, clock *stepClock) *timer.Timer {
	t.Helper()
	tm, err := timer.New(TimerConfig(config.Default().Timer), clock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go tm.Run(ctx)
	return tm
}

func waitCount(s *fakeSender, want int) int {
	for i := 0; i < 100 && s.count() < want; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond) // and nothing more arrives
	return s.count()
}

func TestPhaseNotifierOneToastPerPhaseEnd(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	clock := newStepClock()
	tm := runTimer(t, clock)
	sender := &fakeSender{}
	NewPhaseNotifier(tm, sender, settings, nil)

	_ = tm.Start()
	tm.Skip()
	if got := waitCount(sender, 0); got != 0 {
		t.Fatalf("start/skip sent %d toasts", got)
	}

	_ = tm.Start()               // short break
	clock.advance(3 * time.Hour) // the PC slept through several phases
	clock.advance(time.Second)
	if got := waitCount(sender, 1); got != 1 {
		t.Fatalf("sent %d toasts after sleep, want exactly 1", got)
	}
	if o := sender.sent[0]; o.ID != PhaseEndToastID || o.Title != "Перерыв окончен" {
		t.Fatalf("toast = %+v", o)
	}
}

func TestPhaseNotifierRespectsSettingAndReportsErrors(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	clock := newStepClock()
	tm := runTimer(t, clock)
	sender := &fakeSender{err: errors.New("toast failed")}
	var errs []error
	var mu sync.Mutex
	NewPhaseNotifier(tm, sender, settings, func(err error) { mu.Lock(); errs = append(errs, err); mu.Unlock() })

	off := settings.Get()
	off.Notify.Toast = false
	if _, err := settings.Update(off); err != nil {
		t.Fatal(err)
	}
	_ = tm.Start()
	clock.advance(26 * time.Minute)
	if got := waitCount(sender, 0); got != 0 {
		t.Fatalf("toast sent with toasts switched off")
	}

	on := settings.Get()
	on.Notify.Toast = true
	_, _ = settings.Update(on)
	_ = tm.Start()
	clock.advance(6 * time.Minute)
	waitCount(sender, 1)
	mu.Lock()
	defer mu.Unlock()
	if len(errs) != 1 {
		t.Fatalf("errors reported = %v", errs)
	}
}
