// Package firebase verifies Firebase ID tokens with Google's public keys.
// No service-account key is needed (14.8.1).
// DesignDocument/7_BackendDesign.md 7.4 / 9_SecurityDesign.md
package firebase

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/interviewhack/backend/internal/port/external"
)

// CertsURL publishes the x509 certificates that sign Firebase ID tokens.
const CertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// ErrInvalidToken is returned for any token that fails verification.
var ErrInvalidToken = errors.New("invalid firebase id token")

// KeySource returns the current signing keys by key ID.
type KeySource interface {
	Keys(ctx context.Context) (map[string]*rsa.PublicKey, error)
}

// Verifier implements external.AuthVerifier.
type Verifier struct {
	ProjectID string
	Keys      KeySource
	Now       func() time.Time
	// Skew tolerates small clock differences for iat / auth_time.
	Skew time.Duration
}

// NewVerifier returns a verifier that fetches Google's certificates over HTTP.
func NewVerifier(projectID string) *Verifier {
	return &Verifier{
		ProjectID: projectID,
		Keys:      &HTTPKeySource{URL: CertsURL, Client: &http.Client{Timeout: 10 * time.Second}},
		Now:       time.Now,
		Skew:      time.Minute,
	}
}

type header struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type claims struct {
	Iss      string  `json:"iss"`
	Aud      string  `json:"aud"`
	Sub      string  `json:"sub"`
	Exp      int64   `json:"exp"`
	Iat      int64   `json:"iat"`
	AuthTime int64   `json:"auth_time"`
	Email    *string `json:"email"`
}

// VerifyToken checks signature (RS256, kid), aud, iss, exp, iat and sub.
func (v *Verifier) VerifyToken(ctx context.Context, token string) (external.Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || v.ProjectID == "" {
		return external.Claims{}, ErrInvalidToken
	}
	var h header
	if err := decodeSegment(parts[0], &h); err != nil || h.Alg != "RS256" || h.Kid == "" {
		return external.Claims{}, ErrInvalidToken
	}
	keys, err := v.Keys.Keys(ctx)
	if err != nil {
		return external.Claims{}, fmt.Errorf("load signing keys: %w", err)
	}
	key, ok := keys[h.Kid]
	if !ok {
		return external.Claims{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return external.Claims{}, ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return external.Claims{}, ErrInvalidToken
	}
	var c claims
	if err := decodeSegment(parts[1], &c); err != nil {
		return external.Claims{}, ErrInvalidToken
	}
	now := v.Now()
	switch {
	case c.Aud != v.ProjectID,
		c.Iss != "https://securetoken.google.com/"+v.ProjectID,
		c.Sub == "" || len(c.Sub) > 128,
		!now.Before(time.Unix(c.Exp, 0)),
		time.Unix(c.Iat, 0).After(now.Add(v.Skew)),
		c.AuthTime != 0 && time.Unix(c.AuthTime, 0).After(now.Add(v.Skew)):
		return external.Claims{}, ErrInvalidToken
	}
	return external.Claims{UID: c.Sub, Email: c.Email}, nil
}

func decodeSegment(seg string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// HTTPKeySource fetches and caches Google's x509 certificates, honoring Cache-Control max-age.
type HTTPKeySource struct {
	URL    string
	Client *http.Client

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
}

var maxAgeRe = regexp.MustCompile(`max-age=(\d+)`)

// Keys returns the cached keys, refreshing them when expired.
func (s *HTTPKeySource) Keys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys != nil && time.Now().Before(s.expires) {
		return s.keys, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch certs: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	keys, err := ParseCerts(body)
	if err != nil {
		return nil, err
	}
	ttl := time.Hour
	if m := maxAgeRe.FindStringSubmatch(resp.Header.Get("Cache-Control")); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			ttl = time.Duration(n) * time.Second
		}
	}
	s.keys, s.expires = keys, time.Now().Add(ttl)
	return keys, nil
}

// ParseCerts parses the {kid: PEM certificate} document.
func ParseCerts(body []byte) (map[string]*rsa.PublicKey, error) {
	var raw map[string]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode certs: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(raw))
	for kid, p := range raw {
		block, _ := pem.Decode([]byte(p))
		if block == nil {
			return nil, fmt.Errorf("decode pem for %s", kid)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse cert %s: %w", kid, err)
		}
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("cert %s is not RSA", kid)
		}
		keys[kid] = pub
	}
	return keys, nil
}
