// DesignDocument/7_BackendDesign.md 7.8.1 構造化ログ
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/presenter"
)

// Logger writes one JSON line per request. The path is the route pattern
// (no ids or share tokens), and no bodies or headers are logged.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		attrs := []any{
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		}
		if uid := c.GetString(UserIDKey); uid != "" {
			attrs = append(attrs, "user_id", uid)
		}
		if code := c.GetString(presenter.ErrorCodeKey); code != "" {
			attrs = append(attrs, "error_code", code)
		}
		for _, k := range []string{"session_id", "use_case", "template_version", "detected_template_version", "parse_result", "response_length", "share_link_id", "share_result"} {
			if v, ok := c.Get(k); ok {
				attrs = append(attrs, k, v)
			}
		}
		switch status := c.Writer.Status(); {
		case status >= 500:
			slog.Error("request completed", attrs...)
		case status >= 400:
			slog.Warn("request completed", attrs...)
		default:
			slog.Info("request completed", attrs...)
		}
	}
}
