// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/es-refine
package ai

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type ESRefineInput struct {
	UserID       string
	CurrentText  string
	Instructions string
}

type ESRefineOutput struct {
	GenerationID string
	RawContent   string
}

type ESRefineUsecase struct {
	base *BaseAIUsecase
}

func NewESRefineUsecase(base *BaseAIUsecase) *ESRefineUsecase {
	return &ESRefineUsecase{base: base}
}

func (u *ESRefineUsecase) Execute(ctx context.Context, input *ESRefineInput) (*ESRefineOutput, error) {
	prompt := fmt.Sprintf(
		"以下のES本文を修正指示に従ってブラッシュアップしてください。\n本文: %s\n修正指示: %s",
		input.CurrentText, input.Instructions,
	)

	snapshot := map[string]any{
		"current_text": "(omitted for log safety)",
		"instructions": input.Instructions,
	}

	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeESRefine.String(),
		vo.JobTypeESRefine, snapshot, prompt, "gpt-4o-mini", 4096,
	)
	if err != nil {
		return nil, err
	}

	return &ESRefineOutput{
		GenerationID: result.GenerationID,
		RawContent:   result.Response.Content,
	}, nil
}
