package timer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers []*fakeTicker
}

func newFakeClock() *fakeClock { return &fakeClock{now: t0} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) NewTicker(time.Duration) Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	tk := &fakeTicker{c: make(chan time.Time, 1)}
	c.tickers = append(c.tickers, tk)
	return tk
}

// Advance moves time forward and fires every ticker once.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now, tickers := c.now, c.tickers
	c.mu.Unlock()
	for _, tk := range tickers {
		select {
		case tk.c <- now:
		default:
		}
	}
}

func (c *fakeClock) tickerCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.tickers)
}

type fakeTicker struct{ c chan time.Time }

func (t *fakeTicker) C() <-chan time.Time { return t.c }
func (t *fakeTicker) Stop()               {}

// recorder collects published events in order.
type recorder struct {
	mu     sync.Mutex
	events []any // State or Transition
}

func (r *recorder) attach(tm *Timer) {
	tm.OnTransition(func(tr Transition) { r.add(tr) })
	tm.OnState(func(st State) { r.add(st) })
}

func (r *recorder) add(e any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) take() []any {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev := r.events
	r.events = nil
	return ev
}

func newTestTimer(t *testing.T, cfg Config) (*Timer, *fakeClock, *recorder) {
	t.Helper()
	clock := newFakeClock()
	tm, err := New(cfg, clock)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	rec.attach(tm)
	return tm, clock, rec
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	cfg := testConfig()
	cfg.Focus = 0
	if _, err := New(cfg, newFakeClock()); err == nil {
		t.Fatal("New accepted a zero focus duration")
	}
}

func TestCommandPublishesTransitionThenState(t *testing.T) {
	tm, _, rec := newTestTimer(t, testConfig())
	if err := tm.Start(); err != nil {
		t.Fatal(err)
	}
	ev := rec.take()
	if len(ev) != 2 {
		t.Fatalf("got %d events, want transition + state: %+v", len(ev), ev)
	}
	tr, ok := ev[0].(Transition)
	if !ok || tr.Cause != CauseStart {
		t.Fatalf("first event = %+v, want start transition", ev[0])
	}
	if st, ok := ev[1].(State); !ok || st.Status != Running {
		t.Fatalf("second event = %+v, want running state", ev[1])
	}
}

func TestRejectedCommandPublishesNothing(t *testing.T) {
	tm, _, rec := newTestTimer(t, testConfig())
	if err := tm.Pause(); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("Pause() = %v", err)
	}
	if ev := rec.take(); len(ev) != 0 {
		t.Fatalf("published %+v", ev)
	}
}

func TestTickPublishesOncePerSecond(t *testing.T) {
	tm, clock, rec := newTestTimer(t, testConfig())
	_ = tm.Start()
	rec.take()

	for range 4 { // one second in TickInterval steps
		clock.Advance(TickInterval)
		tm.tick()
	}
	ev := rec.take()
	if len(ev) != 1 {
		t.Fatalf("got %d events in one second, want 1: %+v", len(ev), ev)
	}
	if st := ev[0].(State); st.RemainingMs != (25*time.Minute - time.Second).Milliseconds() {
		t.Fatalf("remaining = %d", st.RemainingMs)
	}

	clock.Advance(100 * time.Millisecond) // still within the same displayed second
	tm.tick()
	if ev := rec.take(); len(ev) != 0 {
		t.Fatalf("published within the same second: %+v", ev)
	}
}

func TestTickIdleOrPausedPublishesNothing(t *testing.T) {
	tm, clock, rec := newTestTimer(t, testConfig())
	clock.Advance(time.Hour)
	tm.tick()
	_ = tm.Start()
	_ = tm.Pause()
	rec.take()
	clock.Advance(time.Hour)
	tm.tick()
	if ev := rec.take(); len(ev) != 0 {
		t.Fatalf("published %+v", ev)
	}
}

func TestExpirePublishesTransition(t *testing.T) {
	tm, clock, rec := newTestTimer(t, testConfig())
	_ = tm.Start()
	rec.take()

	clock.Advance(25 * time.Minute)
	tm.tick()
	ev := rec.take()
	if len(ev) != 2 {
		t.Fatalf("got %+v, want expire transition + state", ev)
	}
	if tr := ev[0].(Transition); tr.Cause != CauseExpire || tr.To.Phase != ShortBreak {
		t.Fatalf("transition = %+v", tr)
	}
	if st := ev[1].(State); st.Phase != ShortBreak || st.Status != Running {
		t.Fatalf("state = %+v", st)
	}
}

func TestUpdateConfig(t *testing.T) {
	tm, _, rec := newTestTimer(t, testConfig())
	cfg := testConfig()
	cfg.Focus = 45 * time.Minute
	if err := tm.UpdateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	ev := rec.take()
	if len(ev) != 1 || ev[0].(State).RemainingMs != (45*time.Minute).Milliseconds() {
		t.Fatalf("events = %+v, want one state with 45m", ev)
	}

	cfg.LongBreakEvery = 0
	if err := tm.UpdateConfig(cfg); err == nil {
		t.Fatal("invalid config accepted")
	}
}

func TestListenerMayReadState(t *testing.T) {
	tm, _, _ := newTestTimer(t, testConfig())
	got := make(chan State, 1)
	tm.OnTransition(func(Transition) { got <- tm.State() })

	done := make(chan struct{})
	go func() {
		_ = tm.Start()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start deadlocked when a listener called State")
	}
	if st := <-got; st.Status != Running {
		t.Fatalf("State() inside listener = %+v", st)
	}
}

func TestRunTicksUntilCancelled(t *testing.T) {
	tm, clock, _ := newTestTimer(t, testConfig())
	states := make(chan State, 16)
	tm.OnState(func(st State) { states <- st })
	_ = tm.Start()
	<-states

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		tm.Run(ctx)
		close(stopped)
	}()
	for clock.tickerCount() == 0 {
		time.Sleep(time.Millisecond)
	}

	clock.Advance(time.Second)
	select {
	case st := <-states:
		if st.RemainingMs != (25*time.Minute - time.Second).Milliseconds() {
			t.Fatalf("state from Run = %+v", st)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not tick")
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

// Commands and ticks racing from many goroutines must still deliver events in
// order: each transition starts where the previous one ended, and the last
// state a listener sees equals the timer's final state.
func TestConcurrentCommandsKeepEventOrder(t *testing.T) {
	cfg := testConfig()
	cfg.Focus, cfg.ShortBreak, cfg.LongBreak = time.Second, time.Second, time.Second
	cfg.AutoStartFocus = true
	tm, clock, _ := newTestTimer(t, cfg)

	var mu sync.Mutex
	var last State
	var transitions []Transition
	tm.OnTransition(func(tr Transition) {
		mu.Lock()
		transitions = append(transitions, tr)
		mu.Unlock()
	})
	tm.OnState(func(st State) {
		mu.Lock()
		last = st
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				switch (g + i) % 6 {
				case 0:
					_ = tm.Start()
				case 1:
					_ = tm.Pause()
				case 2:
					_ = tm.Resume()
				case 3:
					tm.Skip()
				case 4:
					clock.Advance(300 * time.Millisecond)
					tm.tick()
				case 5:
					_ = tm.State()
				}
			}
		})
	}
	wg.Wait()

	tm.Reset() // a final, deterministic event
	mu.Lock()
	defer mu.Unlock()
	if want := tm.State(); last != want {
		t.Fatalf("last published state %+v, timer state %+v", last, want)
	}
	// RemainingMs legitimately moves between events, so compare the rest.
	type key struct {
		phase     Phase
		status    Status
		completed int
	}
	keyOf := func(s State) key { return key{s.Phase, s.Status, s.CompletedFocus} }
	for i := 1; i < len(transitions); i++ {
		if prev, cur := keyOf(transitions[i-1].To), keyOf(transitions[i].From); prev != cur {
			t.Fatalf("transition %d starts at %+v but the previous one ended at %+v", i, cur, prev)
		}
	}
	if len(transitions) < 100 {
		t.Fatalf("only %d transitions; the test is not exercising enough commands", len(transitions))
	}
}
