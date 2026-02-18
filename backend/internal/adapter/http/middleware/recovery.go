// DesignDocument/7_BackendDesign.md 7.7
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				requestID, _ := c.Get("request_id")
				slog.Error("panic recovered",
					"panic", r,
					"request_id", requestID,
					"path", c.Request.URL.Path,
				)
				c.JSON(http.StatusInternalServerError, presenter.ErrorBody{
					Status:  "error",
					Code:    "INTERNAL_ERROR",
					Message: "内部エラーが発生しました",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
