package localmusic

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsMusicInSubfoldersInPathOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.mp3", "a.FLAC", "notes.txt", "cover.jpg", `album\01 Intro.m4a`, `album\02.opus`} {
		touch(t, filepath.Join(dir, name))
	}
	tracks, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, tr := range tracks {
		titles = append(titles, tr.Title)
		if !filepath.IsAbs(tr.Path) {
			t.Fatalf("path %q is not absolute", tr.Path)
		}
	}
	want := []string{"a", "01 Intro", "02", "b"}
	if len(titles) != len(want) {
		t.Fatalf("titles = %q, want %q", titles, want)
	}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("titles = %q, want %q", titles, want)
		}
	}
}

func TestScanMissingFolderIsAnError(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Fatal("no error for a missing folder")
	}
}

func TestScanEmptyFolder(t *testing.T) {
	tracks, err := Scan(t.TempDir())
	if err != nil || len(tracks) != 0 {
		t.Fatalf("Scan = %v, %v", tracks, err)
	}
}
