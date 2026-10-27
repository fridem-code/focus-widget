package app

import "github.com/wailsapp/wails/v3/pkg/w32"

// trayIconSize is the tray's own icon size in pixels (20 at 125 %), so the
// drawn icon is used as is rather than rescaled.
func trayIconSize() int {
	if n := w32.GetSystemMetrics(w32.SM_CXSMICON); n > 0 {
		return n
	}
	return 16
}
