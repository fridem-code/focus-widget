package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

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

// harmlessWailsErrors start the Wails errors that break nothing. Logged at
// Debug, so that ERROR in app.log means a real problem.
var harmlessWailsErrors = []string{
	// On every target WebView2 auto-attaches (the YouTube iframe's workers)
	// Wails enables request cancellation over DevTools; WebView2 rejects that
	// for some targets ("The parameter is incorrect"), and Wails then resumes
	// the target anyway. Our asset requests do not come from those targets
	// (docs/BUGS.md B3, Wails pkg/application/request_cancellation_windows.go).
	"Worker request cancellation setup:",
}

// wailsErrorHandler is Options.ErrorHandler: Wails' internal errors go to the
// log, the known harmless ones at Debug.
func wailsErrorHandler(logger *slog.Logger) func(error) {
	return func(err error) {
		msg := err.Error()
		for _, prefix := range harmlessWailsErrors {
			if strings.HasPrefix(msg, prefix) {
				logger.Debug(msg)
				return
			}
		}
		logger.Error(msg)
	}
}
