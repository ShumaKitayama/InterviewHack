// DesignDocument/7_BackendDesign.md 7.8.1 構造化ログ
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		requestID, _ := c.Get("request_id")
		userID, _ := c.Get("user_id")

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"request_id", requestID,
		}

		if uid, ok := userID.(string); ok && uid != "" {
			attrs = append(attrs, "user_id", uid)
		}

		if c.Writer.Status() >= 500 {
			slog.Error("request completed", attrs...)
		} else if c.Writer.Status() >= 400 {
			slog.Warn("request completed", attrs...)
		} else {
			slog.Info("request completed", attrs...)
		}
	}
}
