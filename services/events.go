// Package services exposes the app's logic to the frontend as Wails services
// (bindings) and events (docs/SPEC.md §5, §5.2).
package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/stats"
	"focuswidget/internal/timer"
)

// Event names, see docs/SPEC.md §5.2.
const (
	EventTimerState      = "timer:state"
	EventTimerTransition = "timer:transition"
	EventSettingsChanged = "settings:changed"
	EventWindowMode      = "window:mode"
	EventWindowVisible   = "window:visible"
	// EventStatsChanged carries the new statistics summary after a focus session ends (T23).
	EventStatsChanged = "stats:changed"
	// EventHotkeysStatus carries the shortcuts that could not be registered (T26).
	EventHotkeysStatus = "hotkeys:status"
	// EventUICommand asks the frontend to do something it owns: "compact",
	// "expand" (it sequences music and resizing), "next-genre" or "next-track"
	// (the player lives there). Sent by the tray and the global shortcuts.
	EventUICommand = "ui:command"
)

// UI commands carried by EventUICommand.
const (
	UICompact   = "compact"
	UIExpand    = "expand"
	UINextGenre = "next-genre"
	// UINextTrack comes from the global shortcut (T26).
	UINextTrack = "next-track"
)

func init() {
	// Registered events get typed TS bindings, and Emit rejects a payload of
	// the wrong type.
	application.RegisterEvent[timer.State](EventTimerState)
	application.RegisterEvent[timer.Transition](EventTimerTransition)
	application.RegisterEvent[config.Settings](EventSettingsChanged)
	application.RegisterEvent[config.WindowMode](EventWindowMode)
	application.RegisterEvent[bool](EventWindowVisible)
	application.RegisterEvent[string](EventUICommand)
	application.RegisterEvent[stats.Summary](EventStatsChanged)
	application.RegisterEvent[map[string]string](EventHotkeysStatus)
}

// Emitter publishes an event to the frontend; *application.EventManager
// implements it.
type Emitter interface {
	Emit(name string, data ...any) bool
}
