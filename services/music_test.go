package services

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"focuswidget/internal/catalog"
)

// musicFolder makes a folder with two tracks and a non-music file.
func musicFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"a.mp3": "first track", "b.ogg": "second", "notes.txt": "x"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLocalGenreFollowsTheMusicFolder(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	dir := musicFolder(t)
	picked := dir
	svc := NewMusicService(catalog.Default(), settings, func() (string, error) { return picked, nil })

	if got := svc.Genres(); len(got.Genres) != len(catalog.Default().Genres) {
		t.Fatalf("genres without a folder = %d", len(got.Genres))
	}
	if folder, err := svc.ChooseFolder(); err != nil || folder != dir || settings.Get().Music.LocalFolder != dir {
		t.Fatalf("ChooseFolder = %q, %v; saved %q", folder, err, settings.Get().Music.LocalFolder)
	}
	genres := svc.Genres().Genres
	local := genres[len(genres)-1]
	if local.ID != LocalGenreID || local.Name != "Мои файлы" || len(local.Sources) != 2 {
		t.Fatalf("local genre = %+v", local)
	}
	if s := local.Sources[1]; s.Type != catalog.SourceFile || s.ID != "1" || s.Title != "b" {
		t.Fatalf("second source = %+v", s)
	}
	if err := svc.Select(LocalGenreID, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.Select(LocalGenreID, 2); err == nil {
		t.Fatal("Select accepted a track past the folder's end")
	}

	picked = "" // cancelled: the folder stays
	if folder, err := svc.ChooseFolder(); err != nil || folder != "" || settings.Get().Music.LocalFolder != dir {
		t.Fatalf("cancelled ChooseFolder = %q, %v; saved %q", folder, err, settings.Get().Music.LocalFolder)
	}
	if err := svc.ClearFolder(); err != nil {
		t.Fatal(err)
	}
	if got := svc.Genres(); len(got.Genres) != len(catalog.Default().Genres) {
		t.Fatal("Мои файлы still listed after ClearFolder")
	}
}

func TestLocalGenreOfAMissingFolderHasNoSources(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	gone := filepath.Join(t.TempDir(), "gone")
	svc := NewMusicService(catalog.Default(), settings, func() (string, error) { return gone, nil })
	if _, err := svc.ChooseFolder(); err != nil {
		t.Fatal(err)
	}
	genres := svc.Genres().Genres
	if local := genres[len(genres)-1]; local.ID != LocalGenreID || len(local.Sources) != 0 {
		t.Fatalf("local genre = %+v, want no sources", local)
	}
}

func TestLocalMusicServesOnlyTheScannedFiles(t *testing.T) {
	settings, _, _, _ := newSettingsService(t)
	dir := musicFolder(t)
	svc := NewMusicService(catalog.Default(), settings, func() (string, error) { return dir, nil })
	if _, err := svc.ChooseFolder(); err != nil {
		t.Fatal(err)
	}
	svc.Genres() // scans
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "asset") })
	h := LocalMusic(svc, next)
	get := func(path, rangeHeader string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if rangeHeader != "" {
			r.Header.Set("Range", rangeHeader)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	if w := get("/local-music/0", ""); w.Code != 200 || w.Body.String() != "first track" || w.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("track 0: %d %q %q", w.Code, w.Body.String(), w.Header().Get("Content-Type"))
	}
	if w := get("/local-music/0", "bytes=6-"); w.Code != http.StatusPartialContent || w.Body.String() != "track" {
		t.Fatalf("range: %d %q", w.Code, w.Body.String())
	}
	for _, path := range []string{"/local-music/2", "/local-music/-1", "/local-music/../notes.txt", "/local-music/x"} {
		if w := get(path, ""); w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d, want 404", path, w.Code)
		}
	}
	if w := get("/index.html", ""); w.Body.String() != "asset" {
		t.Fatal("other paths must go to the asset server")
	}
}
