package httpapi

import (
	"bytes"
	"io"
	"log/slog"
)

// testLogger returns a slog.Logger that discards its output, for tests
// that need a *Server but don't assert on log content.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newJSONBody wraps a raw JSON payload as an io.Reader suitable for
// httptest.NewRequest's body parameter.
func newJSONBody(payload []byte) *bytes.Reader {
	return bytes.NewReader(payload)
}
