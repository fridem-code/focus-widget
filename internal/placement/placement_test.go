package placement

import "testing"

func TestPlace(t *testing.T) {
	primary := Rect{0, 0, 1920, 1032}              // taskbar at the bottom
	left := Rect{-1280, 100, 1280, 984}            // monitor to the left, lower
	defaultX, defaultY := 1920-380-16, 1032-540-16 // bottom-right of primary
	at := func(x, y int) *[2]int { return &[2]int{x, y} }

	tests := []struct {
		name  string
		saved *[2]int
		areas []Rect
		wantX int
		wantY int
	}{
		{"never moved", nil, []Rect{primary}, defaultX, defaultY},
		{"saved on primary", at(100, 200), []Rect{primary}, 100, 200},
		{"saved on the left monitor (negative x)", at(-900, 300), []Rect{primary, left}, -900, 300},
		{"left monitor unplugged", at(-900, 300), []Rect{primary}, defaultX, defaultY},
		{"mostly off the right edge: pulled in", at(1800, 200), []Rect{primary}, 1920 - 380, 200},
		{"saved compact spot near the bottom, opened expanded: pulled up (B4)", at(3, 813), []Rect{primary}, 3, 1032 - 540},
		{"on the left monitor, sticking out of its top: pulled down", at(-900, 90), []Rect{primary, left}, -900, 100},
		{"title bar off the right edge", at(1860, 200), []Rect{primary}, defaultX, defaultY},
		{"title bar above the top edge", at(100, -30), []Rect{primary}, defaultX, defaultY},
		{"title bar under the taskbar", at(100, 1020), []Rect{primary}, defaultX, defaultY},
		{"in the gap above the lower left monitor", at(-900, 20), []Rect{primary, left}, defaultX, defaultY},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := Place(tt.saved, 380, 540, tt.areas, primary)
			if x != tt.wantX || y != tt.wantY {
				t.Fatalf("Place = (%d, %d), want (%d, %d)", x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestPlaceCompactUsesItsOwnSize(t *testing.T) {
	primary := Rect{0, 0, 1920, 1032}
	x, y := Place(nil, 300, 96, []Rect{primary}, primary)
	if x != 1920-300-16 || y != 1032-96-16 {
		t.Fatalf("Place = (%d, %d)", x, y)
	}
}

func TestPlaceSecondaryPrimaryWithOffset(t *testing.T) {
	primary := Rect{2560, 0, 1920, 1040} // primary is not at the origin
	x, y := Place(nil, 380, 540, []Rect{{0, 0, 2560, 1400}, primary}, primary)
	if x != 2560+1920-380-16 || y != 1040-540-16 {
		t.Fatalf("Place = (%d, %d)", x, y)
	}
}

func TestResize(t *testing.T) {
	primary := Rect{0, 0, 1920, 1032}
	tests := []struct {
		name  string
		cur   Rect
		w, h  int
		areas []Rect
		wantX int
		wantY int
	}{
		{"top-left quarter keeps the top-left corner", Rect{100, 50, 300, 96}, 380, 540, []Rect{primary}, 100, 50},
		{"bottom-right corner grows up and left", Rect{1604, 920, 300, 96}, 380, 540, []Rect{primary}, 1524, 476},
		{"and shrinks back into the same corner", Rect{1524, 476, 380, 540}, 300, 96, []Rect{primary}, 1604, 920},
		{"compact near the bottom-left grows up (B4)", Rect{3, 813, 300, 96}, 380, 540, []Rect{{0, 0, 1536, 912}}, 3, 369},
		{"would stick out of the top: pulled down", Rect{100, 400, 300, 96}, 380, 540, []Rect{{0, 0, 1920, 800}}, 100, 0},
		{"screen smaller than the window: top-left on it", Rect{10, 10, 300, 96}, 380, 540, []Rect{{0, 0, 300, 400}}, 0, 0},
		{"on the second monitor", Rect{-500, 700, 300, 96}, 380, 540, []Rect{primary, {-1280, 100, 1280, 984}}, -580, 256},
		{"off every screen keeps its corner", Rect{5000, 5000, 300, 96}, 380, 540, []Rect{primary}, 5000, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := Resize(tt.cur, tt.w, tt.h, tt.areas)
			if x != tt.wantX || y != tt.wantY {
				t.Fatalf("Resize = (%d, %d), want (%d, %d)", x, y, tt.wantX, tt.wantY)
			}
		})
	}
}
