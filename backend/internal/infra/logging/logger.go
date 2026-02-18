// DesignDocument/7_BackendDesign.md 7.8.1 構造化ログ（Cloud Run向け）
package logging

import (
	"log/slog"
	"os"
)

// Setup initializes the structured JSON logger for Cloud Run.
// Outputs to stdout for Cloud Logging collection.
func Setup(release string) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler).With("release", release)
	slog.SetDefault(logger)
}
