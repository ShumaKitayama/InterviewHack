// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/qa-generate
package ai

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type QAGenerateInput struct {
	UserID        string
	CompanyID     string
	JobTitle      string
	AssetIDs      []string
	QuestionCount int
}

type QAGenerateOutput struct {
	GenerationID string
	RawContent   string
}

type QAGenerateUsecase struct {
	base *BaseAIUsecase
}

func NewQAGenerateUsecase(base *BaseAIUsecase) *QAGenerateUsecase {
	return &QAGenerateUsecase{base: base}
}

func (u *QAGenerateUsecase) Execute(ctx context.Context, input *QAGenerateInput) (*QAGenerateOutput, error) {
	if input.QuestionCount <= 0 {
		input.QuestionCount = 10
	}

	prompt := fmt.Sprintf(
		"企業ID: %s, 職種: %s に対する想定面接Q&Aを%d件生成してください。",
		input.CompanyID, input.JobTitle, input.QuestionCount,
	)

	snapshot := map[string]any{
		"company_id":     input.CompanyID,
		"job_title":      input.JobTitle,
		"asset_ids":      input.AssetIDs,
		"question_count": input.QuestionCount,
	}

	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeQAGenerate.String(),
		vo.JobTypeQAGenerate, snapshot, prompt, "gpt-4o-mini", 4096,
	)
	if err != nil {
		return nil, err
	}

	return &QAGenerateOutput{
		GenerationID: result.GenerationID,
		RawContent:   result.Response.Content,
	}, nil
}
