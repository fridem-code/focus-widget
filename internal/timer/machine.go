// Package timer implements the Pomodoro state machine (docs/SPEC.md §5.1) and a
// driver that ticks it and publishes state changes. It has no Wails dependency.
package timer

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type Phase string

const (
	Focus      Phase = "focus"
	ShortBreak Phase = "short_break"
	LongBreak  Phase = "long_break"
)

type Status string

const (
	Idle    Status = "idle"
	Running Status = "running"
	Paused  Status = "paused"
)

// Cause names what triggered a transition.
type Cause string

const (
	CauseStart  Cause = "start"
	CausePause  Cause = "pause"
	CauseResume Cause = "resume"
	CauseExpire Cause = "expire"
	CauseSkip   Cause = "skip"
	CauseReset  Cause = "reset"
)

// ErrInvalidCommand is returned for a command the current status does not
// allow, e.g. Pause while idle.
var ErrInvalidCommand = errors.New("timer: command not allowed in current state")

type Config struct {
	Focus      time.Duration
	ShortBreak time.Duration
	LongBreak  time.Duration
	// LongBreakEvery is the number of completed focus phases before a long break.
	LongBreakEvery  int
	AutoStartBreaks bool
	AutoStartFocus  bool
}

func (c Config) validate() error {
	if c.Focus <= 0 || c.ShortBreak <= 0 || c.LongBreak <= 0 {
		return fmt.Errorf("timer: phase durations must be positive: %+v", c)
	}
	if c.LongBreakEvery < 1 {
		return fmt.Errorf("timer: LongBreakEvery must be at least 1, got %d", c.LongBreakEvery)
	}
	return nil
}

func (c Config) duration(p Phase) time.Duration {
	switch p {
	case ShortBreak:
		return c.ShortBreak
	case LongBreak:
		return c.LongBreak
	default:
		return c.Focus
	}
}

// State is the snapshot published to the frontend as the timer:state event.
type State struct {
	Phase          Phase  `json:"phase"`
	Status         Status `json:"status"`
	DurationMs     int64  `json:"durationMs"`
	RemainingMs    int64  `json:"remainingMs"`
	CompletedFocus int    `json:"completedFocus"`
	LongBreakEvery int    `json:"longBreakEvery"`
}

type Transition struct {
	From  State `json:"from"`
	To    State `json:"to"`
	Cause Cause `json:"cause"`
}

// machine is the pure state machine: every method takes the current time and
// nothing else, so it is trivially testable. It is not safe for concurrent use.
type machine struct {
	cfg            Config
	phase          Phase
	status         Status
	duration       time.Duration // length of the current phase, fixed when it began
	remaining      time.Duration // meaningful while idle or paused
	endsAt         time.Time     // meaningful while running
	completedFocus int
}

func newMachine(cfg Config) machine {
	return machine{
		cfg:       cfg,
		phase:     Focus,
		status:    Idle,
		duration:  cfg.Focus,
		remaining: cfg.Focus,
	}
}

func (m *machine) state(now time.Time) State {
	remaining := m.remaining
	if m.status == Running {
		remaining = max(m.endsAt.Sub(now), 0)
	}
	return State{
		Phase:          m.phase,
		Status:         m.status,
		DurationMs:     m.duration.Milliseconds(),
		RemainingMs:    remaining.Milliseconds(),
		CompletedFocus: m.completedFocus,
		LongBreakEvery: m.cfg.LongBreakEvery,
	}
}

// do records the state around change, which fails with ErrInvalidCommand unless
// the machine is in one of the allowed statuses (none means any).
func (m *machine) do(now time.Time, cause Cause, allowed []Status, change func()) (Transition, error) {
	if len(allowed) > 0 && !slices.Contains(allowed, m.status) {
		return Transition{}, fmt.Errorf("%w: %s while %s", ErrInvalidCommand, cause, m.status)
	}
	from := m.state(now)
	change()
	return Transition{From: from, To: m.state(now), Cause: cause}, nil
}

func (m *machine) start(now time.Time) (Transition, error) {
	return m.do(now, CauseStart, []Status{Idle}, func() { m.run(now) })
}

func (m *machine) pause(now time.Time) (Transition, error) {
	return m.do(now, CausePause, []Status{Running}, func() {
		m.remaining = max(m.endsAt.Sub(now), 0)
		m.status = Paused
	})
}

func (m *machine) resume(now time.Time) (Transition, error) {
	return m.do(now, CauseResume, []Status{Paused}, func() { m.run(now) })
}

// skip moves to the next phase without counting the current one as completed.
func (m *machine) skip(now time.Time) Transition {
	tr, _ := m.do(now, CauseSkip, nil, func() { m.advance(false) })
	return tr
}

func (m *machine) reset(now time.Time) Transition {
	tr, _ := m.do(now, CauseReset, nil, func() {
		m.completedFocus = 0
		m.enter(Focus)
	})
	return tr
}

// tick expires the running phase once its deadline has passed. However long
// the deadline was overshot (e.g. the PC slept), it produces exactly one
// expiry, and an auto-started next phase gets its full length from now.
func (m *machine) tick(now time.Time) (Transition, bool) {
	if m.status != Running || now.Before(m.endsAt) {
		return Transition{}, false
	}
	tr, _ := m.do(now, CauseExpire, nil, func() {
		m.advance(true)
		autoStart := m.cfg.AutoStartFocus
		if m.phase != Focus {
			autoStart = m.cfg.AutoStartBreaks
		}
		if autoStart {
			m.run(now)
		}
	})
	return tr, true
}

// setConfig applies new settings. Durations take effect immediately for an idle
// phase and from the next phase otherwise.
func (m *machine) setConfig(cfg Config) {
	m.cfg = cfg
	if m.status == Idle {
		m.duration = cfg.duration(m.phase)
		m.remaining = m.duration
	}
}

func (m *machine) run(now time.Time) {
	m.endsAt = now.Add(m.remaining)
	m.status = Running
}

// advance enters the phase after the current one, idle. A long break follows
// once LongBreakEvery focus phases have been completed (skipped ones don't
// count); finishing or skipping a long break starts a new cycle.
func (m *machine) advance(completed bool) {
	next := Focus
	switch m.phase {
	case Focus:
		if completed {
			m.completedFocus++
		}
		next = ShortBreak
		if completed && m.completedFocus >= m.cfg.LongBreakEvery {
			next = LongBreak
		}
	case LongBreak:
		m.completedFocus = 0
	}
	m.enter(next)
}

func (m *machine) enter(p Phase) {
	m.phase = p
	m.status = Idle
	m.duration = m.cfg.duration(p)
	m.remaining = m.duration
}
