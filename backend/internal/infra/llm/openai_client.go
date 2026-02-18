// DesignDocument/7_BackendDesign.md 7.5.4 / 7.2.3 infra/llm
package llm

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/port/external"
	openai "github.com/sashabaranov/go-openai"
)

type openaiClient struct {
	client *openai.Client
}

func NewOpenAIClient(apiKey string) external.LLMClient {
	return &openaiClient{
		client: openai.NewClient(apiKey),
	}
}

func (c *openaiClient) Generate(ctx context.Context, req *external.LLMRequest) (*external.LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = openai.GPT4oMini
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: req.Prompt},
		},
		MaxTokens: req.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("openai chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return &external.LLMResponse{
		Content:      resp.Choices[0].Message.Content,
		TokensUsed:   resp.Usage.TotalTokens,
		CostEstimate: 0,
		Model:        resp.Model,
	}, nil
}
