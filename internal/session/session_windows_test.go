package session

import "testing"

// Locking the session can't be scripted; this checks the watcher sets up
// (window class, message-only window, WTS registration) and stops, twice.
func TestWatchStartsAndStops(t *testing.T) {
	for range 2 {
		stop, err := Watch(func(bool) {})
		if err != nil {
			t.Fatal(err)
		}
		stop()
	}
}
