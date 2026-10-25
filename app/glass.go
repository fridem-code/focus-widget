package app

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/glass"
)

// windowGlass is services.Glass for the widget window.
type windowGlass struct {
	window func() *application.WebviewWindow // created after the services
	logger *slog.Logger
}

func (g windowGlass) Apply(dark bool) bool {
	if !glass.Available() { // transparency effects are off in Windows
		return false
	}
	hwnd := g.window().NativeWindow()
	if hwnd == nil {
		return false
	}
	if err := glass.Apply(uintptr(hwnd), dark); err != nil {
		g.logger.Warn("turn the window glass on", "err", err)
		return false
	}
	return true
}

func (windowGlass) SystemDark() bool { return glass.SystemDark() }
