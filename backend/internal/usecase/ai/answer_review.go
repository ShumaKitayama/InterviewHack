// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/answer-review
package ai

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type AnswerReviewInput struct {
	UserID       string
	QuestionText string
	AnswerText   string
	AssetIDs     []string
	Strictness   string
}

type AnswerReviewOutput struct {
	GenerationID string
	RawContent   string
}

type AnswerReviewUsecase struct {
	base *BaseAIUsecase
}

func NewAnswerReviewUsecase(base *BaseAIUsecase) *AnswerReviewUsecase {
	return &AnswerReviewUsecase{base: base}
}

func (u *AnswerReviewUsecase) Execute(ctx context.Context, input *AnswerReviewInput) (*AnswerReviewOutput, error) {
	if input.Strictness == "" {
		input.Strictness = "normal"
	}

	prompt := fmt.Sprintf(
		"以下の面接回答を添削してください。\n質問: %s\n回答: %s\n厳しさ: %s",
		input.QuestionText, input.AnswerText, input.Strictness,
	)

	snapshot := map[string]any{
		"question_text": input.QuestionText,
		"answer_text":   input.AnswerText,
		"asset_ids":     input.AssetIDs,
		"strictness":    input.Strictness,
	}

	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeAnswerReview.String(),
		vo.JobTypeAnswerReview, snapshot, prompt, "gpt-4o-mini", 4096,
	)
	if err != nil {
		return nil, err
	}

	return &AnswerReviewOutput{
		GenerationID: result.GenerationID,
		RawContent:   result.Response.Content,
	}, nil
}
