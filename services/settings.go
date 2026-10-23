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
