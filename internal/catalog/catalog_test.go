package catalog

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var discard = slog.New(slog.DiscardHandler)

func TestDefaultCatalogIsValid(t *testing.T) {
	c := Default()
	if _, ok := c.Genre("lofi"); !ok {
		t.Fatal(`default catalog lacks the "lofi" genre that config.Default() selects`)
	}
	// The four genres of the design (SPEC §5.3); whether each source actually
	// plays is checked by scripts/catalog-check, which needs the real player.
	var ids []string
	for _, g := range c.Genres {
		ids = append(ids, g.ID)
	}
	if want := []string{"lofi", "ambient", "classical", "noise"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("genres = %v, want %v", ids, want)
	}
}

func TestValidate(t *testing.T) {
	video := Source{Type: SourceVideo, ID: "CFGLoQIhmow"}
	playlist := Source{Type: SourcePlaylist, ID: "PLx0sYbCqOb8TBPRdmBHs5Iftvv9TPboYG"}
	tests := []struct {
		name    string
		catalog Catalog
		wantErr string // empty means valid
	}{
		{"valid", Catalog{Genres: []Genre{{ID: "a", Name: "A", Sources: []Source{video, playlist}}}}, ""},
		{"generated noise", Catalog{Genres: []Genre{{ID: "n", Name: "N", Sources: []Source{
			{Type: SourceNoise, ID: "white"}, {Type: SourceNoise, ID: "rain"}}}}}, ""},
		{"unknown noise", Catalog{Genres: []Genre{{ID: "n", Name: "N", Sources: []Source{{Type: SourceNoise, ID: "blue"}}}}},
			`noise "blue" must be one of white|pink|brown|rain`},
		{"no genres", Catalog{}, "no genres"},
		{"empty id", Catalog{Genres: []Genre{{Name: "A", Sources: []Source{video}}}}, "empty id"},
		{"duplicate id", Catalog{Genres: []Genre{
			{ID: "a", Name: "A", Sources: []Source{video}},
			{ID: "a", Name: "B", Sources: []Source{video}},
		}}, `duplicate id "a"`},
		{"blank name", Catalog{Genres: []Genre{{ID: "a", Name: "  ", Sources: []Source{video}}}}, "empty name"},
		{"no sources", Catalog{Genres: []Genre{{ID: "a", Name: "A"}}}, "no sources"},
		{"unknown type", Catalog{Genres: []Genre{{ID: "a", Name: "A", Sources: []Source{{Type: "radio", ID: "x"}}}}},
			"unknown source type"},
		{"short video id", Catalog{Genres: []Genre{{ID: "a", Name: "A", Sources: []Source{{Type: SourceVideo, ID: "abc"}}}}},
			"video id"},
		{"url instead of id", Catalog{Genres: []Genre{{ID: "a", Name: "A",
			Sources: []Source{{Type: SourcePlaylist, ID: "https://youtube.com/playlist?list=PL1"}}}}}, "playlist id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.catalog.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	override := `{"genres": [{"id": "rain", "name": "Rain", "sources": [{"type": "video", "id": "abcdefghijk"}]}]}`
	tests := []struct {
		name    string
		content *string // nil: no override file
		want    Catalog
	}{
		{"no override", nil, Default()},
		{"valid override replaces default", &override, Catalog{Genres: []Genre{
			{ID: "rain", Name: "Rain", Sources: []Source{{Type: SourceVideo, ID: "abcdefghijk"}}},
		}}},
		{"broken json falls back", new(`{"genres": [`), Default()},
		{"invalid catalog falls back", new(`{"genres": []}`), Default()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.content != nil {
				if err := os.WriteFile(filepath.Join(dir, OverrideFileName), []byte(*tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := Load(dir, discard); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGenre(t *testing.T) {
	c := Catalog{Genres: []Genre{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}}
	if g, ok := c.Genre("b"); !ok || g.Name != "B" {
		t.Fatalf("Genre(b) = %+v, %v", g, ok)
	}
	if _, ok := c.Genre("zzz"); ok {
		t.Fatal("Genre(zzz) found a genre")
	}
}
