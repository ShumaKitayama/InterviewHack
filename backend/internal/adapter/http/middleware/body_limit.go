// DesignDocument/9_SecurityDesign.md 入力検証（リクエストサイズの上限）
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes bounds request bodies. The largest field is contentRaw
// (200,000 characters, up to ~800 KB of UTF-8).
const MaxBodyBytes = 2 << 20

// BodyLimit wraps the request body with http.MaxBytesReader.
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)
		}
		c.Next()
	}
}
