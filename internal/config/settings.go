// Package config holds the user settings (docs/SPEC.md §5.3): schema, defaults,
// validation and atomic persistence to config.json.
package config

import (
	"fmt"
	"math"
	"strings"
)

// CurrentVersion is the schema version written by this build.
const CurrentVersion = 1

type BreakBehavior string

const (
	BreakPause    BreakBehavior = "pause"
	BreakDuck     BreakBehavior = "duck"
	BreakContinue BreakBehavior = "continue"
)

type WindowMode string

const (
	WindowExpanded WindowMode = "expanded"
	WindowCompact  WindowMode = "compact"
)

type Settings struct {
	Version int            `json:"version"`
	Timer   TimerSettings  `json:"timer"`
	Music   MusicSettings  `json:"music"`
	Window  WindowSettings `json:"window"`
	Notify  NotifySettings `json:"notify"`
}

type TimerSettings struct {
	FocusMin        int  `json:"focusMin"`
	ShortBreakMin   int  `json:"shortBreakMin"`
	LongBreakMin    int  `json:"longBreakMin"`
	LongBreakEvery  int  `json:"longBreakEvery"`
	AutoStartBreaks bool `json:"autoStartBreaks"`
	AutoStartFocus  bool `json:"autoStartFocus"`
}

type MusicSettings struct {
	Volume        int           `json:"volume"`
	Muted         bool          `json:"muted"`
	Shuffle       bool          `json:"shuffle"`
	BreakBehavior BreakBehavior `json:"breakBehavior"`
	DuckVolume    int           `json:"duckVolume"`
	GenreID       string        `json:"genreId"`
	SourceIndex   int           `json:"sourceIndex"`
}

type WindowSettings struct {
	Mode        WindowMode `json:"mode"`
	AlwaysOnTop bool       `json:"alwaysOnTop"`
	// X and Y are nil until the window has been moved; the app then picks a default position.
	X *int `json:"x"`
	Y *int `json:"y"`
}

type NotifySettings struct {
	Sound bool `json:"sound"`
	Toast bool `json:"toast"`
}

const (
	MinPhaseMin       = 1
	MaxPhaseMin       = 180
	MinLongBreakEvery = 2
	MaxLongBreakEvery = 10
)

// Default returns the settings used on first launch.
func Default() Settings {
	return Settings{
		Version: CurrentVersion,
		Timer: TimerSettings{
			FocusMin:        25,
			ShortBreakMin:   5,
			LongBreakMin:    15,
			LongBreakEvery:  4,
			AutoStartBreaks: true,
			AutoStartFocus:  false,
		},
		Music: MusicSettings{
			Volume:        50,
			Shuffle:       true,
			BreakBehavior: BreakPause,
			DuckVolume:    20,
			GenreID:       "lofi",
		},
		Window: WindowSettings{
			Mode:        WindowExpanded,
			AlwaysOnTop: true,
		},
		Notify: NotifySettings{Sound: true, Toast: true},
	}
}

// ValidationError lists every invalid field, e.g. "timer.focusMin: 0 not in [1, 180]".
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "invalid settings: " + strings.Join(e.Problems, "; ")
}

// Validate reports all out-of-range or unknown values.
func (s Settings) Validate() error {
	if problems := s.check(false); len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

// Normalize resets every invalid field to its default and returns what it fixed.
func (s *Settings) Normalize() []string {
	return s.check(true)
}

// check is the single list of validation rules shared by Validate and Normalize.
func (s *Settings) check(fix bool) []string {
	def := Default()
	var problems []string
	report := func(field string, value any, rule string, reset func()) {
		problems = append(problems, fmt.Sprintf("%s: %v %s", field, value, rule))
		if fix {
			reset()
		}
	}
	intRange := func(field string, v *int, lo, hi, defV int) {
		if *v < lo || *v > hi {
			report(field, *v, fmt.Sprintf("not in [%d, %d]", lo, hi), func() { *v = defV })
		}
	}

	t := &s.Timer
	intRange("timer.focusMin", &t.FocusMin, MinPhaseMin, MaxPhaseMin, def.Timer.FocusMin)
	intRange("timer.shortBreakMin", &t.ShortBreakMin, MinPhaseMin, MaxPhaseMin, def.Timer.ShortBreakMin)
	intRange("timer.longBreakMin", &t.LongBreakMin, MinPhaseMin, MaxPhaseMin, def.Timer.LongBreakMin)
	intRange("timer.longBreakEvery", &t.LongBreakEvery, MinLongBreakEvery, MaxLongBreakEvery, def.Timer.LongBreakEvery)

	m := &s.Music
	intRange("music.volume", &m.Volume, 0, 100, def.Music.Volume)
	intRange("music.duckVolume", &m.DuckVolume, 0, 100, def.Music.DuckVolume)
	intRange("music.sourceIndex", &m.SourceIndex, 0, math.MaxInt, 0)
	switch m.BreakBehavior {
	case BreakPause, BreakDuck, BreakContinue:
	default:
		report("music.breakBehavior", m.BreakBehavior, "is not pause|duck|continue",
			func() { m.BreakBehavior = def.Music.BreakBehavior })
	}

	switch s.Window.Mode {
	case WindowExpanded, WindowCompact:
	default:
		report("window.mode", s.Window.Mode, "is not expanded|compact",
			func() { s.Window.Mode = def.Window.Mode })
	}
	if (s.Window.X == nil) != (s.Window.Y == nil) {
		report("window.x/y", "", "must be both set or both null", func() { s.Window.X, s.Window.Y = nil, nil })
	}

	return problems
}
