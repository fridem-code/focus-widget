package logfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func write(t *testing.T, w *Writer, s string) {
	t.Helper()
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
}

func TestAppendsToExistingLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "app.log")
	w, err := Open(path, 100, 3)
	if err != nil {
		t.Fatal(err)
	}
	write(t, w, "one\n")
	_ = w.Close()

	w, err = Open(path, 100, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()
	write(t, w, "two\n")
	if got := read(t, path); got != "one\ntwo\n" {
		t.Fatalf("log = %q", got)
	}
}

func TestRotatesAndKeepsTheNewestFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := Open(path, 10, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()

	for _, line := range []string{"aaaaaaaa\n", "bbbbbbbb\n", "cccccccc\n", "dddddddd\n"} {
		write(t, w, line)
	}
	if got := read(t, path); got != "dddddddd\n" {
		t.Fatalf("app.log = %q", got)
	}
	if got := read(t, path+".1"); got != "cccccccc\n" {
		t.Fatalf("app.log.1 = %q", got)
	}
	if got := read(t, path+".2"); got != "bbbbbbbb\n" {
		t.Fatalf("app.log.2 = %q", got)
	}
	if _, err := os.Stat(path + ".3"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("app.log.3 exists: %v", err)
	}
}

func TestWriteLargerThanMaxGoesIntoOneFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := Open(path, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()

	big := strings.Repeat("x", 25) + "\n"
	write(t, w, big)
	write(t, w, "small\n")
	if got := read(t, path+".1"); got != big {
		t.Fatalf("app.log.1 = %q", got)
	}
	if got := read(t, path); got != "small\n" {
		t.Fatalf("app.log = %q", got)
	}
}

func TestConcurrentWritesKeepLinesWhole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := Open(path, 1<<20, 2)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Repeat("y", 50) + "\n"
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 10 {
				if _, err := w.Write([]byte(line)); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	_ = w.Close()
	if got := read(t, path); got != strings.Repeat(line, 200) {
		t.Fatalf("log has %d bytes, want %d whole lines", len(got), 200)
	}
}

func TestWriteAfterCloseFails(t *testing.T) {
	w, err := Open(filepath.Join(t.TempDir(), "app.log"), 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	if _, err := w.Write([]byte("x")); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("Write after Close = %v", err)
	}
}

func TestOpenRejectsBadLimits(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "app.log"), 10, 0); err == nil {
		t.Fatal("keep 0 accepted")
	}
}
