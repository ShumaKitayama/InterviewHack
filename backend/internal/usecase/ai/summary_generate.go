// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/summary-generate
// DesignDocument/4_LLMDesign.md 4.2.3 要約生成（ダッシュボード用）
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type SummaryGenerateInput struct {
	UserID          string
	InterviewID     string
	InterviewLogIDs []string
	LogTexts        []string
}

type SummaryGenerateOutput struct {
	GenerationID   string
	SummaryBullets []string
}

type summaryGenerateJSON struct {
	SummaryBullets []string `json:"summaryBullets"`
}

type SummaryGenerateUsecase struct {
	base *BaseAIUsecase
}

func NewSummaryGenerateUsecase(base *BaseAIUsecase) *SummaryGenerateUsecase {
	return &SummaryGenerateUsecase{base: base}
}

func (u *SummaryGenerateUsecase) Execute(ctx context.Context, input *SummaryGenerateInput) (*SummaryGenerateOutput, error) {
	var logsSection string
	if len(input.LogTexts) > 0 {
		logsSection = "【面接ログデータ】\n" + strings.Join(input.LogTexts, "\n---\n")
	} else {
		logsSection = fmt.Sprintf("【面接ログID】\n%s", strings.Join(input.InterviewLogIDs, ", "))
	}

	userPrompt := fmt.Sprintf(`以下の面接ログ（添削結果含む）を分析し、次回の面接で特に意識すべきポイントを3〜5個に要約してください。

%s

【出力要件】
- 抽象論ではなく「次に○○する」のように行動レベルに落とせる表現にすること
- 複数の面接ログに共通する傾向を優先的にピックアップすること
- 各項目は1〜2文で簡潔にまとめること

以下のJSON形式のみで出力してください:
{
  "summaryBullets": ["要約ポイント1", "要約ポイント2", "要約ポイント3"]
}`, logsSection)

	snapshot := map[string]any{
		"interview_id":      input.InterviewID,
		"interview_log_ids": input.InterviewLogIDs,
	}

	var parsed summaryGenerateJSON
	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeSummaryGenerate.String(),
		vo.JobTypeSummaryGenerate, snapshot,
		buildMessages(userPrompt), defaultModel, 2048,
		jsonValidator(&parsed),
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(result.Response.Content), &parsed)

	return &SummaryGenerateOutput{
		GenerationID:   result.GenerationID,
		SummaryBullets: parsed.SummaryBullets,
	}, nil
}
