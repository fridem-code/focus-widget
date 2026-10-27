package services

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"focuswidget/internal/stats"
	"focuswidget/internal/timer"
)

func TestPhaseEndMessageLabelAndGoal(t *testing.T) {
	const min = 60_000
	toBreak := tr(timer.Focus, timer.State{Phase: timer.ShortBreak, Status: timer.Running, DurationMs: 5 * min}, timer.CauseExpire)
	toFocus := tr(timer.ShortBreak, timer.State{Phase: timer.Focus, Status: timer.Idle, DurationMs: 25 * min, LongBreakEvery: 4}, timer.CauseExpire)
	tests := []struct {
		name        string
		tr          timer.Transition
		day         PhaseDay
		title, body string
	}{
		{"label", toBreak, PhaseDay{Label: "Отчёт", Done: 3, Goal: 8},
			"Фокус завершён · Отчёт", "Начался короткий перерыв, 5 мин"},
		{"goal reached", toBreak, PhaseDay{Label: "Отчёт", Done: 8, Goal: 8},
			"Цель на день выполнена", "8 из 8 помидоров за сегодня. Начался короткий перерыв, 5 мин"},
		{"goal of one", toBreak, PhaseDay{Done: 1, Goal: 1},
			"Цель на день выполнена", "1 из 1 помидора за сегодня. Начался короткий перерыв, 5 мин"},
		{"past the goal", toBreak, PhaseDay{Done: 9, Goal: 8},
			"Фокус завершён", "Начался короткий перерыв, 5 мин"},
		{"no goal", toBreak, PhaseDay{Done: 0, Goal: 0},
			"Фокус завершён", "Начался короткий перерыв, 5 мин"},
		{"a break ends: no label, no goal", toFocus, PhaseDay{Label: "Отчёт", Done: 8, Goal: 8},
			"Перерыв окончен", "Дальше фокус, сессия 1 из 4. Нажмите «Старт», когда будете готовы"},
	}
	for _, tt := range tests {
		title, body, ok := PhaseEndMessage(tt.tr, tt.day)
		if !ok || title != tt.title || body != tt.body {
			t.Errorf("%s: got %q / %q, want %q / %q", tt.name, title, body, tt.title, tt.body)
		}
	}
}

// statsFixture runs a real timer on a step clock with the session log, the
// stats service and the phase notifier wired as the app wires them.
type statsFixture struct {
	clock    *stepClock
	timer    *timer.Timer
	settings *SettingsService
	stats    *StatsService
	emit     *fakeEmitter
	sender   *fakeSender
	dir      string
}

func newStatsFixture(t *testing.T) *statsFixture {
	settings, _, _, _ := newSettingsService(t)
	f := &statsFixture{clock: newStepClock(), settings: settings, emit: &fakeEmitter{}, sender: &fakeSender{}, dir: t.TempDir()}
	f.timer = runTimer(t, f.clock)
	log := stats.Open(f.dir, slog.New(slog.DiscardHandler))
	f.stats = NewStatsService(f.timer, log, settings, f.emit, f.clock.Now, func(err error) { t.Error(err) })
	NewPhaseNotifier(f.timer, f.sender, settings, f.stats.FocusToday, nil)
	return f
}

// focus runs one focus phase of the default 25 minutes to its end; the break
// that follows starts by itself.
func (f *statsFixture) focus(t *testing.T) {
	t.Helper()
	before := f.stats.FocusToday()
	if err := f.timer.Start(); err != nil {
		t.Fatal(err)
	}
	f.clock.advance(25*time.Minute + time.Second)
	f.until(t, func() bool { return f.stats.FocusToday() == before+1 })
}

// until waits for the timer's listeners: advance returns once the tick is
// taken, not handled.
func (f *statsFixture) until(t *testing.T, done func() bool) {
	t.Helper()
	for i := 0; i < 200 && !done(); i++ {
		time.Sleep(5 * time.Millisecond)
	}
	if !done() {
		t.Fatal("timed out")
	}
}

func TestStatsRecordsFinishedFocusWithLabel(t *testing.T) {
	f := newStatsFixture(t)
	label := "Отчёт"
	if _, err := f.settings.Patch(SettingsPatch{Label: &label}); err != nil {
		t.Fatal(err)
	}

	_ = f.timer.Start()
	f.timer.Skip() // skipped focus: not recorded
	f.timer.Skip() // back to focus
	f.focus(t)

	sum := f.stats.Summary()
	if sum.Today.Focus != 1 || sum.Today.Minutes != 25 || len(sum.Sessions) != 1 || sum.Sessions[0].Label != "Отчёт" {
		t.Fatalf("summary = %+v", sum)
	}
	// Written to the file, so a restart keeps it.
	if got := stats.Open(f.dir, slog.New(slog.DiscardHandler)).Sessions(); len(got) != 1 || got[0].Minutes != 25 {
		t.Fatalf("log file = %+v", got)
	}
	var changed []stats.Summary
	for _, ev := range f.emit.take() {
		if ev.name == EventStatsChanged {
			changed = append(changed, ev.data.(stats.Summary))
		}
	}
	if len(changed) != 1 || changed[0].Today.Focus != 1 {
		t.Fatalf("stats:changed = %+v", changed)
	}
}

func TestGoalToastCountsTheSessionThatJustEnded(t *testing.T) {
	f := newStatsFixture(t)
	goal := 2
	if _, err := f.settings.Patch(SettingsPatch{DailyGoal: &goal}); err != nil {
		t.Fatal(err)
	}

	f.focus(t)
	f.clock.advance(5*time.Minute + time.Second) // the break ends; focus waits for «Старт»
	f.until(t, func() bool { return f.timer.State().Phase == timer.Focus })
	f.focus(t)
	waitCount(f.sender, 3)

	var titles []string
	for _, o := range f.sender.sent {
		titles = append(titles, o.Title)
	}
	want := "Фокус завершён|Перерыв окончен|Цель на день выполнена"
	if strings.Join(titles, "|") != want {
		t.Fatalf("toasts = %q, want %q", titles, want)
	}
	if body := f.sender.sent[2].Body; !strings.HasPrefix(body, "2 из 2 помидоров за сегодня.") {
		t.Fatalf("goal toast body = %q", body)
	}
}
