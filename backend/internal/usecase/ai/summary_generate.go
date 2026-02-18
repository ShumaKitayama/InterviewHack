// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/summary-generate
package ai

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type SummaryGenerateInput struct {
	UserID          string
	InterviewID     string
	InterviewLogIDs []string
}

type SummaryGenerateOutput struct {
	GenerationID string
	RawContent   string
}

type SummaryGenerateUsecase struct {
	base *BaseAIUsecase
}

func NewSummaryGenerateUsecase(base *BaseAIUsecase) *SummaryGenerateUsecase {
	return &SummaryGenerateUsecase{base: base}
}

func (u *SummaryGenerateUsecase) Execute(ctx context.Context, input *SummaryGenerateInput) (*SummaryGenerateOutput, error) {
	prompt := "直近の面接ログから、次回直すべきポイントを3〜5個に要約してください。"

	snapshot := map[string]any{
		"interview_id":      input.InterviewID,
		"interview_log_ids": input.InterviewLogIDs,
	}

	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeSummaryGenerate.String(),
		vo.JobTypeSummaryGenerate, snapshot, prompt, "gpt-4o-mini", 2048,
	)
	if err != nil {
		return nil, err
	}

	return &SummaryGenerateOutput{
		GenerationID: result.GenerationID,
		RawContent:   result.Response.Content,
	}, nil
}
