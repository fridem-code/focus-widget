//go:build !windows

package session

// Watch does nothing outside Windows: the widget targets Windows only.
func Watch(func(locked bool)) (stop func(), err error) { return func() {}, nil }
