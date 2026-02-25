// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/qa-generate
// DesignDocument/4_LLMDesign.md 4.2.1 想定Q&A生成（事前対策）
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type QAGenerateInput struct {
	UserID        string
	CompanyID     string
	CompanyName   string
	JobTitle      string
	AssetIDs      []string
	AssetTexts    []string
	QuestionCount int
}

type QAItem struct {
	QuestionText       string   `json:"questionText"`
	AnswerText         string   `json:"answerText"`
	ReferencedAssetIDs []string `json:"referencedAssetIds"`
}

type QAGenerateOutput struct {
	GenerationID string
	Items        []QAItem
}

type qaGenerateJSON struct {
	Items []QAItem `json:"items"`
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

	var assetsSection string
	if len(input.AssetTexts) > 0 {
		assetsSection = "【自己資産データ】\n" + strings.Join(input.AssetTexts, "\n---\n")
	} else if len(input.AssetIDs) > 0 {
		assetsSection = fmt.Sprintf("【参照自己資産ID】\n%s", strings.Join(input.AssetIDs, ", "))
	}

	userPrompt := fmt.Sprintf(`以下の情報に基づき、想定面接Q&Aを%d件生成してください。

【企業情報】
企業名: %s
企業ID: %s
職種: %s

%s

【出力要件】
- 質問は「深掘りされやすい」具体的な観点であること
- 回答は自己資産データに基づき、具体例・数値・学びを含むこと
- 回答はSTAR法（状況・課題・行動・結果）を意識した構成にすること
- 一般論ではなく、応募者固有のエピソードに根ざした回答にすること

以下のJSON形式のみで出力してください:
{
  "items": [
    {
      "questionText": "質問テキスト",
      "answerText": "模範回答テキスト",
      "referencedAssetIds": ["参照した自己資産ID"]
    }
  ]
}`, input.QuestionCount, input.CompanyName, input.CompanyID, input.JobTitle, assetsSection)

	snapshot := map[string]any{
		"company_id":     input.CompanyID,
		"job_title":      input.JobTitle,
		"asset_ids":      input.AssetIDs,
		"question_count": input.QuestionCount,
	}

	var parsed qaGenerateJSON
	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeQAGenerate.String(),
		vo.JobTypeQAGenerate, snapshot,
		buildMessages(userPrompt), defaultModel, 4096,
		jsonValidator(&parsed),
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(result.Response.Content), &parsed)

	return &QAGenerateOutput{
		GenerationID: result.GenerationID,
		Items:        parsed.Items,
	}, nil
}
