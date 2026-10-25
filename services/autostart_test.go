package services

import (
	"context"
	"errors"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type fakeAutostart struct {
	on         bool
	refreshes  int
	refreshErr error
}

func (a *fakeAutostart) Enabled() (bool, error) { return a.on, nil }
func (a *fakeAutostart) Set(on bool) error      { a.on = on; return nil }
func (a *fakeAutostart) Refresh() (bool, error) { a.refreshes++; return false, a.refreshErr }

func TestAutostartService(t *testing.T) {
	reg := &fakeAutostart{}
	var warned []error
	svc := NewAutostartService(reg, func(err error) { warned = append(warned, err) })

	if err := svc.Set(true); err != nil {
		t.Fatal(err)
	}
	if on, _ := svc.Enabled(); !on {
		t.Fatal("not enabled after Set(true)")
	}

	// Startup refreshes the entry; a failure must not stop the app.
	reg.refreshErr = errors.New("access denied")
	if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() = %v, want nil", err)
	}
	if reg.refreshes != 1 || len(warned) != 1 {
		t.Fatalf("refreshes = %d, warned = %v", reg.refreshes, warned)
	}
}
