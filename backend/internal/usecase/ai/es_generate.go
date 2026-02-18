// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/es-generate
package ai

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type ESGenerateInput struct {
	UserID              string
	ESProjectID         string
	SectionTitle        string
	AssetIDs            []string
	ResearchMaterialIDs []string
	CharacterLimit      int
}

type ESGenerateOutput struct {
	GenerationID string
	RawContent   string
}

type ESGenerateUsecase struct {
	base *BaseAIUsecase
}

func NewESGenerateUsecase(base *BaseAIUsecase) *ESGenerateUsecase {
	return &ESGenerateUsecase{base: base}
}

func (u *ESGenerateUsecase) Execute(ctx context.Context, input *ESGenerateInput) (*ESGenerateOutput, error) {
	prompt := fmt.Sprintf(
		"ESセクション「%s」のドラフトを生成してください。文字数上限: %d文字",
		input.SectionTitle, input.CharacterLimit,
	)

	snapshot := map[string]any{
		"es_project_id":         input.ESProjectID,
		"section_title":         input.SectionTitle,
		"asset_ids":             input.AssetIDs,
		"research_material_ids": input.ResearchMaterialIDs,
		"character_limit":       input.CharacterLimit,
	}

	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeESGenerate.String(),
		vo.JobTypeESGenerate, snapshot, prompt, "gpt-4o-mini", 4096,
	)
	if err != nil {
		return nil, err
	}

	return &ESGenerateOutput{
		GenerationID: result.GenerationID,
		RawContent:   result.Response.Content,
	}, nil
}
