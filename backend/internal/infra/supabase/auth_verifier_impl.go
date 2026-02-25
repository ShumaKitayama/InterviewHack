// DesignDocument/7_BackendDesign.md 7.4 認証・認可
package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/interviewhack/backend/internal/port/external"
)

type authVerifierImpl struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type authUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewAuthVerifier(baseURL, publishableKey, anonKey string) external.AuthVerifier {
	apiKey := publishableKey
	if apiKey == "" {
		apiKey = anonKey
	}

	return &authVerifierImpl{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (v *authVerifierImpl) VerifyToken(ctx context.Context, token string) (*external.AuthClaims, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	if v.baseURL == "" || v.apiKey == "" {
		return nil, fmt.Errorf("auth verifier is not configured: SUPABASE_URL and SUPABASE_PUBLISHABLE_KEY (or SUPABASE_ANON_KEY) are required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, fmt.Errorf("build auth verify request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", v.apiKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request supabase auth user endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase auth rejected token: status=%d", resp.StatusCode)
	}

	body := io.LimitReader(resp.Body, 1<<20)
	var user authUserResponse
	if err := json.NewDecoder(body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode supabase auth user response: %w", err)
	}
	if user.ID == "" {
		return nil, fmt.Errorf("supabase auth response does not contain user id")
	}

	return &external.AuthClaims{
		UserID: user.ID,
		Email:  user.Email,
	}, nil
}

var _ external.AuthVerifier = (*authVerifierImpl)(nil)
