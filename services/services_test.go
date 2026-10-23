package services

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/catalog"
	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

type event struct {
	name string
	data any
}

type fakeEmitter struct {
	mu     sync.Mutex
	events []event
}

func (e *fakeEmitter) Emit(name string, data ...any) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	var d any
	if len(data) == 1 {
		d = data[0]
	}
	e.events = append(e.events, event{name, d})
	return false
}

func (e *fakeEmitter) take() []event {
	e.mu.Lock()
	defer e.mu.Unlock()
	ev := e.events
	e.events = nil
	return ev
}

func (e *fakeEmitter) names() []string {
	var names []string
	for _, ev := range e.take() {
		names = append(names, ev.name)
	}
	return names
}

func newTimer(t *testing.T, s config.TimerSettings) *timer.Timer {
	t.Helper()
	tm, err := timer.New(TimerConfig(s), timer.SystemClock())
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestTimerConfig(t *testing.T) {
	got := TimerConfig(config.TimerSettings{
		FocusMin: 50, ShortBreakMin: 10, LongBreakMin: 30, LongBreakEvery: 3, AutoStartFocus: true,
	})
	want := timer.Config{
		Focus: 50 * time.Minute, ShortBreak: 10 * time.Minute, LongBreak: 30 * time.Minute,
		LongBreakEvery: 3, AutoStartFocus: true,
	}
	if got != want {
		t.Fatalf("TimerConfig = %+v, want %+v", got, want)
	}
}

func TestTimerServiceForwardsEvents(t *testing.T) {
	emit := &fakeEmitter{}
	svc := NewTimerService(newTimer(t, config.Default().Timer), emit)

	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	ev := emit.take()
	if len(ev) != 2 || ev[0].name != EventTimerTransition || ev[1].name != EventTimerState {
		t.Fatalf("events = %+v, want transition then state", ev)
	}
	// Payloads must have exactly the registered types or Wails drops the event.
	tr, ok := ev[0].data.(timer.Transition)
	if !ok || tr.Cause != timer.CauseStart {
		t.Fatalf("transition payload = %#v", ev[0].data)
	}
	if st, ok := ev[1].data.(timer.State); !ok || st.Status != timer.Running {
		t.Fatalf("state payload = %#v", ev[1].data)
	}

	if err := svc.Start(); !errors.Is(err, timer.ErrInvalidCommand) {
		t.Fatalf("second Start() = %v, want ErrInvalidCommand", err)
	}
	if err := svc.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := svc.Resume(); err != nil {
		t.Fatal(err)
	}
	svc.Skip()
	if st := svc.State(); st.Phase != timer.ShortBreak || st.Status != timer.Idle {
		t.Fatalf("after Skip: %+v", st)
	}
	_ = svc.Start()
	emit.take()
	svc.ResetPhase()
	if st := svc.State(); st.Phase != timer.ShortBreak || st.Status != timer.Idle {
		t.Fatalf("after ResetPhase: %+v", st)
	}
	if ev := emit.take(); len(ev) != 2 || ev[0].data.(timer.Transition).Cause != timer.CauseResetPhase {
		t.Fatalf("ResetPhase events = %+v", ev)
	}
	svc.Reset()
	if st := svc.State(); st.Phase != timer.Focus || st.Status != timer.Idle {
		t.Fatalf("after Reset: %+v", st)
	}
}

func TestMusicServiceGenres(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	c := catalog.Default()
	if got := NewMusicService(c, settings).Genres(); !reflect.DeepEqual(got, c) {
		t.Fatalf("Genres() = %+v, want %+v", got, c)
	}
}

func TestMusicServiceSelect(t *testing.T) {
	settings, store, _, emit := newSettingsService(t)
	svc := NewMusicService(catalog.Default(), settings)

	if err := svc.Select("classical", 1); err != nil {
		t.Fatal(err)
	}
	saved, _ := store.Load()
	if saved.Music.GenreID != "classical" || saved.Music.SourceIndex != 1 {
		t.Fatalf("saved music = %+v", saved.Music)
	}
	if names := emit.names(); !reflect.DeepEqual(names, []string{EventSettingsChanged}) {
		t.Fatalf("events = %v", names)
	}

	for _, tc := range []struct {
		genre string
		index int
	}{{"nope", 0}, {"classical", 2}, {"classical", -1}} {
		if err := svc.Select(tc.genre, tc.index); err == nil {
			t.Errorf("Select(%q, %d) accepted", tc.genre, tc.index)
		}
	}
	if got := settings.Get().Music; got.GenreID != "classical" || got.SourceIndex != 1 {
		t.Fatalf("rejected Select changed settings: %+v", got)
	}
}

func TestMusicServiceSaveVolume(t *testing.T) {
	settings, store, _, _ := newSettingsService(t)
	svc := NewMusicService(catalog.Default(), settings)
	before := settings.Get()

	if err := svc.SaveVolume(15, true); err != nil {
		t.Fatal(err)
	}
	saved, _ := store.Load()
	if saved.Music.Volume != 15 || !saved.Music.Muted {
		t.Fatalf("saved music = %+v", saved.Music)
	}
	// Only the volume fields change.
	saved.Music.Volume, saved.Music.Muted = before.Music.Volume, before.Music.Muted
	if !reflect.DeepEqual(saved, before) {
		t.Fatalf("SaveVolume touched other settings: %+v", saved)
	}

	var ve *config.ValidationError
	if err := svc.SaveVolume(101, false); !errors.As(err, &ve) {
		t.Fatalf("SaveVolume(101) = %v, want ValidationError", err)
	}
}

func TestTimerServiceLifecycle(t *testing.T) {
	svc := NewTimerService(newTimer(t, config.Default().Timer), &fakeEmitter{})
	if err := svc.ServiceShutdown(); err != nil { // shutdown without startup is harmless
		t.Fatal(err)
	}
	if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := svc.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
}

func newSettingsService(t *testing.T) (*SettingsService, *config.Store, *timer.Timer, *fakeEmitter) {
	t.Helper()
	store := config.NewStore(t.TempDir(), slog.New(slog.DiscardHandler))
	initial := config.Default()
	emit := &fakeEmitter{}
	tm := newTimer(t, initial.Timer)
	NewTimerService(tm, emit) // forward timer events into the same emitter
	return NewSettingsService(store, initial, tm, emit), store, tm, emit
}

func TestSettingsUpdateSavesAppliesAndAnnounces(t *testing.T) {
	svc, store, tm, emit := newSettingsService(t)

	next := svc.Get()
	next.Timer.FocusMin = 50
	next.Music.Volume = 80
	got, err := svc.Update(next)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, next) || !reflect.DeepEqual(svc.Get(), next) {
		t.Fatalf("Update returned %+v, Get %+v, want %+v", got, svc.Get(), next)
	}

	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, next) {
		t.Fatalf("saved %+v, want %+v", saved, next)
	}

	if st := tm.State(); st.RemainingMs != (50 * time.Minute).Milliseconds() {
		t.Fatalf("timer not updated: %+v", st)
	}

	ev := emit.take()
	if len(ev) != 2 || ev[0].name != EventTimerState || ev[1].name != EventSettingsChanged {
		t.Fatalf("events = %+v, want timer:state then settings:changed", ev)
	}
	if s, ok := ev[1].data.(config.Settings); !ok || !reflect.DeepEqual(s, next) {
		t.Fatalf("settings payload = %#v", ev[1].data)
	}
}

func TestSettingsUpdateWithoutTimerChangeLeavesTimerAlone(t *testing.T) {
	svc, _, _, emit := newSettingsService(t)
	next := svc.Get()
	next.Notify.Sound = false
	if _, err := svc.Update(next); err != nil {
		t.Fatal(err)
	}
	if names := emit.names(); !reflect.DeepEqual(names, []string{EventSettingsChanged}) {
		t.Fatalf("events = %v, want only settings:changed", names)
	}
}

func TestSettingsUpdateRejectsInvalid(t *testing.T) {
	svc, store, tm, emit := newSettingsService(t)
	before := svc.Get()

	bad := before
	bad.Timer.FocusMin = 0
	bad.Music.Volume = 500
	got, err := svc.Update(bad)
	var ve *config.ValidationError
	if !errors.As(err, &ve) || len(ve.Problems) != 2 {
		t.Fatalf("Update() err = %v, want ValidationError with 2 problems", err)
	}
	if !reflect.DeepEqual(got, before) || !reflect.DeepEqual(svc.Get(), before) {
		t.Fatalf("settings changed after a rejected update: %+v", svc.Get())
	}
	if ev := emit.take(); len(ev) != 0 {
		t.Fatalf("emitted %+v", ev)
	}
	if st := tm.State(); st.RemainingMs != (25 * time.Minute).Milliseconds() {
		t.Fatalf("timer changed: %+v", st)
	}
	if loaded, _ := store.Load(); !reflect.DeepEqual(loaded, config.Default()) {
		t.Fatalf("file written: %+v", loaded)
	}
}

func TestSettingsConcurrentUpdatesStayConsistent(t *testing.T) {
	svc, store, tm, _ := newSettingsService(t)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			next := svc.Get()
			next.Timer.FocusMin = 10 + i
			next.Music.Volume = i
			if _, err := svc.Update(next); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	final := svc.Get()
	saved, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, final) {
		t.Fatalf("file %+v differs from memory %+v", saved, final)
	}
	if st := tm.State(); st.RemainingMs != (time.Duration(final.Timer.FocusMin) * time.Minute).Milliseconds() {
		t.Fatalf("timer has %dms, settings say %d min", st.RemainingMs, final.Timer.FocusMin)
	}
}
