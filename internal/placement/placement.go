// Package placement decides where the widget window opens (docs/SPEC.md §3.4):
// the saved position if the window can still be grabbed there, otherwise the
// bottom-right corner of the primary screen.
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
func Place(saved *[2]int, w, h int, workAreas []Rect, primary Rect) (x, y int) {
	if saved != nil && reachable(Rect{saved[0], saved[1], w, min(GrabHeight, h)}, workAreas) {
		return saved[0], saved[1]
	}
	return primary.X + primary.W - w - Margin, primary.Y + primary.H - h - Margin
}

// reachable reports whether enough of the grab strip is on some work area.
func reachable(strip Rect, areas []Rect) bool {
	for _, a := range areas {
		ix := min(strip.X+strip.W, a.X+a.W) - max(strip.X, a.X)
		iy := min(strip.Y+strip.H, a.Y+a.H) - max(strip.Y, a.Y)
		if ix >= min(minGrabWidth, strip.W) && iy >= min(minGrabHeight, strip.H) {
			return true
		}
	}
	return false
}
