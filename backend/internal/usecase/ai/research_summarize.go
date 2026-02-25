// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/research-summarize
// DesignDocument/4_LLMDesign.md 4.2.4 企業研究資料の要約・構造化
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type ResearchSummarizeInput struct {
	UserID  string
	Type    string // "url" | "text" | "pdf"
	Content string
}

type ResearchSummarizeOutput struct {
	GenerationID    string
	Summary         string
	KeyPoints       []string
	UsageSuggestion string
}

type researchSummarizeJSON struct {
	Summary         string   `json:"summary"`
	KeyPoints       []string `json:"keyPoints"`
	UsageSuggestion string   `json:"usageSuggestion"`
}

type ResearchSummarizeUsecase struct {
	base *BaseAIUsecase
}

func NewResearchSummarizeUsecase(base *BaseAIUsecase) *ResearchSummarizeUsecase {
	return &ResearchSummarizeUsecase{base: base}
}

func (u *ResearchSummarizeUsecase) Execute(ctx context.Context, input *ResearchSummarizeInput) (*ResearchSummarizeOutput, error) {
	sourceLabel := "テキスト"
	switch input.Type {
	case "url":
		sourceLabel = "Webページ"
	case "pdf":
		sourceLabel = "PDF"
	}

	userPrompt := fmt.Sprintf(`以下の企業研究資料（%s）を分析し、構造化された要約を生成してください。

【資料内容】
%s

【出力要件】
- 広告、ヘッダー/フッターなどのノイズは除去すること
- 「なぜこの会社か」に使えそうな情報を優先的に抽出すること
- 要約は企業の魅力・特徴を中心に3〜5文でまとめること
- 重要ポイントは箇条書きで3〜5項目
- 活用提案は「志望動機に使えそう」「自己PRとの接点になりそう」等の具体的な提案

以下のJSON形式のみで出力してください:
{
  "summary": "企業の要約テキスト",
  "keyPoints": ["重要ポイント1", "重要ポイント2"],
  "usageSuggestion": "ES・面接での活用方法の提案"
}`, sourceLabel, input.Content)

	snapshot := map[string]any{
		"type":    input.Type,
		"content": "(omitted for log safety)",
	}

	var parsed researchSummarizeJSON
	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeResearchSummarize.String(),
		vo.JobTypeResearchSummarize, snapshot,
		buildMessages(userPrompt), defaultModel, 4096,
		jsonValidator(&parsed),
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(result.Response.Content), &parsed)

	return &ResearchSummarizeOutput{
		GenerationID:    result.GenerationID,
		Summary:         parsed.Summary,
		KeyPoints:       parsed.KeyPoints,
		UsageSuggestion: parsed.UsageSuggestion,
	}, nil
}
