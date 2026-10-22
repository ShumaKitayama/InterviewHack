// DesignDocument/7_BackendDesign.md 7.8.1 構造化ログ（Cloud Run向け）
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the JSON logger for Cloud Run. Keys follow 7.8.1
// (severity, timestamp, ...); bodies, tokens and PII are never logged.
func Setup(release, level string) {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lv,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Key = "severity"
				if a.Value.String() == "WARN" {
					a.Value = slog.StringValue("WARNING")
				}
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler).With("release", release))
}
