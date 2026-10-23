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
		{"mostly off the right edge but the title bar is reachable", at(1800, 200), []Rect{primary}, 1800, 200},
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
