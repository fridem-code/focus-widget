// Package config holds the user settings (docs/SPEC.md §5.3): schema, defaults,
// validation and atomic persistence to config.json.
package config

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"focuswidget/internal/hotkey"
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

// Theme: "system" follows the Windows light/dark setting.
type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeDark   Theme = "dark"
	ThemeLight  Theme = "light"
)

type Settings struct {
	Version int            `json:"version"`
	Timer   TimerSettings  `json:"timer"`
	Music   MusicSettings  `json:"music"`
	Window  WindowSettings `json:"window"`
	Notify  NotifySettings `json:"notify"`
	Hotkeys HotkeySettings `json:"hotkeys"`
}

type TimerSettings struct {
	FocusMin        int  `json:"focusMin"`
	ShortBreakMin   int  `json:"shortBreakMin"`
	LongBreakMin    int  `json:"longBreakMin"`
	LongBreakEvery  int  `json:"longBreakEvery"`
	AutoStartBreaks bool `json:"autoStartBreaks"`
	AutoStartFocus  bool `json:"autoStartFocus"`
	// PauseOnLock pauses a running focus phase while the PC is locked (T20).
	PauseOnLock bool `json:"pauseOnLock"`
	// Label is what the user is working on (T24), kept until they change it;
	// it goes into the phase-end toast and the session log.
	Label string `json:"label"`
	// DailyGoal is the number of focus sessions a day to aim for (T25); 0 is no goal.
	DailyGoal int `json:"dailyGoal"`
}

type MusicSettings struct {
	Volume        int           `json:"volume"`
	Muted         bool          `json:"muted"`
	Shuffle       bool          `json:"shuffle"`
	BreakBehavior BreakBehavior `json:"breakBehavior"`
	DuckVolume    int           `json:"duckVolume"`
	GenreID       string        `json:"genreId"`
	SourceIndex   int           `json:"sourceIndex"`
	// LocalFolder is the user's music folder for the "Мои файлы" genre
	// (T22); empty when none is chosen.
	LocalFolder string `json:"localFolder"`
}

type WindowSettings struct {
	Mode        WindowMode `json:"mode"`
	AlwaysOnTop bool       `json:"alwaysOnTop"`
	Theme       Theme      `json:"theme"`
	// X and Y are nil until the window has been moved; the app then picks a default position.
	X *int `json:"x"`
	Y *int `json:"y"`
}

type NotifySettings struct {
	Sound bool `json:"sound"`
	Toast bool `json:"toast"`
}

// HotkeySettings are system-wide shortcuts such as "Ctrl+Alt+Shift+P" (T26,
// the format of hotkey.Parse); "" turns one off.
type HotkeySettings struct {
	StartPause string `json:"startPause"`
	ShowHide   string `json:"showHide"`
	NextTrack  string `json:"nextTrack"`
}

// Fields lists the shortcuts by their JSON names, for code that treats them alike.
func (h *HotkeySettings) Fields() []struct {
	Name  string
	Value *string
} {
	return []struct {
		Name  string
		Value *string
	}{{"startPause", &h.StartPause}, {"showHide", &h.ShowHide}, {"nextTrack", &h.NextTrack}}
}

const (
	MinPhaseMin       = 1
	MaxPhaseMin       = 180
	MinLongBreakEvery = 2
	MaxLongBreakEvery = 10
	MaxLabelRunes     = 60
	MaxDailyGoal      = 24
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
			PauseOnLock:     true,
			DailyGoal:       8,
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
			Theme:       ThemeSystem,
		},
		Notify: NotifySettings{Sound: true, Toast: true},
		// Three modifiers: unlikely to be taken by the IDE the user works in.
		Hotkeys: HotkeySettings{
			StartPause: "Ctrl+Alt+Shift+P",
			ShowHide:   "Ctrl+Alt+Shift+F",
			NextTrack:  "Ctrl+Alt+Shift+N",
		},
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
	intRange("timer.dailyGoal", &t.DailyGoal, 0, MaxDailyGoal, def.Timer.DailyGoal)
	if n := utf8.RuneCountInString(t.Label); n > MaxLabelRunes || strings.ContainsAny(t.Label, "\r\n") {
		report("timer.label", fmt.Sprintf("%q", t.Label), fmt.Sprintf("is not one line of at most %d characters", MaxLabelRunes),
			func() { t.Label = "" })
	}

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
	if m.LocalFolder != "" && !filepath.IsAbs(m.LocalFolder) {
		report("music.localFolder", m.LocalFolder, "is not an absolute path", func() { m.LocalFolder = "" })
	}

	switch s.Window.Mode {
	case WindowExpanded, WindowCompact:
	default:
		report("window.mode", s.Window.Mode, "is not expanded|compact",
			func() { s.Window.Mode = def.Window.Mode })
	}
	switch s.Window.Theme {
	case ThemeSystem, ThemeDark, ThemeLight:
	default:
		report("window.theme", s.Window.Theme, "is not system|dark|light",
			func() { s.Window.Theme = def.Window.Theme })
	}
	if (s.Window.X == nil) != (s.Window.Y == nil) {
		report("window.x/y", "", "must be both set or both null", func() { s.Window.X, s.Window.Y = nil, nil })
	}

	// Each shortcut parses, and no two actions share one (compared parsed, so
	// "Alt+Ctrl+P" and "Ctrl+Alt+P" are the same).
	used := map[hotkey.Combo]string{}
	defHotkeys := def.Hotkeys.Fields()
	for i, f := range s.Hotkeys.Fields() {
		if *f.Value == "" {
			continue
		}
		field := "hotkeys." + f.Name
		c, err := hotkey.Parse(*f.Value)
		if err != nil {
			report(field, *f.Value, err.Error(), func() { *f.Value = *defHotkeys[i].Value })
			continue
		}
		if other, ok := used[c]; ok {
			report(field, *f.Value, "is also hotkeys."+other, func() { *f.Value = "" })
			continue
		}
		used[c] = f.Name
	}

	return problems
}
