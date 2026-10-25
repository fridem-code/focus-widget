package services

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

// The fakes are locked: the tray refreshes from its own goroutines.
type fakeItem struct {
	mu      sync.Mutex
	label   string
	checked bool
}

func (i *fakeItem) SetLabel(s string) *application.MenuItem {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.label = s
	return nil
}

func (i *fakeItem) SetChecked(c bool) *application.MenuItem {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.checked = c
	return nil
}

func (i *fakeItem) isChecked() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.checked
}

type fakeTrayUI struct {
	mu       sync.Mutex
	tooltips []string
	updates  int
}

func (u *fakeTrayUI) SetTooltip(s string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.tooltips = append(u.tooltips, s)
}

func (u *fakeTrayUI) UpdateMenu() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.updates++
}

type trayFixture struct {
	tray     *Tray
	ui       *fakeTrayUI
	items    TrayItems
	vis      *fakeItem
	timerIt  *fakeItem
	mode     *fakeItem
	onTop    *fakeItem
	win      *fakeWindow
	window   *WindowService
	settings *SettingsService
	timer    *timer.Timer
	emit     *fakeEmitter
}

func newTray(t *testing.T) *trayFixture {
	t.Helper()
	window, win, settings, emit := newWindowService(t)
	tm := settings.timer
	f := &trayFixture{
		ui: &fakeTrayUI{}, vis: &fakeItem{}, timerIt: &fakeItem{}, mode: &fakeItem{}, onTop: &fakeItem{},
		win: win, window: window, settings: settings, timer: tm, emit: emit,
	}
	f.items = TrayItems{Visibility: f.vis, Timer: f.timerIt, Mode: f.mode, AlwaysOnTop: f.onTop}
	f.tray = NewTray(tm, settings, window, emit)
	window.Show()
	f.tray.Attach(f.ui, f.items)
	return f
}

func TestBuildTrayView(t *testing.T) {
	min := func(m int64) int64 { return m * 60_000 }
	st := func(p timer.Phase, s timer.Status, rem int64) timer.State {
		return timer.State{Phase: p, Status: s, RemainingMs: rem, DurationMs: min(25)}
	}
	s := config.Default()
	tests := []struct {
		state   timer.State
		tooltip string
		label   string
	}{
		{st(timer.Focus, timer.Idle, min(25)), "Focus Widget · готов к фокусу, 25 мин", "Старт"},
		{st(timer.Focus, timer.Running, min(18)+1), "Focus Widget · фокус, осталось 19 мин", "Пауза"},
		{st(timer.Focus, timer.Paused, min(18)), "Focus Widget · пауза, фокус, осталось 18 мин", "Продолжить"},
		{st(timer.ShortBreak, timer.Running, 30_000), "Focus Widget · короткий перерыв, осталось 1 мин", "Пауза"},
		{st(timer.LongBreak, timer.Idle, min(15)), "Focus Widget · длинный перерыв ждёт старта, 15 мин", "Старт"},
	}
	for _, tt := range tests {
		v := BuildTrayView(tt.state, s, true)
		if v.Tooltip != tt.tooltip || v.Timer != tt.label {
			t.Errorf("%+v: got %q / %q, want %q / %q", tt.state, v.Tooltip, v.Timer, tt.tooltip, tt.label)
		}
	}

	s.Window.Mode = config.WindowCompact
	s.Window.AlwaysOnTop = false
	v := BuildTrayView(tests[0].state, s, false)
	if v.Visibility != "Показать" || v.Mode != "Развернуть" || v.AlwaysOnTop {
		t.Fatalf("view = %+v", v)
	}
	s.Window.Mode = config.WindowExpanded
	if v := BuildTrayView(tests[0].state, s, true); v.Visibility != "Скрыть" || v.Mode != "Компактный режим" {
		t.Fatalf("view = %+v", v)
	}
}

func TestTrayAttachShowsStateAndRefreshOnlyOnChange(t *testing.T) {
	f := newTray(t)
	if f.vis.label != "Скрыть" || f.timerIt.label != "Старт" || f.mode.label != "Компактный режим" || !f.onTop.checked {
		t.Fatalf("items after attach: %+v %+v %+v %+v", f.vis, f.timerIt, f.mode, f.onTop)
	}
	if len(f.ui.tooltips) != 1 || f.ui.updates != 1 {
		t.Fatalf("tooltips=%v updates=%d", f.ui.tooltips, f.ui.updates)
	}

	f.tray.Refresh() // nothing changed
	if len(f.ui.tooltips) != 1 || f.ui.updates != 1 {
		t.Fatal("refresh without changes touched the tray")
	}

	f.tray.ToggleTimer() // start
	f.tray.Refresh()
	if f.timerIt.label != "Пауза" || f.ui.updates != 2 {
		t.Fatalf("after start: label=%q updates=%d", f.timerIt.label, f.ui.updates)
	}
	if f.ui.tooltips[len(f.ui.tooltips)-1] != "Focus Widget · фокус, осталось 25 мин" {
		t.Fatalf("tooltip = %q", f.ui.tooltips)
	}
}

// The app wires the tray to every emitted event. SettingsService emits while
// holding its lock and the tray reads settings: this must not deadlock.
func TestTrayFollowsEventsWithoutDeadlock(t *testing.T) {
	store := config.NewStore(storeDir(t), slog.New(slog.DiscardHandler))
	tm := newTimer(t, config.Default().Timer)
	emitter := &TrayEmitter{Emitter: &fakeEmitter{}}
	settings := NewSettingsService(store, config.Default(), tm, emitter)
	win := &fakeWindow{}
	window := NewWindowService(settings, emitter, func() Window { return win },
		func() []*application.Screen { return nil }, func() {}, &fakeGlass{})
	tray := NewTray(tm, settings, window, emitter)
	emitter.Tray = tray
	ui, onTop := &fakeTrayUI{}, &fakeItem{}
	tray.Attach(ui, TrayItems{Visibility: &fakeItem{}, Timer: &fakeItem{}, Mode: &fakeItem{}, AlwaysOnTop: onTop})

	done := make(chan struct{})
	go func() {
		_ = window.SetAlwaysOnTop(false) // emits settings:changed under the settings lock
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock: settings change with the tray attached")
	}
	for i := 0; i < 100 && onTop.isChecked(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if onTop.isChecked() {
		t.Fatal("tray did not follow the settings change")
	}
}

func TestTrayTimerCommands(t *testing.T) {
	f := newTray(t)
	f.tray.ToggleTimer()
	if f.timer.State().Status != timer.Running {
		t.Fatal("not started")
	}
	f.tray.ToggleTimer()
	if f.timer.State().Status != timer.Paused {
		t.Fatal("not paused")
	}
	f.tray.ToggleTimer()
	if f.timer.State().Status != timer.Running {
		t.Fatal("not resumed")
	}
	f.tray.Skip()
	if f.timer.State().Phase != timer.ShortBreak {
		t.Fatal("not skipped")
	}
	f.tray.ResetCycle()
	if st := f.timer.State(); st.Phase != timer.Focus || st.Status != timer.Idle {
		t.Fatalf("not reset: %+v", st)
	}
}

func TestTrayWindowCommands(t *testing.T) {
	f := newTray(t)
	f.tray.ToggleVisible()
	f.tray.Refresh()
	if f.window.Visible() || f.vis.label != "Показать" {
		t.Fatalf("hide: visible=%v label=%q", f.window.Visible(), f.vis.label)
	}

	f.emit.take()
	f.tray.ToggleMode() // hidden + expanded: show, then ask the frontend to compact
	ev := f.emit.take()
	last := ev[len(ev)-1]
	if !f.window.Visible() || last.name != EventUICommand || last.data != UICompact {
		t.Fatalf("toggle mode: visible=%v events=%+v", f.window.Visible(), ev)
	}

	f.tray.NextGenre()
	if ev := f.emit.take(); ev[0].data != UINextGenre {
		t.Fatalf("next genre events = %+v", ev)
	}

	f.tray.ToggleAlwaysOnTop()
	if f.win.alwaysOnTop || f.settings.Get().Window.AlwaysOnTop || f.onTop.checked {
		t.Fatal("always-on-top not switched off everywhere")
	}

	f.tray.Quit()
	if f.win.quits != 1 {
		t.Fatal("quit")
	}
}
