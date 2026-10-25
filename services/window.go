package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"focuswidget/internal/config"
	"focuswidget/internal/placement"
)

// Window sizes per mode, in DIPs (docs/SPEC.md §3.4).
const (
	ExpandedWidth  = 380
	ExpandedHeight = 540
	CompactWidth   = 300
	CompactHeight  = 96
)

// PositionSaveDelay: moving the window fires many events; save once it rests.
const PositionSaveDelay = 500 * time.Millisecond

// WindowSize returns the window size for a mode.
func WindowSize(mode config.WindowMode) (width, height int) {
	if mode == config.WindowCompact {
		return CompactWidth, CompactHeight
	}
	return ExpandedWidth, ExpandedHeight
}

// Window is the part of *application.WebviewWindow the service uses.
type Window interface {
	Bounds() application.Rect
	SetBounds(application.Rect)
	SetAlwaysOnTop(bool) application.Window
	Show() application.Window
	Hide() application.Window
	Focus()
	OnWindowEvent(events.WindowEventType, func(*application.WindowEvent)) func()
	RegisterHook(events.WindowEventType, func(*application.WindowEvent)) func()
}

// WindowService switches between expanded and compact mode, toggles
// always-on-top, hides to and shows from the tray, and remembers and
// restores the window position.
type WindowService struct {
	settings *SettingsService
	emit     Emitter
	window   func() Window // the window is created after the services
	screens  func() []*application.Screen
	quit     func()

	mu        sync.Mutex
	saveTimer *time.Timer
	visible   bool
	quitting  bool
}

func NewWindowService(settings *SettingsService, emit Emitter, window func() Window,
	screens func() []*application.Screen, quit func()) *WindowService {
	return &WindowService{settings: settings, emit: emit, window: window, screens: screens, quit: quit}
}

// ServiceStartup remembers the position after every move, turns closing the
// window into hiding it, and, once the page has loaded, puts the (hidden)
// window in place and shows it.
//
// Wails creates native windows asynchronously, so ApplicationStarted can
// arrive before the window exists and SetBounds/Show would silently do
// nothing; WindowRuntimeReady only fires for a window that exists. Once: a
// page reload must not move the window back.
func (s *WindowService) ServiceStartup(context.Context, application.ServiceOptions) error {
	win := s.window()
	win.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) { s.moved() })
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		s.mu.Lock()
		quitting := s.quitting
		s.mu.Unlock()
		if !quitting { // Alt+F4 and the like hide to the tray; only Quit exits (SPEC §3.4)
			e.Cancel()
			s.Hide()
		}
	})
	var shown sync.Once
	win.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		shown.Do(func() {
			s.place()
			s.Show()
		})
	})
	return nil
}

// Visible reports whether the window is shown (not hidden in the tray).
func (s *WindowService) Visible() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.visible
}

// Hide puts the window into the tray. window:visible=false goes out first so
// the frontend pauses the music before the player disappears (SPEC §2).
func (s *WindowService) Hide() {
	s.setVisible(false)
	s.window().Hide()
}

// Show brings the window back from the tray and focuses it.
func (s *WindowService) Show() {
	win := s.window()
	win.Show()
	win.Focus()
	s.setVisible(true)
}

func (s *WindowService) ToggleVisible() {
	if s.Visible() {
		s.Hide()
	} else {
		s.Show()
	}
}

// Quit exits the app; it is the only way out (closing just hides).
func (s *WindowService) Quit() {
	s.mu.Lock()
	s.quitting = true
	s.mu.Unlock()
	s.quit()
}

func (s *WindowService) setVisible(v bool) {
	s.mu.Lock()
	s.visible = v
	s.mu.Unlock()
	s.emit.Emit(EventWindowVisible, v)
}

func (s *WindowService) ServiceShutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}
	return nil
}

func (s *WindowService) Mode() config.WindowMode { return s.settings.Get().Window.Mode }

// SetMode resizes the window, keeping the edges nearest to its screen's edges
// in place and the whole window on the screen (placement.Resize: a widget
// expanded from the bottom of the screen used to grow off it, B4), remembers
// the mode and position, and announces the mode as window:mode (the music
// policy reacts to it).
func (s *WindowService) SetMode(mode config.WindowMode) error {
	if mode != config.WindowExpanded && mode != config.WindowCompact {
		return fmt.Errorf("unknown window mode %q", mode)
	}
	w, h := WindowSize(mode)
	win := s.window()
	areas, _ := s.workAreas()
	x, y := placement.Resize(rect(win.Bounds()), w, h, areas)
	win.SetBounds(application.Rect{X: x, Y: y, Width: w, Height: h})
	if _, err := s.settings.modify(func(cur *config.Settings) {
		cur.Window.Mode = mode
		cur.Window.X, cur.Window.Y = &x, &y
	}); err != nil {
		return err
	}
	s.emit.Emit(EventWindowMode, mode)
	return nil
}

// SetAlwaysOnTop pins the window above others (or not) and remembers it.
func (s *WindowService) SetAlwaysOnTop(on bool) error {
	s.window().SetAlwaysOnTop(on)
	_, err := s.settings.modify(func(cur *config.Settings) { cur.Window.AlwaysOnTop = on })
	return err
}

// place puts the window at its saved position, or the default corner if that
// position is no longer on any screen (SPEC §3.4).
func (s *WindowService) place() {
	areas, primary := s.workAreas()
	if len(areas) == 0 {
		return
	}
	cur := s.settings.Get().Window
	var saved *[2]int
	if cur.X != nil && cur.Y != nil {
		saved = &[2]int{*cur.X, *cur.Y}
	}
	w, h := WindowSize(cur.Mode)
	x, y := placement.Place(saved, w, h, areas, primary)
	s.window().SetBounds(application.Rect{X: x, Y: y, Width: w, Height: h})
}

// workAreas returns the usable areas of all screens and the primary one's;
// none if Wails reports no screens.
func (s *WindowService) workAreas() (areas []placement.Rect, primary placement.Rect) {
	screens := s.screens()
	for i, sc := range screens {
		areas = append(areas, rect(sc.WorkArea))
		if i == 0 || sc.IsPrimary {
			primary = rect(sc.WorkArea)
		}
	}
	return areas, primary
}

// moved saves the position once the window has stopped moving.
func (s *WindowService) moved() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}
	s.saveTimer = time.AfterFunc(PositionSaveDelay, s.savePosition)
}

func (s *WindowService) savePosition() {
	b := s.window().Bounds()
	_, _ = s.settings.modify(func(cur *config.Settings) {
		cur.Window.X, cur.Window.Y = &b.X, &b.Y
	})
}

func rect(r application.Rect) placement.Rect {
	return placement.Rect{X: r.X, Y: r.Y, W: r.Width, H: r.Height}
}
