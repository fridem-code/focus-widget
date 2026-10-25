package app

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestWailsErrorHandlerQuietsHarmlessErrors(t *testing.T) {
	var buf bytes.Buffer
	handle := wailsErrorHandler(slog.New(slog.NewTextHandler(&buf, nil)))

	handle(errors.New("Worker request cancellation setup: The parameter is incorrect."))
	if buf.Len() != 0 {
		t.Fatalf("harmless error logged at Info or above: %q", buf.String())
	}

	handle(errors.New("Disabling Windows request interception: boom"))
	if out := buf.String(); !strings.Contains(out, "level=ERROR") || !strings.Contains(out, "boom") {
		t.Fatalf("real error not logged as ERROR: %q", out)
	}
}
