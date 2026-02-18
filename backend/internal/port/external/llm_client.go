// DesignDocument/7_BackendDesign.md 7.5.4 AI
package external

import "context"

type LLMRequest struct {
	Prompt    string
	Model     string
	MaxTokens int
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
