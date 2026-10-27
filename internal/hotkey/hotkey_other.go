//go:build !windows

package hotkey

import "errors"

// ErrTaken: another application already has the shortcut.
var ErrTaken = errors.New("taken by another application")

// Watch does nothing outside Windows: the widget targets Windows only.
func Watch(map[int]Combo, func(id int)) (stop func(), failed map[int]error) {
	return func() {}, nil
}
