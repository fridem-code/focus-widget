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
	focused     int
	quits       int
	listeners   map[events.WindowEventType][]func(*application.WindowEvent)
	hooks       map[events.WindowEventType][]func(*application.WindowEvent)
}

func (w *fakeWindow) fire(e events.WindowEventType) {
	for _, fn := range w.listeners[e] {
		fn(nil)
	}
}

// hook runs the hooks for e and reports whether one of them cancelled it.
func (w *fakeWindow) hook(e events.WindowEventType) (cancelled bool) {
	ev := application.NewWindowEvent()
	for _, fn := range w.hooks[e] {
		fn(ev)
	}
	return ev.IsCancelled()
}

func (w *fakeWindow) Hide() application.Window {
	w.shown = false
	return nil
}

func (w *fakeWindow) Focus() { w.focused++ }

func (w *fakeWindow) RegisterHook(e events.WindowEventType, fn func(*application.WindowEvent)) func() {
	if w.hooks == nil {
		w.hooks = map[events.WindowEventType][]func(*application.WindowEvent){}
	}
	w.hooks[e] = append(w.hooks[e], fn)
	return func() {}
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
		func() []*application.Screen { return screens }, func() { win.quits++ }, &fakeGlass{on: true}, nil)
	return svc, win, settings, emit
}

func TestHideShowAnnounceVisibility(t *testing.T) {
	svc, win, _, emit := newWindowService(t)
	svc.Show()
	if !win.shown || win.focused != 1 || !svc.Visible() {
		t.Fatalf("after Show: shown=%v focused=%d visible=%v", win.shown, win.focused, svc.Visible())
	}
	svc.ToggleVisible()
	if win.shown || svc.Visible() {
		t.Fatal("ToggleVisible did not hide")
	}
	var vis []any
	for _, e := range emit.take() {
		if e.name == EventWindowVisible {
			vis = append(vis, e.data)
		}
	}
	if !reflect.DeepEqual(vis, []any{true, false}) {
		t.Fatalf("window:visible events = %v", vis)
	}
}

func TestClosingHidesUntilQuit(t *testing.T) {
	svc, win, _, _ := newWindowService(t)
	if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	svc.Show()
	if cancelled := win.hook(events.Common.WindowClosing); !cancelled || win.shown {
		t.Fatalf("close: cancelled=%v shown=%v, want hidden instead of closed", cancelled, win.shown)
	}
	svc.Quit()
	if win.quits != 1 {
		t.Fatal("Quit did not quit the app")
	}
	if cancelled := win.hook(events.Common.WindowClosing); cancelled {
		t.Fatal("closing was blocked while quitting")
	}
}

func TestWindowSize(t *testing.T) {
	if w, h := WindowSize(config.WindowCompact); w != 300 || h != 96 {
		t.Fatalf("compact = %dx%d", w, h)
	}
	if w, h := WindowSize(config.WindowExpanded); w != 380 || h != 540 {
		t.Fatalf("expanded = %dx%d", w, h)
	}
}

// B4: a compact widget at the bottom of the screen expands upwards, stays on
// the screen, and collapses back to where it was.
func TestSetModeKeepsTheWindowOnScreen(t *testing.T) {
	screen := &application.Screen{IsPrimary: true, WorkArea: application.Rect{X: 0, Y: 0, Width: 1536, Height: 912}}
	svc, win, settings, _ := newWindowService(t, screen)
	win.bounds = application.Rect{X: 3, Y: 813, Width: CompactWidth, Height: CompactHeight}

	if err := svc.SetMode(config.WindowExpanded); err != nil {
		t.Fatal(err)
	}
	if want := (application.Rect{X: 3, Y: 813 + 96 - 540, Width: 380, Height: 540}); win.Bounds() != want {
		t.Fatalf("expanded bounds = %+v, want %+v", win.Bounds(), want)
	}
	if s := settings.Get().Window; *s.X != 3 || *s.Y != 813+96-540 {
		t.Fatalf("saved position = (%d, %d)", *s.X, *s.Y)
	}

	if err := svc.SetMode(config.WindowCompact); err != nil {
		t.Fatal(err)
	}
	if want := (application.Rect{X: 3, Y: 813, Width: 300, Height: 96}); win.Bounds() != want {
		t.Fatalf("compact bounds = %+v, want %+v", win.Bounds(), want)
	}
}

// Without screen info the window resizes around its top-left corner.
func TestSetModeResizesSavesAndAnnounces(t *testing.T) {
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

type fakeGlass struct {
	on         bool
	systemDark bool
	applied    []bool // dark per Apply call
}

func (g *fakeGlass) Apply(dark bool) bool {
	g.applied = append(g.applied, dark)
	return g.on
}

func (g *fakeGlass) SystemDark() bool { return g.systemDark }

// B1: the glass goes on before the window shows, for the theme the page will
// show ("system" follows Windows), and the page can switch it with its theme.
func TestGlassGoesOnBeforeTheWindowShows(t *testing.T) {
	svc, win, settings, _ := newWindowService(t)
	glass := svc.glass.(*fakeGlass)
	glass.systemDark = true
	if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(glass.applied) != 0 {
		t.Fatal("glass applied before the page was ready")
	}
	win.fire(events.Common.WindowRuntimeReady)
	if !reflect.DeepEqual(glass.applied, []bool{true}) || !win.shown {
		t.Fatalf("applied = %v, shown = %v; want dark glass (system theme) and a shown window", glass.applied, win.shown)
	}

	if _, err := settings.Patch(SettingsPatch{Theme: new(config.ThemeLight)}); err != nil {
		t.Fatal(err)
	}
	if svc.themeDark() {
		t.Fatal("light theme setting resolved as dark")
	}
	if on := svc.SetGlass(false); !on || glass.applied[1] {
		t.Fatalf("SetGlass(false) = %v, applied = %v", on, glass.applied)
	}
}

// T19: started by Windows, the window shows without taking the focus; later
// shows (from the tray) focus it as usual.
func TestQuietStartDoesNotTakeTheFocus(t *testing.T) {
	quiet := 0
	settings, _, _, emit := newSettingsService(t)
	win := &fakeWindow{}
	svc := NewWindowService(settings, emit, func() Window { return win },
		func() []*application.Screen { return nil }, func() {}, &fakeGlass{},
		func(show func()) { quiet++; show() })
	if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	win.fire(events.Common.WindowRuntimeReady)
	if quiet != 1 || !win.shown || win.focused != 0 || !svc.Visible() {
		t.Fatalf("quiet start: quiet=%d shown=%v focused=%d visible=%v", quiet, win.shown, win.focused, svc.Visible())
	}
	svc.Hide()
	svc.Show()
	if win.focused != 1 {
		t.Fatalf("Show from the tray focused %d times, want 1", win.focused)
	}
}
