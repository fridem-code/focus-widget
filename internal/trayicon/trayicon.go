// Package trayicon draws the tray icon that shows the phase and how much of
// it is left (T21): a rounded square in the phase colour, as in the app icon,
// with a white ring of the time remaining; two bars in the middle while
// paused. Drawn at the tray's own icon size, so nothing gets rescaled.
package trayicon

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Steps is how finely the ring shows the time left: a 25-minute focus moves
// it every 25 s, so the icon is redrawn rarely.
const Steps = 60

// Ring is what the icon shows. The zero value is the idle icon.
type Ring struct {
	// Phase is the timer phase ("focus", "short_break", "long_break"); empty
	// while the timer is idle.
	Phase string
	// Left is the time remaining in steps, 0…Steps.
	Left   int
	Paused bool
}

// Idle is the app icon, for when no phase runs (made from build/appicon.png).
//
//go:embed idle.png
var Idle []byte

// Phase colours: the frontend's --accent (frontend/src/styles.css).
var phaseColours = map[string]color.NRGBA{
	"focus":       {0x4c, 0x63, 0xe6, 0xff},
	"short_break": {0xb5, 0x53, 0x0f, 0xff},
	"long_break":  {0x0e, 0x7c, 0x6b, 0xff},
}

// Geometry in units of the icon size.
const (
	squareInset  = 0.03
	squareRadius = 0.22
	ringRadius   = 0.29
	ringWidth    = 0.13
	trackAlpha   = 0.35
	pausedAlpha  = 0.6
	barWidth     = 0.08
	barHeight    = 0.24
	barGap       = 0.06
	supersample  = 4 // 4×4 samples per pixel for smooth edges
)

// PNG returns the icon for r as a PNG of size×size pixels: Idle for the
// zero value, the drawn ring otherwise.
func PNG(r Ring, size int) []byte {
	if r.Phase == "" {
		return Idle
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, Draw(r, size)) // encoding to memory cannot fail
	return buf.Bytes()
}

// Draw renders the ring icon.
func Draw(r Ring, size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	bg, ok := phaseColours[r.Phase]
	if !ok {
		bg = phaseColours["focus"]
	}
	left := float64(min(max(r.Left, 0), Steps)) / Steps
	arcAlpha := 1.0
	if r.Paused {
		arcAlpha = pausedAlpha
	}
	n := float64(size * supersample)
	for py := range size {
		for px := range size {
			// Average of premultiplied samples.
			var sr, sg, sb, sa float64
			for sy := range supersample {
				for sx := range supersample {
					x := (float64(px*supersample+sx) + 0.5) / n
					y := (float64(py*supersample+sy) + 0.5) / n
					if !inRoundedSquare(x, y) {
						continue
					}
					cr, cg, cb := float64(bg.R)/255, float64(bg.G)/255, float64(bg.B)/255
					if w := whiteAt(x, y, left, arcAlpha, r.Paused); w > 0 {
						cr, cg, cb = cr+(1-cr)*w, cg+(1-cg)*w, cb+(1-cb)*w
					}
					sr, sg, sb, sa = sr+cr, sg+cg, sb+cb, sa+1
				}
			}
			if sa == 0 {
				continue
			}
			k := float64(supersample * supersample)
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(math.Round(sr / sa * 255)),
				G: uint8(math.Round(sg / sa * 255)),
				B: uint8(math.Round(sb / sa * 255)),
				A: uint8(math.Round(sa / k * 255)),
			})
		}
	}
	return img
}

func inRoundedSquare(x, y float64) bool {
	lo, hi := squareInset+squareRadius, 1-squareInset-squareRadius
	if x < squareInset || x > 1-squareInset || y < squareInset || y > 1-squareInset {
		return false
	}
	// Outside the corners' circles only in the corner boxes.
	cx, cy := math.Max(lo, math.Min(x, hi)), math.Max(lo, math.Min(y, hi))
	return math.Hypot(x-cx, y-cy) <= squareRadius
}

// whiteAt is how much white covers the point: the ring's track and arc,
// and the pause bars.
func whiteAt(x, y, left, arcAlpha float64, paused bool) float64 {
	dx, dy := x-0.5, y-0.5
	if math.Abs(math.Hypot(dx, dy)-ringRadius) <= ringWidth/2 {
		// Clockwise from 12 o'clock: the arc is the time left.
		angle := math.Atan2(dx, -dy)
		if angle < 0 {
			angle += 2 * math.Pi
		}
		if angle <= left*2*math.Pi {
			return arcAlpha
		}
		return trackAlpha
	}
	if paused && math.Abs(dy) <= barHeight/2 {
		ax := math.Abs(dx)
		if ax >= barGap/2 && ax <= barGap/2+barWidth {
			return 1
		}
	}
	return 0
}
