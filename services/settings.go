package services

import (
	"sync"

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

	mu      sync.Mutex
	current config.Settings
}

// NewSettingsService starts from initial, which must already be valid (as
// returned by config.Store.Load).
func NewSettingsService(store *config.Store, initial config.Settings, t *timer.Timer, emit Emitter) *SettingsService {
	return &SettingsService{store: store, timer: t, emit: emit, current: initial}
}

func (s *SettingsService) Get() config.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
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
	Shuffle         *bool                 `json:"shuffle,omitempty"`
	BreakBehavior   *config.BreakBehavior `json:"breakBehavior,omitempty"`
	DuckVolume      *int                  `json:"duckVolume,omitempty"`
	Sound           *bool                 `json:"sound,omitempty"`
	Toast           *bool                 `json:"toast,omitempty"`
	Theme           *config.Theme         `json:"theme,omitempty"`
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
		set(&cur.Music.Shuffle, p.Shuffle)
		set(&cur.Music.BreakBehavior, p.BreakBehavior)
		set(&cur.Music.DuckVolume, p.DuckVolume)
		set(&cur.Notify.Sound, p.Sound)
		set(&cur.Notify.Toast, p.Toast)
		set(&cur.Window.Theme, p.Theme)
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

	next := s.current
	change(&next)
	next.Version = config.CurrentVersion
	if err := s.store.Save(next); err != nil { // Save validates first
		return s.current, err
	}
	timerChanged := next.Timer != s.current.Timer
	s.current = next

	if timerChanged {
		// Cannot fail: Save has validated the ranges, which are stricter than the timer's.
		if err := s.timer.UpdateConfig(TimerConfig(next.Timer)); err != nil {
			return next, err
		}
	}
	s.emit.Emit(EventSettingsChanged, next)
	return next, nil
}
