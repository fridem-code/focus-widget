// Package catalog holds the music genres (docs/SPEC.md §3.2, §5.3): the
// catalog embedded in the binary and an optional user override, genres.json.
package catalog

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// OverrideFileName is looked up in the settings directory; when present and
// valid it replaces the embedded catalog entirely.
const OverrideFileName = "genres.json"

type SourceType string

const (
	SourcePlaylist SourceType = "playlist"
	// SourceVideo is a single video or a live stream.
	SourceVideo SourceType = "video"
)

type Source struct {
	Type  SourceType `json:"type"`
	ID    string     `json:"id"`
	Title string     `json:"title,omitempty"`
}

type Genre struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Sources []Source `json:"sources"`
}

type Catalog struct {
	Genres []Genre `json:"genres"`
}

//go:embed default_genres.json
var defaultJSON []byte

var (
	videoIDRe    = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	playlistIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{10,64}$`)
)

// Default returns the embedded catalog. It panics if the embedded file is
// invalid, which TestDefaultCatalogIsValid guards against.
func Default() Catalog {
	c, err := Parse(defaultJSON)
	if err != nil {
		panic(fmt.Sprintf("embedded default_genres.json: %v", err))
	}
	return c
}

// Parse decodes and validates a catalog document.
func Parse(data []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return Catalog{}, err
	}
	if err := c.Validate(); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// Load returns the override from dir/genres.json when it exists and is valid,
// otherwise the embedded catalog. A broken override is logged, never fatal.
func Load(dir string, logger *slog.Logger) Catalog {
	path := filepath.Join(dir, OverrideFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default()
	}
	if err != nil {
		logger.Warn("cannot read genre override, using built-in genres", "path", path, "err", err)
		return Default()
	}
	c, err := Parse(data)
	if err != nil {
		logger.Warn("invalid genre override, using built-in genres", "path", path, "err", err)
		return Default()
	}
	return c
}

// Validate checks that there is at least one genre, genre ids are unique and
// non-empty, and every genre has at least one well-formed source.
func (c Catalog) Validate() error {
	if len(c.Genres) == 0 {
		return errors.New("catalog has no genres")
	}
	var problems []string
	seen := map[string]bool{}
	for i, g := range c.Genres {
		where := fmt.Sprintf("genres[%d]", i)
		if g.ID == "" {
			problems = append(problems, where+": empty id")
		} else if seen[g.ID] {
			problems = append(problems, fmt.Sprintf("%s: duplicate id %q", where, g.ID))
		}
		seen[g.ID] = true
		if strings.TrimSpace(g.Name) == "" {
			problems = append(problems, where+": empty name")
		}
		if len(g.Sources) == 0 {
			problems = append(problems, where+": no sources")
		}
		for j, s := range g.Sources {
			if err := s.validate(); err != nil {
				problems = append(problems, fmt.Sprintf("%s.sources[%d]: %v", where, j, err))
			}
		}
	}
	if len(problems) > 0 {
		return errors.New("invalid catalog: " + strings.Join(problems, "; "))
	}
	return nil
}

func (s Source) validate() error {
	switch s.Type {
	case SourceVideo:
		if !videoIDRe.MatchString(s.ID) {
			return fmt.Errorf("video id %q must be 11 characters of [A-Za-z0-9_-]", s.ID)
		}
	case SourcePlaylist:
		if !playlistIDRe.MatchString(s.ID) {
			return fmt.Errorf("playlist id %q must be 10-64 characters of [A-Za-z0-9_-]", s.ID)
		}
	default:
		return fmt.Errorf("unknown source type %q (want playlist|video)", s.Type)
	}
	return nil
}

// Genre returns the genre with the given id.
func (c Catalog) Genre(id string) (Genre, bool) {
	for _, g := range c.Genres {
		if g.ID == id {
			return g, true
		}
	}
	return Genre{}, false
}
