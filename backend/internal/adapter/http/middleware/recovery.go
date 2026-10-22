// DesignDocument/7_BackendDesign.md 7.7
package middleware

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/presenter"
)

// Recovery turns panics into 500 INTERNAL_ERROR without leaking details.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "request_id", c.GetString("request_id"), "path", c.FullPath(), "panic", fmt.Sprint(r))
				presenter.RespondError(c, fmt.Errorf("panic"))
			}
		}()
		c.Next()
	}
}
