// Package services exposes the app's logic to the frontend as Wails services
// (bindings) and events (docs/SPEC.md §5, §5.2).
package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/timer"
)

// Event names, see docs/SPEC.md §5.2.
const (
	EventTimerState      = "timer:state"
	EventTimerTransition = "timer:transition"
	EventSettingsChanged = "settings:changed"
)

func init() {
	// Registered events get typed TS bindings, and Emit rejects a payload of
	// the wrong type.
	application.RegisterEvent[timer.State](EventTimerState)
	application.RegisterEvent[timer.Transition](EventTimerTransition)
	application.RegisterEvent[config.Settings](EventSettingsChanged)
}

// Emitter publishes an event to the frontend; *application.EventManager
// implements it.
type Emitter interface {
	Emit(name string, data ...any) bool
}
