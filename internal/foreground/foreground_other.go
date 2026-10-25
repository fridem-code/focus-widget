//go:build !windows

package foreground

// Watch does nothing outside Windows: the widget targets Windows only.
func Watch(func()) (stop func(), err error) { return func() {}, nil }

// Pin does nothing outside Windows.
func Pin(uintptr) error { return nil }

// Keep just runs show outside Windows.
func Keep(show func()) { show() }
