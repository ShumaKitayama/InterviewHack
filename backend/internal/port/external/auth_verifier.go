// DesignDocument/7_BackendDesign.md 7.4 認証・認可
package external

import "context"

type AuthClaims struct {
	UserID string
	Email  string
}

type AuthVerifier interface {
	// VerifyToken validates a Supabase JWT and extracts claims.
	// user_id is always derived from JWT sub, never from request body.
	VerifyToken(ctx context.Context, token string) (*AuthClaims, error)
}
