package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"

	"focuswidget/internal/logfile"
)

const (
	logMaxSize = 1 << 20 // SPEC §4: the last 5 files of 1 MB
	logKeep    = 5
)

// newLoggers write to <dir>\logs\app.log, and in dev builds to stderr too. A
// release build has no console, so a crash report goes to logs\crash.log.
// If the log can't be opened the app still runs, logging to stderr only.
// Wails gets its own logger at Warn: at Info it logs every asset request.
func newLoggers(dir string) (logger, wailsLogger *slog.Logger) {
	logs := filepath.Join(dir, "logs")
	var out []io.Writer
	// Left open: the log lives as long as the process.
	file, openErr := logfile.Open(filepath.Join(logs, "app.log"), logMaxSize, logKeep)
	if openErr == nil {
		out = append(out, file)
	}
	if logToConsole || openErr != nil {
		out = append(out, os.Stderr)
	}
	w := io.MultiWriter(out...)
	logger = slog.New(slog.NewTextHandler(w, nil))
	wailsLogger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if openErr != nil {
		logger.Error("open log file", "err", openErr)
	}

	if f, err := os.OpenFile(filepath.Join(logs, "crash.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
			logger.Warn("set crash output", "err", err)
		}
		_ = f.Close() // SetCrashOutput keeps its own duplicate
	}
	return logger, wailsLogger
}
