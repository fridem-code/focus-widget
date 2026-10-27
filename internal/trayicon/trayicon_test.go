package trayicon

import (
	"bytes"
	"image/png"
	"testing"
)

// size: the ring tests look at a 64 px icon.
const size = 64

// white reports whether the pixel is (nearly) white and opaque.
func white(t *testing.T, r Ring, x, y int) bool {
	t.Helper()
	c := Draw(r, size).NRGBAAt(x, y)
	return c.A > 240 && c.R > 230 && c.G > 230 && c.B > 230
}

// at converts a length in icon units to pixels.
func at(units float64) int { return int(units * size) }

func TestIdleIsTheAppIcon(t *testing.T) {
	if got := PNG(Ring{}, 20); !bytes.Equal(got, Idle) {
		t.Fatal("idle icon is not the embedded app icon")
	}
	if _, err := png.Decode(bytes.NewReader(Idle)); err != nil {
		t.Fatalf("embedded idle icon: %v", err)
	}
}

func TestRingShowsTheTimeLeftClockwiseFromTheTop(t *testing.T) {
	top := [2]int{32, 32 - at(ringRadius)}   // 12 o'clock on the ring
	right := [2]int{32 + at(ringRadius), 32} // 3 o'clock
	left := [2]int{32 - at(ringRadius), 32}  // 9 o'clock

	full := Ring{Phase: "focus", Left: Steps}
	if !white(t, full, top[0], top[1]) || !white(t, full, left[0], left[1]) {
		t.Fatal("full ring is not white all round")
	}
	half := Ring{Phase: "focus", Left: Steps / 2}
	if !white(t, half, right[0], right[1]) || white(t, half, left[0], left[1]) {
		t.Fatal("half ring: want the right half white, the left half track only")
	}
	if white(t, Ring{Phase: "focus", Left: 0}, right[0], right[1]) {
		t.Fatal("empty ring has an arc")
	}
}

func TestPhaseColourAndTransparentCorners(t *testing.T) {
	img := Draw(Ring{Phase: "short_break", Left: Steps}, 32)
	if c := img.NRGBAAt(0, 0); c.A != 0 {
		t.Fatalf("corner = %+v, want transparent", c)
	}
	c := img.NRGBAAt(3, 16) // inside the square, outside the ring
	if c.R != 0xb5 || c.G != 0x53 || c.B != 0x0f || c.A != 0xff {
		t.Fatalf("background = %+v, want the short-break colour", c)
	}
}

func TestPausedShowsBarsAndADimmerArc(t *testing.T) {
	running := Ring{Phase: "focus", Left: Steps}
	paused := Ring{Phase: "focus", Left: Steps, Paused: true}
	barX := 32 + at(barGap/2+barWidth/2)
	if white(t, running, barX, 32) || !white(t, paused, barX, 32) {
		t.Fatal("pause bars: want them only while paused")
	}
	top := 32 - at(ringRadius)
	if white(t, paused, 32, top) {
		t.Fatal("paused arc should be dimmer than white")
	}
}

func TestPNGEncodesAtTheRequestedSize(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(PNG(Ring{Phase: "long_break", Left: 10}, 20)))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 20 || b.Dy() != 20 {
		t.Fatalf("size = %v", b)
	}
}
