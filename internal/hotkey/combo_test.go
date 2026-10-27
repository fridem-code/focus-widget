package hotkey

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Combo
		back string
	}{
		{"Ctrl+Alt+Shift+P", Combo{ModCtrl | ModAlt | ModShift, 'P'}, "Ctrl+Alt+Shift+P"},
		{"shift+alt+ctrl+p", Combo{ModCtrl | ModAlt | ModShift, 'P'}, "Ctrl+Alt+Shift+P"},
		{"Alt+7", Combo{ModAlt, '7'}, "Alt+7"},
		{"Win+F12", Combo{ModWin, 0x7B}, "Win+F12"},
		{"Ctrl+Space", Combo{ModCtrl, 0x20}, "Ctrl+Space"},
		{"Ctrl+Alt+Up", Combo{ModCtrl | ModAlt, 0x26}, "Ctrl+Alt+Up"},
		{"Ctrl+F1", Combo{ModCtrl, 0x70}, "Ctrl+F1"},
	} {
		got, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
		if s := got.String(); s != tc.back {
			t.Errorf("Parse(%q).String() = %q, want %q", tc.in, s, tc.back)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "P", "Shift+P", "Ctrl+", "Ctrl+Ctrl+P", "Hyper+P", "Ctrl+F13", "Ctrl+Tab", "Ctrl+PP"} {
		if c, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %+v, want an error", in, c)
		}
	}
	if _, err := Parse("Shift+F5"); !errors.Is(err, ErrNoModifier) {
		t.Errorf("Parse(Shift+F5) = %v, want ErrNoModifier", err)
	}
}
