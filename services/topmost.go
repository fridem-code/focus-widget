package services

// TopmostKeeper keeps an always-on-top widget above apps that open over the
// topmost band. A freshly started Paint covered the pinned widget, and only
// pinning it again after Paint was up brought it back (docs/BUGS.md B2), so
// the widget is re-pinned whenever another window comes to the foreground.
// Not a Wails service: app wires it to the system hook.
type TopmostKeeper struct {
	settings *SettingsService
	window   *WindowService
	pin      func()
}

// NewTopmostKeeper re-pins with pin, which must not activate the window.
func NewTopmostKeeper(settings *SettingsService, window *WindowService, pin func()) *TopmostKeeper {
	return &TopmostKeeper{settings: settings, window: window, pin: pin}
}

// ForegroundChanged handles a foreground change to another app's window. It
// runs on the hook's thread.
func (k *TopmostKeeper) ForegroundChanged() {
	if k.settings.Get().Window.AlwaysOnTop && k.window.Visible() {
		k.pin()
	}
}
