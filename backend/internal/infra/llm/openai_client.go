// DesignDocument/7_BackendDesign.md 7.5.4 / 7.2.3 infra/llm
// DesignDocument/4_LLMDesign.md 4.3.1 ロール分離, 4.4 モデル選定
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

	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	chatReq := openai.ChatCompletionRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
	}

	if req.JSONMode {
		chatReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
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
