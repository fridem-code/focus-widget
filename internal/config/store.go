package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const (
	appDirName = "FocusWidget"
	fileName   = "config.json"
	backupName = "config.json.bak"
)

// migrations[i] upgrades a raw document from version i+1 to i+2. Append one
// function per schema bump and increase CurrentVersion.
var migrations []func(doc map[string]any) error

// DefaultDir returns %APPDATA%\FocusWidget (or the OS equivalent).
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

// Store loads and saves Settings in a single directory.
type Store struct {
	dir    string
	logger *slog.Logger
	mu     sync.Mutex // serialises Save
}

func NewStore(dir string, logger *slog.Logger) *Store {
	return &Store{dir: dir, logger: logger}
}

func (s *Store) Path() string { return filepath.Join(s.dir, fileName) }

// Load never fails because of the file's content: a missing file yields
// defaults, an unparsable one is moved to config.json.bak and replaced by
// defaults, and invalid values are reset field by field. Only I/O errors other
// than "not found" are returned, together with the defaults.
func (s *Store) Load() (Settings, error) {
	data, err := os.ReadFile(s.Path())
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read settings: %w", err)
	}

	settings, err := decode(data)
	if err != nil {
		s.logger.Warn("settings file is corrupt, using defaults", "path", s.Path(), "err", err)
		if bErr := os.Rename(s.Path(), filepath.Join(s.dir, backupName)); bErr != nil {
			s.logger.Warn("could not back up corrupt settings", "err", bErr)
		}
		return Default(), nil
	}

	if fixed := settings.Normalize(); len(fixed) > 0 {
		s.logger.Warn("reset invalid settings to defaults", "fields", fixed)
	}
	return settings, nil
}

// decode applies migrations and overlays the document on top of the defaults,
// so fields missing from the file keep their default values.
func decode(data []byte) (Settings, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return Settings{}, err
	}
	if doc == nil {
		return Settings{}, errors.New("settings document is null")
	}

	version := 1
	if v, ok := doc["version"].(float64); ok {
		version = int(v)
	}
	if version < 1 {
		return Settings{}, fmt.Errorf("unsupported settings version %d", version)
	}
	// A newer version (app downgrade) is read best-effort: unknown fields are ignored.
	for v := version; v < CurrentVersion; v++ {
		if err := migrations[v-1](doc); err != nil {
			return Settings{}, fmt.Errorf("migrate settings v%d: %w", v, err)
		}
	}

	migrated, err := json.Marshal(doc)
	if err != nil {
		return Settings{}, err
	}
	settings := Default()
	if err := json.Unmarshal(migrated, &settings); err != nil {
		return Settings{}, err
	}
	settings.Version = CurrentVersion
	return settings, nil
}

// Save validates settings and writes them atomically (temp file + rename), so a
// crash mid-write never leaves a truncated config.json behind.
func (s *Store) Save(settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	settings.Version = CurrentVersion
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, fileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp settings file: %w", err)
	}
	// Cleanup on the error paths below; after a successful rename this is a no-op.
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write settings: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close settings: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.Path()); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}
