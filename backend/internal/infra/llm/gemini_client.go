// DesignDocument/7_BackendDesign.md 7.2.3 infra/llm
// DesignDocument/4_LLMDesign.md 4.4 モデル選定方針
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

func (c *geminiClient) Generate(_ context.Context, _ *external.LLMRequest) (*external.LLMResponse, error) {
	return nil, fmt.Errorf("gemini client not yet implemented")
}
