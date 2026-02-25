// DesignDocument/7_BackendDesign.md 7.5.4 AI
// DesignDocument/4_LLMDesign.md 4.3.1 ロール分離
package external

import "context"

type LLMMessage struct {
	Role    string // "system", "user"
	Content string
}

type LLMRequest struct {
	Messages  []LLMMessage
	Model     string
	MaxTokens int
	JSONMode  bool
}

type LLMResponse struct {
	Content      string
	TokensUsed   int
	CostEstimate float64
	Model        string
}

type LLMClient interface {
	Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}
