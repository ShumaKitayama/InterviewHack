// Package external defines interfaces to the outside world (auth, randomness, time).
// DesignDocument/7_BackendDesign.md 7.2.3 / 7.4 / 7.6.2
package external

import (
	"context"
	"time"
)

// Claims are the verified claims of a Firebase ID token.
type Claims struct {
	UID   string
	Email *string
}

// AuthVerifier verifies Firebase ID tokens (implementation: infra/firebase).
type AuthVerifier interface {
	VerifyToken(ctx context.Context, token string) (Claims, error)
}

// TokenGenerator generates share-link tokens from a CSPRNG.
type TokenGenerator interface {
	Generate() (string, error)
}

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// IDGenerator returns new uuid strings.
type IDGenerator interface {
	NewID() string
}
