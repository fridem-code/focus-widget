package autostart

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// testRegistry works on a scratch key under HKCU instead of the real Run key.
func testRegistry(t *testing.T, exe string) *Registry {
	t.Helper()
	key := `Software\FocusWidgetTest\` + t.Name()
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, key)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\FocusWidgetTest`)
	})
	return &Registry{Exe: func() (string, error) { return exe, nil }, key: key}
}

func TestSetAndEnabled(t *testing.T) {
	r := testRegistry(t, `C:\Apps\Focus Widget\focuswidget.exe`)
	if on, err := r.Enabled(); err != nil || on {
		t.Fatalf("Enabled() = %v, %v before Set", on, err)
	}
	if err := r.Set(true); err != nil {
		t.Fatal(err)
	}
	if on, err := r.Enabled(); err != nil || !on {
		t.Fatalf("Enabled() = %v, %v after Set(true)", on, err)
	}
	if v, _ := r.read(); v != `"C:\Apps\Focus Widget\focuswidget.exe" --autostart` {
		t.Fatalf("Run value = %q", v)
	}
	if err := r.Set(false); err != nil {
		t.Fatal(err)
	}
	if err := r.Set(false); err != nil { // already off: fine
		t.Fatal(err)
	}
	if on, _ := r.Enabled(); on {
		t.Fatal("still enabled after Set(false)")
	}
}

func TestRefreshFollowsTheExecutable(t *testing.T) {
	r := testRegistry(t, `C:\Old\focuswidget.exe`)
	if changed, err := r.Refresh(); err != nil || changed {
		t.Fatalf("Refresh() = %v, %v while disabled", changed, err)
	}
	if err := r.Set(true); err != nil {
		t.Fatal(err)
	}
	if changed, _ := r.Refresh(); changed {
		t.Fatal("Refresh changed an entry that was up to date")
	}
	r.Exe = func() (string, error) { return `C:\New\focuswidget.exe`, nil }
	if changed, err := r.Refresh(); err != nil || !changed {
		t.Fatalf("Refresh() = %v, %v after the exe moved", changed, err)
	}
	if v, _ := r.read(); v != Command(`C:\New\focuswidget.exe`) {
		t.Fatalf("Run value = %q", v)
	}
}
