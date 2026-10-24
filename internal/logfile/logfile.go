// Package logfile writes the app log to a file that rotates by size
// (docs/SPEC.md §4: the last 5 files of 1 MB).
package logfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Writer appends to path and, when a write would take the file past maxSize,
// shifts path → path.1 → … → path.<keep-1> first, dropping the oldest. A single
// write larger than maxSize still goes into one file.
type Writer struct {
	path    string
	maxSize int64
	keep    int

	mu   sync.Mutex
	file *os.File
	size int64
}

// Open creates the directory if needed and appends to the existing log.
// keep counts all files, the current one included, and must be at least 1.
func Open(path string, maxSize int64, keep int) (*Writer, error) {
	if keep < 1 || maxSize <= 0 {
		return nil, fmt.Errorf("logfile: keep %d and maxSize %d must be positive", keep, maxSize)
	}
	w := &Writer{path: path, maxSize: maxSize, keep: keep}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat log: %w", err)
	}
	w.file, w.size = f, info.Size()
	return nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, fs.ErrClosed
	}
	if w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// rotate closes the current file, shifts the backups and starts a new file.
// Windows can't rename an open file, hence the close first.
func (w *Writer) rotate() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close log: %w", err)
	}
	w.file = nil
	name := func(i int) string {
		if i == 0 {
			return w.path
		}
		return fmt.Sprintf("%s.%d", w.path, i)
	}
	if err := os.Remove(name(w.keep - 1)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove old log: %w", err)
	}
	for i := w.keep - 2; i >= 0; i-- {
		if err := os.Rename(name(i), name(i+1)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("rotate log: %w", err)
		}
	}
	return w.open()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
