// Package autostart starts the widget with Windows through the current user's
// Run key (T19). The registry is the source of truth, not the settings file:
// the user can also turn it off in Task Manager → Startup apps.
package autostart

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	// ValueName is the Run value; the uninstaller deletes it too.
	ValueName = "Focus Widget"
	// Flag marks a start by Windows: the window then shows without taking
	// the focus from what the user is doing.
	Flag = "--autostart"
)

// Registry is the Run key of the current user.
type Registry struct {
	// Exe returns the path to start; os.Executable when nil.
	Exe func() (string, error)
	key string // tests use another key
}

// New uses the real Run key and this executable.
func New() *Registry { return &Registry{Exe: os.Executable, key: runKey} }

// Command is the Run value for exe.
func Command(exe string) string { return `"` + exe + `" ` + Flag }

// Enabled reports whether the widget starts with Windows.
func (r *Registry) Enabled() (bool, error) {
	_, err := r.read()
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Set turns starting with Windows on (for the current executable) or off.
func (r *Registry) Set(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, r.key, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer func() { _ = k.Close() }()
	if !on {
		if err := k.DeleteValue(ValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("remove autostart: %w", err)
		}
		return nil
	}
	exe, err := r.exe()
	if err != nil {
		return err
	}
	if err := k.SetStringValue(ValueName, Command(exe)); err != nil {
		return fmt.Errorf("set autostart: %w", err)
	}
	return nil
}

// Refresh points an enabled autostart at this executable, so moving or
// reinstalling the widget elsewhere does not leave a dead entry. It reports
// whether the entry changed.
func (r *Registry) Refresh() (bool, error) {
	cur, err := r.read()
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	exe, err := r.exe()
	if err != nil {
		return false, err
	}
	if strings.EqualFold(cur, Command(exe)) {
		return false, nil
	}
	return true, r.Set(true)
}

func (r *Registry) read() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, r.key, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue(ValueName)
	return v, err
}

func (r *Registry) exe() (string, error) {
	exe := os.Executable
	if r.Exe != nil {
		exe = r.Exe
	}
	path, err := exe()
	if err != nil {
		return "", fmt.Errorf("find the executable: %w", err)
	}
	return path, nil
}
