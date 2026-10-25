// Package app assembles the Wails application: options, services, the widget
// window, the tray and the log (see docs/SPEC.md §3.4–3.5, §4, §5).
package app

import (
	"fmt"
	"io/fs"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"focuswidget/internal/catalog"
	"focuswidget/internal/config"
	"focuswidget/internal/foreground"
	"focuswidget/internal/timer"
	"focuswidget/services"
)

const (
	name        = "Focus Widget"
	description = "Pomodoro timer widget with YouTube focus music"
	// Keep in step with build/config.yml.
	version = "0.1.0"
)

// New builds the application with its services and main widget window. assets
// must contain the built frontend (frontend/dist).
func New(assets fs.FS) (*application.App, error) {
	dir, err := config.DefaultDir()
	if err != nil {
		return nil, err
	}
	logger, wailsLogger := newLoggers(dir)
	logger.Info("start", "version", version)
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
	// The tray follows every event too (it is attached below).
	wails := &wailsEmitter{}
	emitter := &services.TrayEmitter{Emitter: wails}
	settingsService := services.NewSettingsService(store, settings, t, emitter)
	var wailsApp *application.App
	var mainWindow *application.WebviewWindow
	// Registers the app with Windows (HKCU AppUserModelId + COM activator) on
	// startup; T13's uninstaller removes those keys (docs/spike-results.md §6).
	toasts := notifications.New()
	windowService := services.NewWindowService(settingsService, emitter,
		func() services.Window { return mainWindow },
		func() []*application.Screen { return wailsApp.Screen.GetAll() },
		func() { wailsApp.Quit() },
		windowGlass{window: func() *application.WebviewWindow { return mainWindow }, logger: logger})
	wailsApp = application.New(application.Options{
		Name:         name,
		Description:  description,
		Logger:       wailsLogger,
		ErrorHandler: wailsErrorHandler(wailsLogger),
		// A second launch exits and brings the running widget's window back (SPEC §3.9).
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:               "com.focuswidget.app",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) { windowService.Show() },
		},
		Services: []application.Service{
			application.NewService(services.NewTimerService(t, emitter)),
			application.NewService(settingsService),
			application.NewService(services.NewMusicService(genres, settingsService)),
			application.NewService(windowService),
			application.NewService(toasts),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			// Without this flag WebView2 blocks unmuted playVideo() when the page has
			// not been clicked (tray "Start", phase auto-start). See docs/spike-results.md §3.
			AdditionalBrowserArgs: append([]string{"--autoplay-policy=no-user-gesture-required"}, devBrowserArgs()...),
		},
		// A tray app: closing the window never ends it, only the tray's "Выход".
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	wails.app = wailsApp

	width, height := services.WindowSize(settings.Window.Mode)
	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "main",
		Title:  name,
		Width:  width,
		Height: height,
		// Shown by WindowService once it is at its saved place, so it never
		// flashes in the centre first.
		Hidden:        true,
		Frameless:     true,
		AlwaysOnTop:   settings.Window.AlwaysOnTop,
		DisableResize: true,
		// Glass: a transparent webview over the accent-policy blur that
		// internal/glass turns on, tinted by CSS. The system Acrylic backdrop is
		// off: Windows replaced it with a solid fill whenever the window was
		// inactive (docs/BUGS.md B1, docs/spike-results.md T16). The accent only
		// blurs with zero DWM frame margins, so Wails must not extend the frame
		// on every activation, as it does for frameless windows with decorations.
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows: application.WindowsWindow{
			BackdropType:                      application.None,
			DisableFramelessWindowDecorations: true,
		},
		URL: "/",
	})

	// Some apps open over always-on-top windows (Paint, docs/BUGS.md B2): pin
	// the widget again whenever another app comes to the foreground.
	keeper := services.NewTopmostKeeper(settingsService, windowService, func() {
		if hwnd := mainWindow.NativeWindow(); hwnd != nil {
			if err := foreground.Pin(uintptr(hwnd)); err != nil {
				logger.Warn("pin the window on top", "err", err)
			}
		}
	})
	if stop, err := foreground.Watch(keeper.ForegroundChanged); err != nil {
		logger.Warn("watch foreground windows", "err", err)
	} else {
		wailsApp.OnShutdown(stop)
	}

	tray := services.NewTray(t, settingsService, windowService, emitter)
	buildTray(wailsApp, tray)
	emitter.Tray = tray

	// Toast when a phase runs out (the chime is the frontend's); clicking the
	// toast brings the window back.
	services.NewPhaseNotifier(t, toasts, settingsService, func(err error) {
		logger.Warn("phase-end toast failed", "err", err)
	})
	toasts.OnNotificationResponse(func(notifications.NotificationResult) { windowService.Show() })

	return wailsApp, nil
}

// wailsEmitter sends events to the frontend.
type wailsEmitter struct {
	app *application.App
}

func (e *wailsEmitter) Emit(name string, data ...any) bool {
	return e.app.Event.Emit(name, data...)
}
