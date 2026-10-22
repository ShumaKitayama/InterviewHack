package app_test

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

// TestMain silences request logs so test output stays readable.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}
