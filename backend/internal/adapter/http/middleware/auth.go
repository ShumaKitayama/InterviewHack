// DesignDocument/7_BackendDesign.md 7.4 認証・認可
// BFF は Supabase JWT を検証し、user_id を確定する
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/external"
)

func Auth(verifier external.AuthVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			presenter.RespondError(c, domainerror.ErrUnauthorized())
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			presenter.RespondError(c, domainerror.ErrUnauthorized())
			c.Abort()
			return
		}

		claims, err := verifier.VerifyToken(c.Request.Context(), parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, presenter.ErrorBody{
				Status:  "error",
				Code:    "UNAUTHORIZED",
				Message: "認証トークンが無効です",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}
