package timer

import (
	"context"
	"sync"
	"time"
)

// TickInterval is how often Run checks the deadline; state is published only
// when the displayed second changes.
const TickInterval = 250 * time.Millisecond

// Timer drives the state machine with a ticker and publishes changes to
// listeners. All methods are safe for concurrent use (commands arrive from the
// UI and from the tray on different goroutines).
//
// Listeners run synchronously, in the order events were produced, and may call
// State but must not call commands (Start, Pause, …) or they deadlock.
type Timer struct {
	clock Clock

	mu           sync.Mutex
	m            machine
	shownSecond  int64 // last published remaining time, in whole seconds rounded up
	onState      []func(State)
	onTransition []func(Transition)

	// emitMu is taken before mu is released, so listeners see events in the
	// order they happened even when ticks and commands race.
	emitMu sync.Mutex
}

func New(cfg Config, clock Clock) (*Timer, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	t := &Timer{clock: clock, m: newMachine(cfg)}
	t.shownSecond = ceilSeconds(t.m.state(clock.Now()).RemainingMs)
	return t, nil
}

// OnState registers fn for every published state (each second while running
// and after every change).
func (t *Timer) OnState(fn func(State)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onState = append(t.onState, fn)
}

// OnTransition registers fn for every transition; it is called before the
// OnState listeners for the same change.
func (t *Timer) OnTransition(fn func(Transition)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onTransition = append(t.onTransition, fn)
}

// Run ticks the timer until ctx is cancelled.
func (t *Timer) Run(ctx context.Context) {
	ticker := t.clock.NewTicker(TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			t.tick()
		}
	}
}

func (t *Timer) State() State {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.m.state(t.clock.Now())
}

func (t *Timer) Start() error  { return t.command(t.m.start) }
func (t *Timer) Pause() error  { return t.command(t.m.pause) }
func (t *Timer) Resume() error { return t.command(t.m.resume) }

func (t *Timer) Skip() {
	_ = t.command(func(now time.Time) (Transition, error) { return t.m.skip(now), nil })
}

func (t *Timer) Reset() {
	_ = t.command(func(now time.Time) (Transition, error) { return t.m.reset(now), nil })
}

// UpdateConfig applies new settings (see machine.setConfig) and publishes the
// resulting state.
func (t *Timer) UpdateConfig(cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	t.mu.Lock()
	t.m.setConfig(cfg)
	t.publish(nil, t.m.state(t.clock.Now()))
	return nil
}

func (t *Timer) command(apply func(now time.Time) (Transition, error)) error {
	t.mu.Lock()
	now := t.clock.Now()
	tr, err := apply(now)
	if err != nil {
		t.mu.Unlock()
		return err
	}
	t.publish([]Transition{tr}, t.m.state(now))
	return nil
}

func (t *Timer) tick() {
	t.mu.Lock()
	now := t.clock.Now()
	var trs []Transition
	if tr, ok := t.m.tick(now); ok {
		trs = append(trs, tr)
	}
	st := t.m.state(now)
	if len(trs) == 0 && ceilSeconds(st.RemainingMs) == t.shownSecond {
		t.mu.Unlock()
		return
	}
	t.publish(trs, st)
}

// publish must be called with mu held; it releases mu and notifies listeners.
func (t *Timer) publish(trs []Transition, st State) {
	t.shownSecond = ceilSeconds(st.RemainingMs)
	onTransition, onState := t.onTransition, t.onState
	t.emitMu.Lock()
	t.mu.Unlock()
	defer t.emitMu.Unlock()

	for _, tr := range trs {
		for _, fn := range onTransition {
			fn(tr)
		}
	}
	for _, fn := range onState {
		fn(st)
	}
}

func ceilSeconds(ms int64) int64 { return (ms + 999) / 1000 }
