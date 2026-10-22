// Package system implements the randomness, clock and id ports.
// DesignDocument/7_BackendDesign.md 7.6.2（TokenGenerator）
package system

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"

	"github.com/interviewhack/backend/internal/domain/service/share"
)

// TokenGenerator generates 32-byte CSPRNG tokens encoded as base64url.
type TokenGenerator struct{}

// Generate returns a new raw share token.
func (TokenGenerator) Generate() (string, error) {
	b := make([]byte, share.TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Clock is the real clock.
type Clock struct{}

// Now returns the current time.
func (Clock) Now() time.Time { return time.Now() }

// IDGenerator returns uuid v4 strings.
type IDGenerator struct{}

// NewID returns a new uuid.
func (IDGenerator) NewID() string { return uuid.NewString() }
