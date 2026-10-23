// Package app assembles the Wails application: options, the widget window and,
// later, the tray (see docs/SPEC.md §3.4–3.5).
package app

import (
	"io/fs"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	name        = "Focus Widget"
	description = "Pomodoro timer widget with YouTube focus music"

	expandedWidth  = 380
	expandedHeight = 540
)

// New builds the application with its main widget window. assets must contain
// the built frontend (frontend/dist).
func New(assets fs.FS) *application.App {
	wailsApp := application.New(application.Options{
		Name:        name,
		Description: description,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			// Without this flag WebView2 blocks unmuted playVideo() when the page has
			// not been clicked (tray "Start", phase auto-start). See docs/spike-results.md §3.
			AdditionalBrowserArgs: []string{"--autoplay-policy=no-user-gesture-required"},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            name,
		Width:            expandedWidth,
		Height:           expandedHeight,
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		BackgroundColour: application.NewRGB(18, 18, 24),
		URL:              "/",
	})

	return wailsApp
}
