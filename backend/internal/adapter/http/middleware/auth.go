// DesignDocument/7_BackendDesign.md 7.4 認証・認可
// Verifies the Firebase ID token, resolves the users row and stores user_id.
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/external"
)

// UserIDKey is the gin context key holding the verified user id.
const UserIDKey = "user_id"

// UserResolver resolves verified claims to a user (account use case).
type UserResolver interface {
	Resolve(ctx context.Context, claims external.Claims) (entity.User, error)
}

// Auth rejects requests without a valid token (401) and sets user_id.
// The user id is taken only from the verified token, never from the request body.
func Auth(verifier external.AuthVerifier, users UserResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
			presenter.RespondError(c, domainerror.ErrUnauthorized())
			return
		}
		claims, err := verifier.VerifyToken(c.Request.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			presenter.RespondError(c, domainerror.ErrUnauthorized())
			return
		}
		u, err := users.Resolve(c.Request.Context(), claims)
		if err != nil {
			presenter.RespondError(c, err)
			return
		}
		c.Set(UserIDKey, u.ID)
		c.Next()
	}
}
