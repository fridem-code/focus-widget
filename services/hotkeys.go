package services

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/config"
	"focuswidget/internal/hotkey"
)

// HotkeyWatch registers shortcuts; hotkey.Watch in the app, a fake in tests.
type HotkeyWatch func(combos map[int]hotkey.Combo, fired func(id int)) (stop func(), failed map[int]error)

// HotkeyService keeps the global shortcuts of the settings registered with
// Windows (T26) and runs their actions. Actions are keyed by the JSON names
// of config.HotkeySettings: "startPause", "showHide", "nextTrack".
//
// A shortcut that Windows refuses (another app has it) is reported by Status
// and the hotkeys:status event; the others still work.
type HotkeyService struct {
	settings *SettingsService
	emit     Emitter
	actions  map[string]func()
	watch    HotkeyWatch

	mu        sync.Mutex
	stop      func()
	names     []string // shortcut id-1 → action
	status    map[string]string
	suspended bool
}

func NewHotkeyService(settings *SettingsService, emit Emitter, actions map[string]func(), watch HotkeyWatch) *HotkeyService {
	h := &HotkeyService{settings: settings, emit: emit, actions: actions, watch: watch, status: map[string]string{}}
	settings.OnChange(func(old, next config.Settings) {
		if old.Hotkeys != next.Hotkeys {
			// Not under the settings lock: registering waits for another thread.
			go h.apply()
		}
	})
	return h
}

func (h *HotkeyService) ServiceStartup(context.Context, application.ServiceOptions) error {
	h.apply()
	return nil
}

func (h *HotkeyService) ServiceShutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.release()
	return nil
}

// Status returns the shortcuts that don't work, as action → reason; empty
// when all do.
func (h *HotkeyService) Status() map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return maps.Clone(h.status)
}

// Suspend frees all shortcuts while the settings screen records a new one, so
// pressing a shortcut that is already ours reaches the page instead of
// running its action; Suspend(false) takes them back.
func (h *HotkeyService) Suspend(on bool) {
	h.mu.Lock()
	h.suspended = on
	h.mu.Unlock()
	h.apply()
}

// apply registers the shortcuts as the settings have them now.
func (h *HotkeyService) apply() {
	h.mu.Lock()
	h.release()
	h.status = map[string]string{}
	if !h.suspended {
		combos := map[int]hotkey.Combo{}
		h.names = nil
		hk := h.settings.Get().Hotkeys
		for _, f := range hk.Fields() {
			if *f.Value == "" {
				continue
			}
			c, err := hotkey.Parse(*f.Value)
			if err != nil { // the settings were validated, so this is not expected
				h.status[f.Name] = "Не удалось включить: " + err.Error()
				continue
			}
			h.names = append(h.names, f.Name)
			combos[len(h.names)] = c
		}
		names := h.names
		var failed map[int]error
		h.stop, failed = h.watch(combos, func(id int) {
			if id >= 1 && id <= len(names) {
				if run := h.actions[names[id-1]]; run != nil {
					run()
				}
			}
		})
		for id, err := range failed {
			h.status[names[id-1]] = hotkeyProblem(err)
		}
	}
	status := maps.Clone(h.status)
	h.mu.Unlock()
	h.emit.Emit(EventHotkeysStatus, status)
}

// release must be called with mu held.
func (h *HotkeyService) release() {
	if h.stop != nil {
		h.stop()
		h.stop = nil
	}
}

func hotkeyProblem(err error) string {
	if errors.Is(err, hotkey.ErrTaken) {
		return "Занято другим приложением"
	}
	return "Не удалось включить: " + err.Error()
}
