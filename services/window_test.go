package services

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"focuswidget/internal/config"
)

type fakeWindow struct {
	mu          sync.Mutex
	bounds      application.Rect
	alwaysOnTop bool
	shown       bool
	listeners   map[events.WindowEventType][]func(*application.WindowEvent)
}

func (w *fakeWindow) fire(e events.WindowEventType) {
	for _, fn := range w.listeners[e] {
		fn(nil)
	}
}

func (w *fakeWindow) Bounds() application.Rect {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.bounds
}

func (w *fakeWindow) SetBounds(r application.Rect) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bounds = r
}

func (w *fakeWindow) SetAlwaysOnTop(on bool) application.Window {
	w.alwaysOnTop = on
	return nil
}

func (w *fakeWindow) Show() application.Window {
	w.shown = true
	return nil
}

func (w *fakeWindow) OnWindowEvent(e events.WindowEventType, fn func(*application.WindowEvent)) func() {
	if w.listeners == nil {
		w.listeners = map[events.WindowEventType][]func(*application.WindowEvent){}
	}
	w.listeners[e] = append(w.listeners[e], fn)
	return func() {}
}

func newWindowService(t *testing.T, screens ...*application.Screen) (*WindowService, *fakeWindow, *SettingsService, *fakeEmitter) {
	t.Helper()
	settings, _, _, emit := newSettingsService(t)
	win := &fakeWindow{bounds: application.Rect{X: 100, Y: 50, Width: ExpandedWidth, Height: ExpandedHeight}}
	svc := NewWindowService(settings, emit, func() Window { return win },
		func() []*application.Screen { return screens })
	return svc, win, settings, emit
}

func TestWindowSize(t *testing.T) {
	if w, h := WindowSize(config.WindowCompact); w != 300 || h != 96 {
		t.Fatalf("compact = %dx%d", w, h)
	}
	if w, h := WindowSize(config.WindowExpanded); w != 380 || h != 540 {
		t.Fatalf("expanded = %dx%d", w, h)
	}
}

func TestSetModeResizesAroundTopLeftSavesAndAnnounces(t *testing.T) {
	svc, win, settings, emit := newWindowService(t)

	if err := svc.SetMode(config.WindowCompact); err != nil {
		t.Fatal(err)
	}
	if want := (application.Rect{X: 100, Y: 50, Width: 300, Height: 96}); win.Bounds() != want {
		t.Fatalf("bounds = %+v, want %+v", win.Bounds(), want)
	}
	if svc.Mode() != config.WindowCompact || settings.Get().Window.Mode != config.WindowCompact {
		t.Fatal("mode not saved")
	}
	ev := emit.take()
	last := ev[len(ev)-1]
	if last.name != EventWindowMode || last.data != config.WindowCompact {
		t.Fatalf("events = %+v, want window:mode compact last", ev)
	}

	if err := svc.SetMode("fullscreen"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if win.Bounds().Width != 300 {
		t.Fatal("rejected mode resized the window")
	}
}

func TestSetAlwaysOnTop(t *testing.T) {
	svc, win, settings, _ := newWindowService(t)
	if err := svc.SetAlwaysOnTop(false); err != nil {
		t.Fatal(err)
	}
	if win.alwaysOnTop || settings.Get().Window.AlwaysOnTop {
		t.Fatal("always-on-top not switched off")
	}
}

func TestPlaceRestoresOrFallsBack(t *testing.T) {
	primary := &application.Screen{IsPrimary: true, WorkArea: application.Rect{X: 0, Y: 0, Width: 1920, Height: 1032}}

	t.Run("never moved: bottom-right of the primary work area", func(t *testing.T) {
		svc, win, _, _ := newWindowService(t, primary)
		svc.place()
		want := application.Rect{X: 1920 - 380 - 16, Y: 1032 - 540 - 16, Width: 380, Height: 540}
		if win.Bounds() != want {
			t.Fatalf("bounds = %+v, want %+v", win.Bounds(), want)
		}
	})

	t.Run("saved compact position on a screen is kept", func(t *testing.T) {
		svc, win, settings, _ := newWindowService(t, primary)
		_, _ = settings.Update(func() config.Settings {
			s := settings.Get()
			s.Window.Mode = config.WindowCompact
			s.Window.X, s.Window.Y = new(40), new(60)
			return s
		}())
		svc.place()
		if want := (application.Rect{X: 40, Y: 60, Width: 300, Height: 96}); win.Bounds() != want {
			t.Fatalf("bounds = %+v, want %+v", win.Bounds(), want)
		}
	})

	t.Run("saved position on an unplugged monitor falls back", func(t *testing.T) {
		svc, win, settings, _ := newWindowService(t, primary)
		_, _ = settings.Update(func() config.Settings {
			s := settings.Get()
			s.Window.X, s.Window.Y = new(-900), new(300)
			return s
		}())
		svc.place()
		if win.Bounds().X != 1920-380-16 {
			t.Fatalf("bounds = %+v", win.Bounds())
		}
	})

	t.Run("no screens reported: leave the window alone", func(t *testing.T) {
		svc, win, _, _ := newWindowService(t)
		before := win.Bounds()
		svc.place()
		if win.Bounds() != before {
			t.Fatal("moved without screen info")
		}
	})
}

func TestStartupShowsTheWindowInPlaceOnceThePageIsReady(t *testing.T) {
	primary := &application.Screen{IsPrimary: true, WorkArea: application.Rect{Width: 1920, Height: 1032}}
	svc, win, _, _ := newWindowService(t, primary)
	if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if win.shown {
		t.Fatal("shown before the page was ready")
	}

	win.fire(events.Common.WindowRuntimeReady)
	if !win.shown || win.Bounds().X != 1920-380-16 {
		t.Fatalf("shown=%v bounds=%+v", win.shown, win.Bounds())
	}

	// A page reload fires the event again: the window must stay where the user put it.
	win.SetBounds(application.Rect{X: 10, Y: 10, Width: 380, Height: 540})
	win.fire(events.Common.WindowRuntimeReady)
	if win.Bounds().X != 10 {
		t.Fatal("reload moved the window")
	}
}

func TestMovedSavesPositionOnceItRests(t *testing.T) {
	svc, win, settings, _ := newWindowService(t)
	for x := range 5 { // a drag: many move events
		win.SetBounds(application.Rect{X: 300 + x, Y: 200, Width: 380, Height: 540})
		svc.moved()
	}
	if settings.Get().Window.X != nil {
		t.Fatal("saved before the window came to rest")
	}
	time.Sleep(PositionSaveDelay + 300*time.Millisecond)
	got := settings.Get().Window
	if got.X == nil || got.Y == nil || !reflect.DeepEqual([]int{*got.X, *got.Y}, []int{304, 200}) {
		t.Fatalf("saved position = %v,%v", got.X, got.Y)
	}
	if err := svc.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
}
