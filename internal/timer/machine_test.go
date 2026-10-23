package timer

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

func testConfig() Config {
	return Config{
		Focus:           25 * time.Minute,
		ShortBreak:      5 * time.Minute,
		LongBreak:       15 * time.Minute,
		LongBreakEvery:  4,
		AutoStartBreaks: true,
		AutoStartFocus:  false,
	}
}

// harness drives a machine with a manual clock.
type harness struct {
	t   *testing.T
	m   machine
	now time.Time
}

func newHarness(t *testing.T, cfg Config) *harness {
	return &harness{t: t, m: newMachine(cfg), now: t0}
}

func (h *harness) advance(d time.Duration) { h.now = h.now.Add(d) }

func (h *harness) must(tr Transition, err error) Transition {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
	return tr
}

// finish jumps to the end of the running phase and ticks.
func (h *harness) finish() Transition {
	h.t.Helper()
	if h.m.status != Running {
		h.t.Fatalf("finish: phase is %s, not running", h.m.status)
	}
	h.now = h.m.endsAt
	tr, ok := h.m.tick(h.now)
	if !ok {
		h.t.Fatal("finish: tick at the deadline did not expire the phase")
	}
	return tr
}

func (h *harness) want(phase Phase, status Status, remaining time.Duration, completed int) {
	h.t.Helper()
	got := h.m.state(h.now)
	if got.Phase != phase || got.Status != status || got.RemainingMs != remaining.Milliseconds() ||
		got.CompletedFocus != completed {
		h.t.Fatalf("state = %s/%s remaining=%dms completed=%d, want %s/%s remaining=%dms completed=%d",
			got.Phase, got.Status, got.RemainingMs, got.CompletedFocus,
			phase, status, remaining.Milliseconds(), completed)
	}
}

func TestInitialState(t *testing.T) {
	h := newHarness(t, testConfig())
	h.want(Focus, Idle, 25*time.Minute, 0)
	st := h.m.state(h.now)
	if st.DurationMs != (25*time.Minute).Milliseconds() || st.LongBreakEvery != 4 {
		t.Fatalf("state = %+v", st)
	}
}

func TestStartPauseResume(t *testing.T) {
	h := newHarness(t, testConfig())

	tr := h.must(h.m.start(h.now))
	if tr.Cause != CauseStart || tr.From.Status != Idle || tr.To.Status != Running {
		t.Fatalf("start transition = %+v", tr)
	}
	h.advance(10 * time.Minute)
	h.want(Focus, Running, 15*time.Minute, 0)

	h.must(h.m.pause(h.now))
	h.advance(time.Hour) // paused time does not count
	h.want(Focus, Paused, 15*time.Minute, 0)

	h.must(h.m.resume(h.now))
	h.advance(14 * time.Minute)
	h.want(Focus, Running, time.Minute, 0)
	if _, ok := h.m.tick(h.now); ok {
		t.Fatal("tick before the deadline expired the phase")
	}

	tr = h.finish()
	if tr.Cause != CauseExpire || tr.From.Phase != Focus || tr.From.RemainingMs != 0 || tr.To.Phase != ShortBreak {
		t.Fatalf("expire transition = %+v", tr)
	}
	h.want(ShortBreak, Running, 5*time.Minute, 1) // AutoStartBreaks
}

func TestInvalidCommands(t *testing.T) {
	tests := []struct {
		name  string
		setup func(h *harness)
		cmd   func(m *machine, now time.Time) (Transition, error)
	}{
		{"pause while idle", func(*harness) {}, (*machine).pause},
		{"resume while idle", func(*harness) {}, (*machine).resume},
		{"start while running", func(h *harness) { h.must(h.m.start(h.now)) }, (*machine).start},
		{"resume while running", func(h *harness) { h.must(h.m.start(h.now)) }, (*machine).resume},
		{"start while paused", func(h *harness) {
			h.must(h.m.start(h.now))
			h.must(h.m.pause(h.now))
		}, (*machine).start},
		{"pause while paused", func(h *harness) {
			h.must(h.m.start(h.now))
			h.must(h.m.pause(h.now))
		}, (*machine).pause},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, testConfig())
			tt.setup(h)
			before := h.m.state(h.now)
			if _, err := tt.cmd(&h.m, h.now); !errors.Is(err, ErrInvalidCommand) {
				t.Fatalf("err = %v, want ErrInvalidCommand", err)
			}
			if after := h.m.state(h.now); after != before {
				t.Fatalf("rejected command changed state: %+v -> %+v", before, after)
			}
		})
	}
}

func TestFullCycleWithLongBreak(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStartFocus = true
	h := newHarness(t, cfg)
	h.must(h.m.start(h.now))

	for i := 1; i <= 3; i++ {
		h.finish()
		h.want(ShortBreak, Running, 5*time.Minute, i)
		h.finish()
		h.want(Focus, Running, 25*time.Minute, i)
	}
	h.finish()
	h.want(LongBreak, Running, 15*time.Minute, 4)
	h.finish()
	h.want(Focus, Running, 25*time.Minute, 0) // new cycle
}

func TestAutoStart(t *testing.T) {
	tests := []struct {
		name                             string
		autoBreaks, autoFocus            bool
		wantBreakStatus, wantFocusStatus Status
	}{
		{"both on", true, true, Running, Running},
		{"breaks only", true, false, Running, Idle},
		{"focus only", false, true, Idle, Running},
		{"both off", false, false, Idle, Idle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.AutoStartBreaks, cfg.AutoStartFocus = tt.autoBreaks, tt.autoFocus
			h := newHarness(t, cfg)
			h.must(h.m.start(h.now))

			h.finish()
			h.want(ShortBreak, tt.wantBreakStatus, 5*time.Minute, 1)
			if h.m.status == Idle {
				h.must(h.m.start(h.now))
			}
			h.finish()
			h.want(Focus, tt.wantFocusStatus, 25*time.Minute, 1)
		})
	}
}

func TestSkip(t *testing.T) {
	t.Run("skipped focus is not counted", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.must(h.m.start(h.now))
		h.advance(time.Minute)
		tr := h.m.skip(h.now)
		if tr.Cause != CauseSkip || tr.From.Status != Running {
			t.Fatalf("skip transition = %+v", tr)
		}
		h.want(ShortBreak, Idle, 5*time.Minute, 0)
	})

	t.Run("skipping the would-be fourth focus gives a short break", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.m.completedFocus = 3
		h.m.skip(h.now)
		h.want(ShortBreak, Idle, 5*time.Minute, 3)
	})

	t.Run("skipping a short break keeps the counter", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.must(h.m.start(h.now))
		h.finish()
		h.m.skip(h.now)
		h.want(Focus, Idle, 25*time.Minute, 1)
	})

	t.Run("skipping a long break starts a new cycle", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.m.completedFocus = 3
		h.must(h.m.start(h.now))
		h.finish()
		h.want(LongBreak, Running, 15*time.Minute, 4)
		h.m.skip(h.now)
		h.want(Focus, Idle, 25*time.Minute, 0)
	})

	t.Run("skip while paused", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.must(h.m.start(h.now))
		h.must(h.m.pause(h.now))
		h.m.skip(h.now)
		h.want(ShortBreak, Idle, 5*time.Minute, 0)
	})
}

func TestReset(t *testing.T) {
	h := newHarness(t, testConfig())
	h.m.completedFocus = 2
	h.must(h.m.start(h.now))
	h.finish()
	h.advance(2 * time.Minute)

	tr := h.m.reset(h.now)
	if tr.Cause != CauseReset || tr.From.Phase != ShortBreak || tr.From.Status != Running {
		t.Fatalf("reset transition = %+v", tr)
	}
	h.want(Focus, Idle, 25*time.Minute, 0)
	if _, ok := h.m.tick(h.now.Add(time.Hour)); ok {
		t.Fatal("idle timer expired after reset")
	}
}

func TestSleepOvershootExpiresOnce(t *testing.T) {
	cfg := testConfig()
	cfg.AutoStartFocus = true
	h := newHarness(t, cfg)
	h.must(h.m.start(h.now))

	h.advance(3 * time.Hour) // longer than several whole phases
	tr, ok := h.m.tick(h.now)
	if !ok || tr.Cause != CauseExpire || tr.To.Phase != ShortBreak {
		t.Fatalf("tick after sleep = %+v, %v", tr, ok)
	}
	h.want(ShortBreak, Running, 5*time.Minute, 1) // full break, counted from wake-up
	if _, ok := h.m.tick(h.now); ok {
		t.Fatal("second tick right after wake-up expired again")
	}
}

func TestSetConfig(t *testing.T) {
	t.Run("idle phase takes the new duration", func(t *testing.T) {
		h := newHarness(t, testConfig())
		cfg := testConfig()
		cfg.Focus = 50 * time.Minute
		h.m.setConfig(cfg)
		h.want(Focus, Idle, 50*time.Minute, 0)
		if h.m.state(h.now).DurationMs != (50 * time.Minute).Milliseconds() {
			t.Fatal("duration not updated")
		}
	})

	t.Run("running phase keeps its duration until the next phase", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.must(h.m.start(h.now))
		h.advance(5 * time.Minute)
		cfg := testConfig()
		cfg.Focus, cfg.ShortBreak = 50*time.Minute, 10*time.Minute
		h.m.setConfig(cfg)
		h.want(Focus, Running, 20*time.Minute, 0)

		h.finish()
		h.want(ShortBreak, Running, 10*time.Minute, 1)
	})

	t.Run("lowering LongBreakEvery below the counter", func(t *testing.T) {
		h := newHarness(t, testConfig())
		h.m.completedFocus = 3
		cfg := testConfig()
		cfg.LongBreakEvery = 2
		h.m.setConfig(cfg)
		h.must(h.m.start(h.now))
		h.finish()
		h.want(LongBreak, Running, 15*time.Minute, 4)
	})
}

func TestConfigValidate(t *testing.T) {
	bad := []func(*Config){
		func(c *Config) { c.Focus = 0 },
		func(c *Config) { c.ShortBreak = -time.Second },
		func(c *Config) { c.LongBreak = 0 },
		func(c *Config) { c.LongBreakEvery = 0 },
	}
	for i, mutate := range bad {
		cfg := testConfig()
		mutate(&cfg)
		if err := cfg.validate(); err == nil {
			t.Errorf("case %d: invalid config accepted: %+v", i, cfg)
		}
	}
	if err := testConfig().validate(); err != nil {
		t.Fatal(err)
	}
}
