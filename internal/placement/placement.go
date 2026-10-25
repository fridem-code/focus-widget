// Package placement decides where the widget window goes (docs/SPEC.md §3.4):
// it opens at the saved position if the window can still be grabbed there,
// otherwise in the bottom-right corner of the primary screen, and it always
// fits on the screen it is on, also after switching modes.
package placement

// Rect is a rectangle in DIPs (device-independent pixels).
type Rect struct{ X, Y, W, H int }

const (
	// Margin keeps the default position off the screen edges.
	Margin = 16
	// GrabHeight is the strip at the top of the window used to drag it (the
	// title bar in expanded mode, the whole bar in compact mode is taller).
	GrabHeight = 36
	// minGrabWidth/minGrabHeight: how much of that strip must lie on a work
	// area for the window to count as reachable.
	minGrabWidth  = 80
	minGrabHeight = 20
)

// Place returns the top-left corner for a window of size w×h. saved is the
// last remembered top-left (nil if never moved); workAreas are the usable
// areas of all screens (without taskbars); primary is the primary screen's.
// A window that is reachable but sticks out of its screen is pulled in.
func Place(saved *[2]int, w, h int, workAreas []Rect, primary Rect) (x, y int) {
	if saved != nil {
		if area, ok := reachable(Rect{saved[0], saved[1], w, min(GrabHeight, h)}, workAreas); ok {
			return fit(saved[0], saved[1], w, h, area)
		}
	}
	return primary.X + primary.W - w - Margin, primary.Y + primary.H - h - Margin
}

// Resize returns the top-left corner for window cur resized to w×h. The window
// keeps the edges nearest to its screen's edges where they are, so a widget in
// the bottom-right corner grows up and to the left and shrinks back into the
// same corner, and it still fits on the screen. Off every screen, it keeps its
// top-left corner.
func Resize(cur Rect, w, h int, workAreas []Rect) (x, y int) {
	area, ok := mostOverlapped(cur, workAreas)
	if !ok {
		return cur.X, cur.Y
	}
	x, y = cur.X, cur.Y
	if cur.X+cur.W/2 > area.X+area.W/2 { // right half: keep the right edge
		x = cur.X + cur.W - w
	}
	if cur.Y+cur.H/2 > area.Y+area.H/2 { // lower half: keep the bottom edge
		y = cur.Y + cur.H - h
	}
	return fit(x, y, w, h, area)
}

// fit moves a w×h window at (x, y) the least so it lies inside area; a window
// larger than the area keeps its top-left corner on it.
func fit(x, y, w, h int, area Rect) (int, int) {
	x = max(min(x, area.X+area.W-w), area.X)
	y = max(min(y, area.Y+area.H-h), area.Y)
	return x, y
}

// reachable returns the work area holding enough of the grab strip.
func reachable(strip Rect, areas []Rect) (Rect, bool) {
	for _, a := range areas {
		ix, iy := overlap(strip, a)
		if ix >= min(minGrabWidth, strip.W) && iy >= min(minGrabHeight, strip.H) {
			return a, true
		}
	}
	return Rect{}, false
}

// mostOverlapped returns the work area sharing the most of r.
func mostOverlapped(r Rect, areas []Rect) (Rect, bool) {
	best, bestArea := Rect{}, 0
	for _, a := range areas {
		if ix, iy := overlap(r, a); ix > 0 && iy > 0 && ix*iy > bestArea {
			best, bestArea = a, ix*iy
		}
	}
	return best, bestArea > 0
}

func overlap(r, a Rect) (ix, iy int) {
	ix = min(r.X+r.W, a.X+a.W) - max(r.X, a.X)
	iy = min(r.Y+r.H, a.Y+a.H) - max(r.Y, a.Y)
	return ix, iy
}
