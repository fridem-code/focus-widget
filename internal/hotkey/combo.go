// Package hotkey registers system-wide keyboard shortcuts (T26): "Ctrl+Alt+P"
// works whichever application has the focus.
package hotkey

import (
	"errors"
	"fmt"
	"strings"
)

// Modifier flags, as RegisterHotKey takes them.
const (
	ModAlt   uint32 = 0x1
	ModCtrl  uint32 = 0x2
	ModShift uint32 = 0x4
	ModWin   uint32 = 0x8
)

// Combo is a shortcut: modifiers and a virtual-key code.
type Combo struct {
	Mods uint32
	Key  uint32
}

// modifiers in the order String writes them.
var modifiers = []struct {
	name string
	flag uint32
}{{"Ctrl", ModCtrl}, {"Alt", ModAlt}, {"Shift", ModShift}, {"Win", ModWin}}

// keys maps key names to virtual-key codes: letters, digits, F1–F12, Space
// and the arrows. The page names keys by their position (KeyboardEvent.code),
// so a shortcut is the same in any keyboard layout.
var keys = func() map[string]uint32 {
	m := map[string]uint32{"Space": 0x20, "Left": 0x25, "Up": 0x26, "Right": 0x27, "Down": 0x28}
	for c := 'A'; c <= 'Z'; c++ {
		m[string(c)] = uint32(c)
	}
	for c := '0'; c <= '9'; c++ {
		m[string(c)] = uint32(c)
	}
	for i := 1; i <= 12; i++ {
		m[fmt.Sprintf("F%d", i)] = 0x70 + uint32(i-1)
	}
	return m
}()

// ErrNoModifier: a shortcut needs Ctrl, Alt or Win, or it would take a key
// (or Shift+key) away from typing in every other application.
var ErrNoModifier = errors.New("needs Ctrl, Alt or Win")

// Parse reads a shortcut such as "Ctrl+Alt+Shift+P": modifiers in any order,
// then one key. Names are case-insensitive.
func Parse(s string) (Combo, error) {
	parts := strings.Split(s, "+")
	var c Combo
	for _, p := range parts[:len(parts)-1] {
		flag, ok := modifierFlag(p)
		if !ok {
			return Combo{}, fmt.Errorf("%q: unknown modifier %q", s, p)
		}
		if c.Mods&flag != 0 {
			return Combo{}, fmt.Errorf("%q: %s twice", s, p)
		}
		c.Mods |= flag
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	for name, vk := range keys {
		if strings.EqualFold(name, last) {
			c.Key = vk
		}
	}
	if c.Key == 0 {
		return Combo{}, fmt.Errorf("%q: unknown key %q", s, last)
	}
	if c.Mods&(ModCtrl|ModAlt|ModWin) == 0 {
		return Combo{}, fmt.Errorf("%q: %w", s, ErrNoModifier)
	}
	return c, nil
}

func modifierFlag(name string) (uint32, bool) {
	for _, m := range modifiers {
		if strings.EqualFold(m.name, strings.TrimSpace(name)) {
			return m.flag, true
		}
	}
	return 0, false
}

// String writes the shortcut the way Parse reads it, modifiers in a fixed
// order, so two spellings of one shortcut compare equal after a round trip.
func (c Combo) String() string {
	var b strings.Builder
	for _, m := range modifiers {
		if c.Mods&m.flag != 0 {
			b.WriteString(m.name)
			b.WriteString("+")
		}
	}
	for name, vk := range keys {
		if vk == c.Key {
			b.WriteString(name)
		}
	}
	return b.String()
}
