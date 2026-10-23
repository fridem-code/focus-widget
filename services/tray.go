package services

import (
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

// TrayItem is the part of *application.MenuItem the tray relabels.
type TrayItem interface {
	SetLabel(string) *application.MenuItem
	SetChecked(bool) *application.MenuItem
}

// TrayUI is the tray icon and its menu.
type TrayUI interface {
	SetTooltip(string)
	UpdateMenu()
}

// TrayItems are the menu items whose label or check mark follows the app state.
type TrayItems struct {
	Visibility  TrayItem // Показать / Скрыть
	Timer       TrayItem // Старт / Пауза / Продолжить
	Mode        TrayItem // Компактный режим / Развернуть
	AlwaysOnTop TrayItem // checkbox
}

// TrayView is everything the tray shows.
type TrayView struct {
	Tooltip     string
	Visibility  string
	Timer       string
	Mode        string
	AlwaysOnTop bool
}

// Tray drives the system tray (docs/SPEC.md §3.5): it keeps the tooltip and
// menu in step with the app and carries out the menu commands. It is not a
// Wails service, so none of it is exposed to the frontend.
type Tray struct {
	timer    *timer.Timer
	settings *SettingsService
	window   *WindowService
	emit     Emitter

	mu    sync.Mutex
	ui    TrayUI
	items TrayItems
	last  TrayView
}

func NewTray(t *timer.Timer, settings *SettingsService, window *WindowService, emit Emitter) *Tray {
	return &Tray{timer: t, settings: settings, window: window, emit: emit}
}

// Attach connects the built tray menu and shows the current state.
func (t *Tray) Attach(ui TrayUI, items TrayItems) {
	t.mu.Lock()
	t.ui, t.items, t.last = ui, items, TrayView{}
	t.mu.Unlock()
	t.Refresh()
}

// TrayEmitter wraps an Emitter so the tray follows every event it sends.
type TrayEmitter struct {
	Emitter
	Tray *Tray
}

func (e TrayEmitter) Emit(name string, data ...any) bool {
	cancelled := e.Emitter.Emit(name, data...)
	if e.Tray != nil {
		e.Tray.OnEvent()
	}
	return cancelled
}

// OnEvent is called for every app event. Events are emitted while their
// sender may hold a lock (SettingsService.modify does), and Refresh reads
// those senders, so it runs on its own goroutine.
func (t *Tray) OnEvent() { go t.Refresh() }

// Refresh updates whatever changed, so the tooltip (whole minutes) changes
// once a minute and on transitions. The state is read under t.mu so that
// concurrent refreshes apply in order and the newest state wins.
func (t *Tray) Refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ui == nil {
		return
	}
	v := BuildTrayView(t.timer.State(), t.settings.Get(), t.window.Visible())
	if v == t.last {
		return
	}
	if v.Tooltip != t.last.Tooltip {
		t.ui.SetTooltip(v.Tooltip)
	}
	if v.Visibility != t.last.Visibility || v.Timer != t.last.Timer || v.Mode != t.last.Mode ||
		v.AlwaysOnTop != t.last.AlwaysOnTop || t.last == (TrayView{}) {
		t.items.Visibility.SetLabel(v.Visibility)
		t.items.Timer.SetLabel(v.Timer)
		t.items.Mode.SetLabel(v.Mode)
		t.items.AlwaysOnTop.SetChecked(v.AlwaysOnTop)
		t.ui.UpdateMenu()
	}
	t.last = v
}

// BuildTrayView computes the tray texts.
func BuildTrayView(st timer.State, s config.Settings, visible bool) TrayView {
	v := TrayView{
		Tooltip:     "Focus Widget · " + trayStatus(st),
		Visibility:  "Показать",
		Mode:        "Компактный режим",
		AlwaysOnTop: s.Window.AlwaysOnTop,
	}
	if visible {
		v.Visibility = "Скрыть"
	}
	if s.Window.Mode == config.WindowCompact {
		v.Mode = "Развернуть"
	}
	switch st.Status {
	case timer.Running:
		v.Timer = "Пауза"
	case timer.Paused:
		v.Timer = "Продолжить"
	default:
		v.Timer = "Старт"
	}
	return v
}

func trayStatus(st timer.State) string {
	minutes := (st.RemainingMs + 59_999) / 60_000
	phase := map[timer.Phase]string{
		timer.Focus:      "фокус",
		timer.ShortBreak: "короткий перерыв",
		timer.LongBreak:  "длинный перерыв",
	}[st.Phase]
	switch st.Status {
	case timer.Running:
		return fmt.Sprintf("%s, осталось %d мин", phase, minutes)
	case timer.Paused:
		return fmt.Sprintf("пауза, %s, осталось %d мин", phase, minutes)
	default:
		if st.Phase == timer.Focus {
			return fmt.Sprintf("готов к фокусу, %d мин", minutes)
		}
		return fmt.Sprintf("%s ждёт старта, %d мин", phase, minutes)
	}
}

// Menu commands.

func (t *Tray) ToggleVisible() { t.window.ToggleVisible() }

// ToggleTimer starts, pauses or resumes, like the main button.
func (t *Tray) ToggleTimer() {
	switch t.timer.State().Status {
	case timer.Running:
		_ = t.timer.Pause()
	case timer.Paused:
		_ = t.timer.Resume()
	default:
		_ = t.timer.Start()
	}
}

func (t *Tray) Skip()       { t.timer.Skip() }
func (t *Tray) ResetCycle() { t.timer.Reset() }

// NextGenre and ToggleMode are carried out by the frontend, which owns the
// player and the order of pausing and resizing.
func (t *Tray) NextGenre() { t.emit.Emit(EventUICommand, UINextGenre) }

func (t *Tray) ToggleMode() {
	if !t.window.Visible() {
		t.window.Show()
	}
	if t.settings.Get().Window.Mode == config.WindowCompact {
		t.emit.Emit(EventUICommand, UIExpand)
	} else {
		t.emit.Emit(EventUICommand, UICompact)
	}
}

func (t *Tray) ToggleAlwaysOnTop() {
	_ = t.window.SetAlwaysOnTop(!t.settings.Get().Window.AlwaysOnTop)
	t.Refresh()
}

func (t *Tray) Quit() { t.window.Quit() }
