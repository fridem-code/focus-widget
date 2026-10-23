// Package app assembles the Wails application: options, services, the widget
// window and, later, the tray (see docs/SPEC.md §3.4–3.5, §5).
package app

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/catalog"
	"focuswidget/internal/config"
	"focuswidget/internal/timer"
	"focuswidget/services"
)

const (
	name        = "Focus Widget"
	description = "Pomodoro timer widget with YouTube focus music"

	expandedWidth  = 380
	expandedHeight = 540
)

// New builds the application with its services and main widget window. assets
// must contain the built frontend (frontend/dist).
func New(assets fs.FS) (*application.App, error) {
	// File logging with rotation arrives in T13 (SPEC §4).
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	dir, err := config.DefaultDir()
	if err != nil {
		return nil, err
	}
	store := config.NewStore(dir, logger)
	settings, err := store.Load()
	if err != nil {
		// Load still returned usable defaults; the next save retries the file.
		logger.Error("load settings", "err", err)
	}

	genres := catalog.Load(dir, logger)

	t, err := timer.New(services.TimerConfig(settings.Timer), timer.SystemClock())
	if err != nil {
		return nil, fmt.Errorf("create timer: %w", err)
	}

	// Services are created before the app they emit through; events only flow
	// once the app runs, by which time emitter.app is set.
	emitter := &appEmitter{}
	wailsApp := application.New(application.Options{
		Name:        name,
		Description: description,
		Logger:      logger,
		Services: []application.Service{
			application.NewService(services.NewTimerService(t, emitter)),
			application.NewService(services.NewSettingsService(store, settings, t, emitter)),
			application.NewService(services.NewMusicService(genres)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			// Without this flag WebView2 blocks unmuted playVideo() when the page has
			// not been clicked (tray "Start", phase auto-start). See docs/spike-results.md §3.
			AdditionalBrowserArgs: append([]string{"--autoplay-policy=no-user-gesture-required"}, devBrowserArgs()...),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	emitter.app = wailsApp

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "main",
		Title:         name,
		Width:         expandedWidth,
		Height:        expandedHeight,
		Frameless:     true,
		AlwaysOnTop:   settings.Window.AlwaysOnTop,
		DisableResize: true,
		// Glass: system Acrylic behind a transparent webview, tinted by CSS. Windows
		// draws the corners and shadow and swaps Acrylic for a solid fill while the
		// window is inactive (docs/spike-results.md, T0.5).
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows: application.WindowsWindow{
			BackdropType: application.Acrylic,
		},
		URL: "/",
	})

	return wailsApp, nil
}

type appEmitter struct {
	app *application.App
}

func (e *appEmitter) Emit(name string, data ...any) bool {
	return e.app.Event.Emit(name, data...)
}
