package services

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Autostart is starting with Windows (internal/autostart).
type Autostart interface {
	Enabled() (bool, error)
	Set(on bool) error
	// Refresh points an enabled entry at the current executable.
	Refresh() (changed bool, err error)
}

// AutostartService lets the settings screen show and switch starting with
// Windows (T19). The Run key is the truth, not the settings file: Task Manager
// can switch it too, so the screen asks every time it opens.
type AutostartService struct {
	reg  Autostart
	warn func(error)
}

func NewAutostartService(reg Autostart, warn func(error)) *AutostartService {
	return &AutostartService{reg: reg, warn: warn}
}

// ServiceStartup keeps an enabled entry pointing at this executable, so a
// moved or reinstalled widget still starts. A failure only gets logged.
func (s *AutostartService) ServiceStartup(context.Context, application.ServiceOptions) error {
	if _, err := s.reg.Refresh(); err != nil {
		s.warn(err)
	}
	return nil
}

// Enabled reports whether the widget starts with Windows.
func (s *AutostartService) Enabled() (bool, error) { return s.reg.Enabled() }

// Set turns starting with Windows on or off.
func (s *AutostartService) Set(on bool) error { return s.reg.Set(on) }
