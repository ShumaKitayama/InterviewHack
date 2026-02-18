// DesignDocument/7_BackendDesign.md 7.2.3 infra/llm
// Gemini client stub - to be fully implemented when Gemini integration is needed
package llm

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/port/external"
)

type geminiClient struct {
	apiKey string
}

func NewGeminiClient(apiKey string) external.LLMClient {
	return &geminiClient{apiKey: apiKey}
}

func (c *geminiClient) Generate(ctx context.Context, req *external.LLMRequest) (*external.LLMResponse, error) {
	// TODO: Implement Gemini API integration
	return nil, fmt.Errorf("gemini client not yet implemented")
}
