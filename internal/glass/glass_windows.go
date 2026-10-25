// Package glass draws the widget's blurred window background on Windows
// (docs/spike-results.md, T16).
//
// The DWM system backdrop (Acrylic) the widget used before is replaced with a
// solid fill whenever the window is inactive, which for an always-on-top
// widget is nearly always (docs/BUGS.md B1). The accent policy of the
// undocumented SetWindowCompositionAttribute keeps blurring inactive windows.
// It only blurs with zero DWM frame margins, so the corners are rounded with
// DWMWA_WINDOW_CORNER_PREFERENCE instead (Windows 11; square on Windows 10).
package glass

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	dwmwaWindowCornerPreference = 33
	dwmwaSystemBackdropType     = 38
	dwmwcpRound                 = 2
	dwmsbtNone                  = 1

	wcaAccentPolicy               = 19
	accentEnableAcrylicBlurBehind = 4
	accentFlagDrawAllBorders      = 2
)

// Accent colours, 0xAABBGGRR. The alpha is low: the tint itself is the page's
// --glass. The colour still sets how light the acrylic is, so it follows the
// theme (rgb(16,24,32) dark, rgb(246,247,250) light).
const (
	darkAccent  = 0x10201810
	lightAccent = 0x10FAF7F6
)

var (
	dwmapi                            = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttribute         = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmExtendFrameIntoClientArea  = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	user32                            = windows.NewLazySystemDLL("user32.dll")
	procSetWindowCompositionAttribute = user32.NewProc("SetWindowCompositionAttribute")
)

type accentPolicy struct {
	state, flags  int32
	gradientColor uint32
	animationID   int32
}

type compositionData struct {
	attrib int32
	data   unsafe.Pointer
	size   uintptr
}

type margins struct{ left, right, top, bottom int32 }

const personalize = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

// Available reports whether Windows draws transparency effects (Settings →
// Personalization → Colors). Without them the accent would not blur, so the
// widget stays opaque.
func Available() bool {
	return readPersonalize("EnableTransparency", 1) != 0
}

// SystemDark reports whether Windows apps use the dark theme.
func SystemDark() bool {
	return readPersonalize("AppsUseLightTheme", 1) == 0
}

func readPersonalize(name string, fallback uint64) uint64 {
	k, err := registry.OpenKey(registry.CURRENT_USER, personalize, registry.QUERY_VALUE)
	if err != nil {
		return fallback
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetIntegerValue(name)
	if err != nil {
		return fallback
	}
	return v
}

// Apply turns the glass on for hwnd, tinted for the dark or light theme.
func Apply(hwnd uintptr, dark bool) error {
	if hwnd == 0 {
		return errors.New("no window")
	}
	none := int32(dwmsbtNone)
	if err := setAttribute(hwnd, dwmwaSystemBackdropType, &none); err != nil {
		return fmt.Errorf("system backdrop off: %w", err)
	}
	var zero margins
	if r, _, _ := procDwmExtendFrameIntoClientArea.Call(hwnd, uintptr(unsafe.Pointer(&zero))); r != 0 {
		return fmt.Errorf("DwmExtendFrameIntoClientArea: 0x%x", r)
	}
	round := int32(dwmwcpRound)
	_ = setAttribute(hwnd, dwmwaWindowCornerPreference, &round) // fails on Windows 10: square corners
	color := uint32(lightAccent)
	if dark {
		color = darkAccent
	}
	accent := accentPolicy{state: accentEnableAcrylicBlurBehind, flags: accentFlagDrawAllBorders, gradientColor: color}
	data := compositionData{attrib: wcaAccentPolicy, data: unsafe.Pointer(&accent), size: unsafe.Sizeof(accent)}
	if r, _, err := procSetWindowCompositionAttribute.Call(hwnd, uintptr(unsafe.Pointer(&data))); r == 0 {
		return fmt.Errorf("SetWindowCompositionAttribute: %w", err)
	}
	return nil
}

func setAttribute(hwnd uintptr, attr int32, value *int32) error {
	if r, _, _ := procDwmSetWindowAttribute.Call(hwnd, uintptr(attr), uintptr(unsafe.Pointer(value)), 4); r != 0 {
		return fmt.Errorf("DwmSetWindowAttribute(%d): 0x%x", attr, r)
	}
	return nil
}
