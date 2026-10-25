//go:build !windows

package autostart

import "errors"

const (
	ValueName = "Focus Widget"
	Flag      = "--autostart"
)

// Registry does nothing outside Windows: the widget targets Windows only.
type Registry struct{}

func New() *Registry { return &Registry{} }

func (*Registry) Enabled() (bool, error) { return false, nil }

func (*Registry) Set(bool) error { return errors.New("autostart: Windows only") }

func (*Registry) Refresh() (bool, error) { return false, nil }
