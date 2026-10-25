package services

import "testing"

func TestTopmostKeeperPinsOnlyAVisibleAlwaysOnTopWindow(t *testing.T) {
	window, _, settings, _ := newWindowService(t)
	pins := 0
	keeper := NewTopmostKeeper(settings, window, func() { pins++ })

	keeper.ForegroundChanged() // not shown yet
	if pins != 0 {
		t.Fatal("pinned a window that is not shown")
	}

	window.Show()
	keeper.ForegroundChanged()
	if pins != 1 {
		t.Fatalf("pins = %d, want 1 for a shown always-on-top window", pins)
	}

	window.Hide()
	keeper.ForegroundChanged()
	if pins != 1 {
		t.Fatal("pinned a window hidden in the tray")
	}

	window.Show()
	if err := window.SetAlwaysOnTop(false); err != nil {
		t.Fatal(err)
	}
	keeper.ForegroundChanged()
	if pins != 1 {
		t.Fatal("pinned a window with always-on-top off")
	}
}
