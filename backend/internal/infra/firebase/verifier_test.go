// DesignDocument/11_TestDesign.md IT-AUTH-007（infra/firebase 単体）/ IT-AUTH-003 / 004
package firebase

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type staticKeys map[string]*rsa.PublicKey

func (s staticKeys) Keys(context.Context) (map[string]*rsa.PublicKey, error) { return s, nil }

func sign(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	head := enc(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	body := enc(claims)
	digest := sha256.Sum256([]byte(head + "." + body))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return head + "." + body + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	v := &Verifier{ProjectID: "ih-dev", Keys: staticKeys{"k1": &key.PublicKey}, Now: func() time.Time { return now }, Skew: time.Minute}
	base := func() map[string]any {
		return map[string]any{
			"iss": "https://securetoken.google.com/ih-dev", "aud": "ih-dev", "sub": "uid-1",
			"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(time.Hour).Unix(), "auth_time": now.Add(-time.Hour).Unix(),
			"email": "u1@example.com",
		}
	}
	with := func(k string, val any) map[string]any { c := base(); c[k] = val; return c }

	tok := sign(t, key, "k1", base())
	c, err := v.VerifyToken(context.Background(), tok)
	if err != nil || c.UID != "uid-1" || c.Email == nil || *c.Email != "u1@example.com" {
		t.Fatalf("valid token rejected: %+v %v", c, err)
	}

	bad := map[string]string{
		"unknown kid":      sign(t, key, "k2", base()),
		"wrong signature":  sign(t, other, "k1", base()),
		"aud mismatch":     sign(t, key, "k1", with("aud", "other")),
		"iss mismatch":     sign(t, key, "k1", with("iss", "https://securetoken.google.com/other")),
		"expired":          sign(t, key, "k1", with("exp", now.Add(-time.Second).Unix())),
		"issued in future": sign(t, key, "k1", with("iat", now.Add(time.Hour).Unix())),
		"empty sub":        sign(t, key, "k1", with("sub", "")),
		"tampered payload": tok[:len(tok)/2] + "x" + tok[len(tok)/2+1:],
		"not a jwt":        "abc",
	}
	for name, token := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := v.VerifyToken(context.Background(), token); err == nil {
				t.Fatal("must be rejected")
			}
		})
	}
}

func TestHTTPKeySource(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]string{"k1": certPEM})
	}))
	defer srv.Close()
	src := &HTTPKeySource{URL: srv.URL, Client: srv.Client()}
	for i := 0; i < 3; i++ {
		keys, err := src.Keys(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if keys["k1"] == nil || keys["k1"].N.Cmp(key.N) != 0 {
			t.Fatal("key mismatch")
		}
	}
	if hits != 1 {
		t.Fatalf("certs must be cached by max-age, fetched %d times", hits)
	}
}
