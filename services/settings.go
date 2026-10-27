package services

import (
	"sync"
	"sync/atomic"

	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

// SettingsService owns the in-memory settings: it validates and persists every
// change, applies timer settings to the running timer and announces the new
// settings as a settings:changed event.
type SettingsService struct {
	store *config.Store
	timer *timer.Timer
	emit  Emitter

	// mu orders changes. Reads need no lock: the current settings are an
	// immutable snapshot swapped atomically, so timer listeners may read them
	// even while a change waits for the timer (see modify).
	mu        sync.Mutex
	current   atomic.Pointer[config.Settings]
	listeners []func(old, next config.Settings)
}

// NewSettingsService starts from initial, which must already be valid (as
// returned by config.Store.Load).
func NewSettingsService(store *config.Store, initial config.Settings, t *timer.Timer, emit Emitter) *SettingsService {
	s := &SettingsService{store: store, timer: t, emit: emit}
	s.current.Store(&initial)
	return s
}

func (s *SettingsService) Get() config.Settings {
	return *s.current.Load()
}

// OnChange registers fn for every committed change. It runs while the change
// still holds the lock, so it must not change settings itself.
func (s *SettingsService) OnChange(fn func(old, next config.Settings)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// Update replaces all settings. Invalid settings are rejected with an error
// listing every bad field and nothing is saved or applied.
func (s *SettingsService) Update(next config.Settings) (config.Settings, error) {
	return s.modify(func(cur *config.Settings) { *cur = next })
}

// SettingsPatch holds the fields the settings screen edits. Nil fields stay as
// they are.
type SettingsPatch struct {
	FocusMin        *int                  `json:"focusMin,omitempty"`
	ShortBreakMin   *int                  `json:"shortBreakMin,omitempty"`
	LongBreakMin    *int                  `json:"longBreakMin,omitempty"`
	LongBreakEvery  *int                  `json:"longBreakEvery,omitempty"`
	AutoStartBreaks *bool                 `json:"autoStartBreaks,omitempty"`
	AutoStartFocus  *bool                 `json:"autoStartFocus,omitempty"`
	PauseOnLock     *bool                 `json:"pauseOnLock,omitempty"`
	Label           *string               `json:"label,omitempty"`
	DailyGoal       *int                  `json:"dailyGoal,omitempty"`
	Shuffle         *bool                 `json:"shuffle,omitempty"`
	BreakBehavior   *config.BreakBehavior `json:"breakBehavior,omitempty"`
	DuckVolume      *int                  `json:"duckVolume,omitempty"`
	Sound           *bool                 `json:"sound,omitempty"`
	Toast           *bool                 `json:"toast,omitempty"`
	Theme           *config.Theme         `json:"theme,omitempty"`
	// Shortcuts, "" turns one off (T26).
	HotkeyStartPause *string `json:"hotkeyStartPause,omitempty"`
	HotkeyShowHide   *string `json:"hotkeyShowHide,omitempty"`
	HotkeyNextTrack  *string `json:"hotkeyNextTrack,omitempty"`
}

// Patch changes only the fields set in p, on top of the settings as they are
// now. Unlike Update with a copy taken earlier, it can't undo what the music
// row, the window or a quick earlier patch saved in the meantime. Invalid
// values are rejected as by Update.
func (s *SettingsService) Patch(p SettingsPatch) (config.Settings, error) {
	return s.modify(func(cur *config.Settings) {
		set(&cur.Timer.FocusMin, p.FocusMin)
		set(&cur.Timer.ShortBreakMin, p.ShortBreakMin)
		set(&cur.Timer.LongBreakMin, p.LongBreakMin)
		set(&cur.Timer.LongBreakEvery, p.LongBreakEvery)
		set(&cur.Timer.AutoStartBreaks, p.AutoStartBreaks)
		set(&cur.Timer.AutoStartFocus, p.AutoStartFocus)
		set(&cur.Timer.PauseOnLock, p.PauseOnLock)
		set(&cur.Timer.Label, p.Label)
		set(&cur.Timer.DailyGoal, p.DailyGoal)
		set(&cur.Music.Shuffle, p.Shuffle)
		set(&cur.Music.BreakBehavior, p.BreakBehavior)
		set(&cur.Music.DuckVolume, p.DuckVolume)
		set(&cur.Notify.Sound, p.Sound)
		set(&cur.Notify.Toast, p.Toast)
		set(&cur.Window.Theme, p.Theme)
		set(&cur.Hotkeys.StartPause, p.HotkeyStartPause)
		set(&cur.Hotkeys.ShowHide, p.HotkeyShowHide)
		set(&cur.Hotkeys.NextTrack, p.HotkeyNextTrack)
	})
}

func set[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}

// modify applies change to a copy of the current settings and commits it only
// if it validates and saves. Later services (music, window) reuse it to change
// their part without overwriting the rest.
//
// The lock is held throughout so concurrent changes reach the file, the timer
// and the frontend in the same order.
func (s *SettingsService) modify(change func(*config.Settings)) (config.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old := s.Get()
	next := old
	change(&next)
	next.Version = config.CurrentVersion
	if err := s.store.Save(next); err != nil { // Save validates first
		return old, err
	}
	s.current.Store(&next)

	// Only what the timer itself uses: not, say, pauseOnLock.
	if TimerConfig(next.Timer) != TimerConfig(old.Timer) {
		// Cannot fail: Save has validated the ranges, which are stricter than the timer's.
		if err := s.timer.UpdateConfig(TimerConfig(next.Timer)); err != nil {
			return next, err
		}
	}
	for _, fn := range s.listeners {
		fn(old, next)
	}
	s.emit.Emit(EventSettingsChanged, next)
	return next, nil
}
