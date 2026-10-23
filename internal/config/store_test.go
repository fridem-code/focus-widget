package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir(), slog.New(slog.DiscardHandler))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	s := newTestStore(t)
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("got %+v, want defaults", got)
	}
	if _, err := os.Stat(s.Path()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load must not create the file, stat err = %v", err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := Default()
	want.Timer.FocusMin = 50
	want.Timer.AutoStartFocus = true
	want.Music.BreakBehavior = BreakDuck
	want.Music.GenreID = "ambient"
	want.Music.SourceIndex = 3
	want.Music.Muted = true
	want.Window.Mode = WindowCompact
	want.Window.X, want.Window.Y = new(-300), new(200)
	want.Notify.Toast = false

	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestSaveCreatesDirAndLeavesNoTempFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", appDirName)
	s := NewStore(dir, slog.New(slog.DiscardHandler))
	for range 3 {
		if err := s.Save(Default()); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != fileName {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("dir contains %v, want only %s", names, fileName)
	}
}

func TestSaveRejectsInvalidSettings(t *testing.T) {
	s := newTestStore(t)
	bad := Default()
	bad.Music.Volume = 300
	var ve *ValidationError
	if err := s.Save(bad); !errors.As(err, &ve) {
		t.Fatalf("Save() = %v, want *ValidationError", err)
	}
	if _, err := os.Stat(s.Path()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("invalid settings were written, stat err = %v", err)
	}
}

func TestLoadPartialFileKeepsDefaultsForMissingFields(t *testing.T) {
	s := newTestStore(t)
	writeFile(t, s.Path(), `{"version": 1, "timer": {"focusMin": 40}, "music": {"volume": 70}}`)

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.Timer.FocusMin = 40
	want.Music.Volume = 70
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadCorruptFileBacksUpAndReturnsDefaults(t *testing.T) {
	for name, content := range map[string]string{
		"not json":    `{"timer": {`,
		"wrong type":  `{"timer": {"focusMin": "twenty"}}`,
		"null":        `null`,
		"old version": `{"version": 0}`,
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			writeFile(t, s.Path(), content)

			got, err := s.Load()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, Default()) {
				t.Fatalf("got %+v, want defaults", got)
			}
			backup, err := os.ReadFile(filepath.Join(filepath.Dir(s.Path()), backupName))
			if err != nil {
				t.Fatalf("backup missing: %v", err)
			}
			if string(backup) != content {
				t.Fatalf("backup = %q, want original %q", backup, content)
			}
			if _, err := os.Stat(s.Path()); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("corrupt file still in place, stat err = %v", err)
			}
		})
	}
}

func TestLoadResetsInvalidValuesOnly(t *testing.T) {
	s := newTestStore(t)
	writeFile(t, s.Path(), `{"version": 1, "timer": {"focusMin": 0, "shortBreakMin": 7}, "music": {"breakBehavior": "louder"}}`)

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.Timer.ShortBreakMin = 7
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadNewerVersionBestEffort(t *testing.T) {
	s := newTestStore(t)
	writeFile(t, s.Path(), `{"version": 99, "timer": {"focusMin": 30}, "futureField": true}`)

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Timer.FocusMin != 30 || got.Version != CurrentVersion {
		t.Fatalf("got focusMin=%d version=%d", got.Timer.FocusMin, got.Version)
	}
}

func TestSavedFileUsesSchemaFieldNames(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(Default()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]map[string]any
	_ = json.Unmarshal(data, &doc) // "version" is a number, not an object; it is checked below
	for _, key := range []string{`"version": 1`, `"focusMin": 25`, `"breakBehavior": "pause"`, `"x": null`} {
		if !strings.Contains(string(data), key) {
			t.Errorf("saved file lacks %s:\n%s", key, data)
		}
	}
	if doc["notify"]["toast"] != true {
		t.Errorf("notify.toast = %v, want true", doc["notify"]["toast"])
	}
}

func TestDefaultDir(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Skipf("no user config dir: %v", err)
	}
	if filepath.Base(dir) != appDirName {
		t.Fatalf("DefaultDir() = %s, want .../%s", dir, appDirName)
	}
}
