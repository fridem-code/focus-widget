package services

import (
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/hotkey"
)

// fakeHotkeys stands in for hotkey.Watch: it records what is registered and
// refuses the combos in taken.
type fakeHotkeys struct {
	mu     sync.Mutex
	taken  map[hotkey.Combo]bool
	held   map[int]hotkey.Combo // the current registration
	fired  func(id int)
	starts int
}

func (f *fakeHotkeys) watch(combos map[int]hotkey.Combo, fired func(id int)) (func(), map[int]error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	failed := map[int]error{}
	f.held = map[int]hotkey.Combo{}
	for id, c := range combos {
		if f.taken[c] {
			failed[id] = hotkey.ErrTaken
			continue
		}
		f.held[id] = c
	}
	f.fired = fired
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.held = nil
	}, failed
}

// registered lists the held shortcuts as strings.
func (f *fakeHotkeys) registered() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.held {
		out = append(out, c.String())
	}
	slices.Sort(out)
	return out
}

// press fires the registered shortcut s, if it is.
func (f *fakeHotkeys) press(s string) {
	f.mu.Lock()
	var id int
	for i, c := range f.held {
		if c.String() == s {
			id = i
		}
	}
	fired := f.fired
	f.mu.Unlock()
	if id != 0 {
		fired(id)
	}
}

func newHotkeyService(t *testing.T, taken ...string) (*HotkeyService, *fakeHotkeys, *SettingsService, *fakeEmitter, *[]string) {
	t.Helper()
	settings, _, _, _ := newSettingsService(t)
	fake := &fakeHotkeys{taken: map[hotkey.Combo]bool{}}
	for _, s := range taken {
		c, err := hotkey.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		fake.taken[c] = true
	}
	var mu sync.Mutex
	var ran []string
	action := func(name string) func() {
		return func() { mu.Lock(); ran = append(ran, name); mu.Unlock() }
	}
	emit := &fakeEmitter{}
	h := NewHotkeyService(settings, emit, map[string]func(){
		"startPause": action("startPause"),
		"showHide":   action("showHide"),
		"nextTrack":  action("nextTrack"),
	}, fake.watch)
	if err := h.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.ServiceShutdown() })
	return h, fake, settings, emit, &ran
}

func TestHotkeysRegisterTheSettingsAndRunActions(t *testing.T) {
	_, fake, _, _, ran := newHotkeyService(t)
	want := []string{"Ctrl+Alt+Shift+F", "Ctrl+Alt+Shift+N", "Ctrl+Alt+Shift+P"}
	if got := fake.registered(); !slices.Equal(got, want) {
		t.Fatalf("registered %v, want %v", got, want)
	}
	fake.press("Ctrl+Alt+Shift+P")
	fake.press("Ctrl+Alt+Shift+N")
	if !slices.Equal(*ran, []string{"startPause", "nextTrack"}) {
		t.Fatalf("ran %v", *ran)
	}
}

func TestHotkeysReportTakenShortcuts(t *testing.T) {
	h, fake, _, emit, _ := newHotkeyService(t, "Ctrl+Alt+Shift+F")
	want := map[string]string{"showHide": "Занято другим приложением"}
	if got := h.Status(); !maps.Equal(got, want) {
		t.Fatalf("Status() = %v, want %v", got, want)
	}
	if got := fake.registered(); len(got) != 2 {
		t.Fatalf("the others must still work: registered %v", got)
	}
	evs := emit.take()
	if len(evs) != 1 || evs[0].name != EventHotkeysStatus || !maps.Equal(evs[0].data.(map[string]string), want) {
		t.Fatalf("events = %+v", evs)
	}
}

func TestHotkeysFollowSettingsChanges(t *testing.T) {
	h, fake, settings, _, _ := newHotkeyService(t)
	key, off := "Alt+F9", ""
	if _, err := settings.Patch(SettingsPatch{HotkeyStartPause: &key, HotkeyNextTrack: &off}); err != nil {
		t.Fatal(err)
	}
	want := []string{"Alt+F9", "Ctrl+Alt+Shift+F"}
	waitFor(t, func() bool { return slices.Equal(fake.registered(), want) })

	// Other settings leave the registration alone.
	starts := fake.starts
	goal := 3
	if _, err := settings.Patch(SettingsPatch{DailyGoal: &goal}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if fake.starts != starts {
		t.Fatal("re-registered for an unrelated change")
	}
	if len(h.Status()) != 0 {
		t.Fatalf("Status() = %v", h.Status())
	}
}

func TestHotkeysSuspendWhileRecording(t *testing.T) {
	h, fake, _, _, _ := newHotkeyService(t)
	h.Suspend(true)
	if got := fake.registered(); len(got) != 0 {
		t.Fatalf("suspended but registered %v", got)
	}
	h.Suspend(false)
	if got := fake.registered(); len(got) != 3 {
		t.Fatalf("resumed: registered %v", got)
	}
}

func TestSettingsRejectsOneShortcutForTwoActions(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	same := "alt+shift+ctrl+p"
	_, err := settings.Patch(SettingsPatch{HotkeyShowHide: &same})
	if err == nil {
		t.Fatal("saved one shortcut for two actions")
	}
	if settings.Get().Hotkeys.ShowHide != "Ctrl+Alt+Shift+F" {
		t.Fatal("the rejected shortcut was applied")
	}
}

func TestSettingsOnChangeSeesOldAndNew(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	var got []string
	settings.OnChange(func(old, next config.Settings) { got = append(got, old.Timer.Label+"→"+next.Timer.Label) })
	label := "Отчёт"
	if _, err := settings.Patch(SettingsPatch{Label: &label}); err != nil {
		t.Fatal(err)
	}
	bad := "a\nb"
	if _, err := settings.Patch(SettingsPatch{Label: &bad}); err == nil {
		t.Fatal("a two-line label was saved")
	}
	if !slices.Equal(got, []string{"→Отчёт"}) {
		t.Fatalf("OnChange saw %q", got)
	}
}

func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	for i := 0; i < 200 && !done(); i++ {
		time.Sleep(5 * time.Millisecond)
	}
	if !done() {
		t.Fatal("timed out")
	}
}
