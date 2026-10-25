//go:build !windows

package glass

import "errors"

// Available is false outside Windows: the widget targets Windows only.
func Available() bool { return false }

// SystemDark reports false outside Windows.
func SystemDark() bool { return false }

// Apply is not supported outside Windows.
func Apply(uintptr, bool) error { return errors.New("glass: Windows only") }
