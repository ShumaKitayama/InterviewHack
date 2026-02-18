// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/research-summarize
package ai

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type ResearchSummarizeInput struct {
	UserID  string
	Type    string // "url" | "text" | "pdf"
	Content string
}

type ResearchSummarizeOutput struct {
	GenerationID string
	RawContent   string
}

type ResearchSummarizeUsecase struct {
	base *BaseAIUsecase
}

func NewResearchSummarizeUsecase(base *BaseAIUsecase) *ResearchSummarizeUsecase {
	return &ResearchSummarizeUsecase{base: base}
}

func (u *ResearchSummarizeUsecase) Execute(ctx context.Context, input *ResearchSummarizeInput) (*ResearchSummarizeOutput, error) {
	prompt := fmt.Sprintf(
		"以下の企業研究資料(%s)を要約し、重要なポイントと活用提案を出してください。\n内容: %s",
		input.Type, input.Content,
	)

	snapshot := map[string]any{
		"type":    input.Type,
		"content": "(omitted for log safety)",
	}

	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeResearchSummarize.String(),
		vo.JobTypeResearchSummarize, snapshot, prompt, "gpt-4o-mini", 4096,
	)
	if err != nil {
		return nil, err
	}

	return &ResearchSummarizeOutput{
		GenerationID: result.GenerationID,
		RawContent:   result.Response.Content,
	}, nil
}
