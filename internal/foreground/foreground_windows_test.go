package foreground

import "testing"

func TestWatchStartsAndStops(t *testing.T) {
	stop, err := Watch(func() {})
	if err != nil {
		t.Fatal(err)
	}
	stop()
}

func TestPinRejectsAMissingWindow(t *testing.T) {
	if err := Pin(0); err == nil {
		t.Fatal("Pin(0) succeeded")
	}
}
