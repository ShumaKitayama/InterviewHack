// DesignDocument/11_TestDesign.md UT-SH-001
package system

import (
	"encoding/base64"
	"testing"
)

func TestTokenGenerator(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok, err := TokenGenerator{}.Generate()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(tok)
		if err != nil {
			t.Fatalf("token is not base64url: %v", err)
		}
		if len(raw) != 32 {
			t.Fatalf("token has %d bytes, want 32 (256 bits)", len(raw))
		}
		if seen[tok] {
			t.Fatal("duplicate token")
		}
		seen[tok] = true
	}
}
