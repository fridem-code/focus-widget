package hotkey

import (
	"errors"
	"testing"
)

// An uncommon shortcut, so the test does not collide with the user's apps.
var testCombo = Combo{ModCtrl | ModAlt | ModShift | ModWin, 0x7B} // Ctrl+Alt+Shift+Win+F12

func TestWatchRegistersAndFreesShortcuts(t *testing.T) {
	stop, failed := Watch(map[int]Combo{1: testCombo}, func(int) {})
	if len(failed) != 0 {
		stop()
		t.Skipf("the test shortcut is taken on this machine: %v", failed)
	}

	// While held, a second registration of the same shortcut fails as taken.
	stop2, failed2 := Watch(map[int]Combo{2: testCombo}, func(int) {})
	stop2()
	if !errors.Is(failed2[2], ErrTaken) {
		t.Errorf("second registration: failed = %v, want ErrTaken", failed2)
	}

	// After stop it is free again.
	stop()
	stop3, failed3 := Watch(map[int]Combo{3: testCombo}, func(int) {})
	stop3()
	if len(failed3) != 0 {
		t.Errorf("after stop: failed = %v, want none", failed3)
	}
}
