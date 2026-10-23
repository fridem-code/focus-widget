package config

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default() is invalid: %v", err)
	}
}

func TestValidateAndNormalize(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Settings)
		field  string // prefix of the expected problem
		check  func(Settings) bool
	}{
		{"focus too short", func(s *Settings) { s.Timer.FocusMin = 0 }, "timer.focusMin",
			func(s Settings) bool { return s.Timer.FocusMin == 25 }},
		{"focus too long", func(s *Settings) { s.Timer.FocusMin = 181 }, "timer.focusMin",
			func(s Settings) bool { return s.Timer.FocusMin == 25 }},
		{"short break", func(s *Settings) { s.Timer.ShortBreakMin = -1 }, "timer.shortBreakMin",
			func(s Settings) bool { return s.Timer.ShortBreakMin == 5 }},
		{"long break", func(s *Settings) { s.Timer.LongBreakMin = 500 }, "timer.longBreakMin",
			func(s Settings) bool { return s.Timer.LongBreakMin == 15 }},
		{"long break every too small", func(s *Settings) { s.Timer.LongBreakEvery = 1 }, "timer.longBreakEvery",
			func(s Settings) bool { return s.Timer.LongBreakEvery == 4 }},
		{"long break every too big", func(s *Settings) { s.Timer.LongBreakEvery = 11 }, "timer.longBreakEvery",
			func(s Settings) bool { return s.Timer.LongBreakEvery == 4 }},
		{"volume", func(s *Settings) { s.Music.Volume = 101 }, "music.volume",
			func(s Settings) bool { return s.Music.Volume == 50 }},
		{"duck volume", func(s *Settings) { s.Music.DuckVolume = -5 }, "music.duckVolume",
			func(s Settings) bool { return s.Music.DuckVolume == 20 }},
		{"source index", func(s *Settings) { s.Music.SourceIndex = -1 }, "music.sourceIndex",
			func(s Settings) bool { return s.Music.SourceIndex == 0 }},
		{"break behavior", func(s *Settings) { s.Music.BreakBehavior = "louder" }, "music.breakBehavior",
			func(s Settings) bool { return s.Music.BreakBehavior == BreakPause }},
		{"window mode", func(s *Settings) { s.Window.Mode = "fullscreen" }, "window.mode",
			func(s Settings) bool { return s.Window.Mode == WindowExpanded }},
		{"half a position", func(s *Settings) { s.Window.X = new(10) }, "window.x/y",
			func(s Settings) bool { return s.Window.X == nil && s.Window.Y == nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Default()
			tt.mutate(&s)

			err := s.Validate()
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Validate() = %v, want *ValidationError", err)
			}
			if len(ve.Problems) != 1 || !strings.HasPrefix(ve.Problems[0], tt.field) {
				t.Fatalf("problems = %q, want one starting with %q", ve.Problems, tt.field)
			}

			fixed := s.Normalize()
			if len(fixed) != 1 {
				t.Fatalf("Normalize() fixed %q, want exactly one field", fixed)
			}
			if !tt.check(s) {
				t.Fatalf("field not reset to default: %+v", s)
			}
			if err := s.Validate(); err != nil {
				t.Fatalf("still invalid after Normalize: %v", err)
			}
		})
	}
}

func TestValidateBoundsAreInclusive(t *testing.T) {
	s := Default()
	s.Timer.FocusMin = MinPhaseMin
	s.Timer.LongBreakMin = MaxPhaseMin
	s.Timer.LongBreakEvery = MaxLongBreakEvery
	s.Music.Volume = 0
	s.Music.DuckVolume = 100
	s.Music.BreakBehavior = BreakContinue
	s.Window.Mode = WindowCompact
	s.Window.X, s.Window.Y = new(-1200), new(40) // negative: monitor left of the primary one
	if err := s.Validate(); err != nil {
		t.Fatalf("boundary values rejected: %v", err)
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	s := Default()
	s.Timer.FocusMin = 0
	s.Music.Volume = 200
	var ve *ValidationError
	if !errors.As(s.Validate(), &ve) || len(ve.Problems) != 2 {
		t.Fatalf("want 2 problems, got %v", s.Validate())
	}
}
