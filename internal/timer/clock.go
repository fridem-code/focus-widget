package timer

import "time"

// Clock abstracts time so tests can drive the timer deterministically.
type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) Ticker
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// SystemClock returns the real clock.
func SystemClock() Clock { return systemClock{} }

type systemClock struct{}

// Now drops the monotonic reading so deadlines are compared on the wall clock:
// time spent in sleep or hibernation always counts towards the running phase
// (docs/SPEC.md §3.1).
func (systemClock) Now() time.Time { return time.Now().Round(0) }

func (systemClock) NewTicker(d time.Duration) Ticker { return systemTicker{time.NewTicker(d)} }

type systemTicker struct{ t *time.Ticker }

func (s systemTicker) C() <-chan time.Time { return s.t.C }
func (s systemTicker) Stop()               { s.t.Stop() }
