// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/es-generate
// DesignDocument/4_LLMDesign.md 4.2.5 ESドラフト生成
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type ESGenerateInput struct {
	UserID              string
	ESProjectID         string
	SectionTitle        string
	AssetIDs            []string
	AssetTexts          []string
	ResearchMaterialIDs []string
	ResearchTexts       []string
	CompanyName         string
	JobTitle            string
	CharacterLimit      int
}

type ESGenerateOutput struct {
	GenerationID string
	DraftText    string
	Rationale    string
}

type esGenerateJSON struct {
	DraftText string `json:"draftText"`
	Rationale string `json:"rationale"`
}

type ESGenerateUsecase struct {
	base *BaseAIUsecase
}

func NewESGenerateUsecase(base *BaseAIUsecase) *ESGenerateUsecase {
	return &ESGenerateUsecase{base: base}
}

func (u *ESGenerateUsecase) Execute(ctx context.Context, input *ESGenerateInput) (*ESGenerateOutput, error) {
	if input.CharacterLimit <= 0 {
		input.CharacterLimit = 400
	}

	var assetsSection string
	if len(input.AssetTexts) > 0 {
		assetsSection = "【自己資産データ（セット1）】\n" + strings.Join(input.AssetTexts, "\n---\n")
	}

	var researchSection string
	if len(input.ResearchTexts) > 0 {
		researchSection = "【企業研究資料（セット2）】\n" + strings.Join(input.ResearchTexts, "\n---\n")
	}

	userPrompt := fmt.Sprintf(`ESセクション「%s」のドラフトを生成してください。

【企業情報】
企業名: %s
職種: %s
ESプロジェクトID: %s

%s

%s

【制約】
- 文字数制限: %d文字（±10%%以内に収めること）
- 結論ファースト（PREP法）を意識すること
- 企業の特徴（セット2）と自分の強み（セット1）の接点を明確にすること
- 抽象論ではなく具体的なエピソードに根ざした文章にすること

【出力要件】
以下のJSON形式のみで出力してください:
{
  "draftText": "ES本文ドラフト",
  "rationale": "この構成を選んだ理由・どのエピソードを使ったかの解説"
}`, input.SectionTitle, input.CompanyName, input.JobTitle, input.ESProjectID,
		assetsSection, researchSection, input.CharacterLimit)

	snapshot := map[string]any{
		"es_project_id":   input.ESProjectID,
		"section_title":   input.SectionTitle,
		"asset_ids":       input.AssetIDs,
		"character_limit": input.CharacterLimit,
	}

	var parsed esGenerateJSON
	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeESGenerate.String(),
		vo.JobTypeESGenerate, snapshot,
		buildMessages(userPrompt), defaultModel, 4096,
		jsonValidator(&parsed),
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(result.Response.Content), &parsed)

	return &ESGenerateOutput{
		GenerationID: result.GenerationID,
		DraftText:    parsed.DraftText,
		Rationale:    parsed.Rationale,
	}, nil
}
