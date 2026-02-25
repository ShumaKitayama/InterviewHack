// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/answer-review
// DesignDocument/4_LLMDesign.md 4.2.2 面接ログ添削（事後対策）
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type AnswerReviewInput struct {
	UserID       string
	QuestionText string
	AnswerText   string
	AssetIDs     []string
	AssetTexts   []string
	Strictness   string
}

type AnswerReviewOutput struct {
	GenerationID       string
	Issues             []string
	Improvements       []string
	ImprovedAnswerText string
	ReferencedAssetIDs []string
}

type answerReviewJSON struct {
	Issues             []string `json:"issues"`
	Improvements       []string `json:"improvements"`
	ImprovedAnswerText string   `json:"improvedAnswerText"`
	ReferencedAssetIDs []string `json:"referencedAssetIds"`
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

	var assetsSection string
	if len(input.AssetTexts) > 0 {
		assetsSection = "【自己資産データ】\n" + strings.Join(input.AssetTexts, "\n---\n")
	}

	strictnessLabel := "標準"
	if input.Strictness == "hard" {
		strictnessLabel = "厳しめ（深掘り視点で厳格に評価）"
	}

	userPrompt := fmt.Sprintf(`以下の面接回答を添削してください。

【質問】
%s

【ユーザーの回答】
%s

%s

【添削の厳しさ】
%s

【評価観点】
- 論理性: 主張→根拠→具体例の流れが明確か
- 具体性: 抽象論ではなく数値・固有名詞・成果が入っているか
- 説得力: 自分ならではのエピソードか。一般論になっていないか
- 構成: STAR法（状況・課題・行動・結果）を意識した構成になっているか

【出力要件】
- 否定だけでなく「どう直せば良いか」を具体的に示すこと
- 自己資産のどのエピソードを使ったか明記すること
- 改善版回答は元の回答の良い部分を残しつつ改善すること

以下のJSON形式のみで出力してください:
{
  "issues": ["問題点1", "問題点2"],
  "improvements": ["改善提案1", "改善提案2"],
  "improvedAnswerText": "改善版の回答テキスト",
  "referencedAssetIds": ["参照した自己資産ID"]
}`, input.QuestionText, input.AnswerText, assetsSection, strictnessLabel)

	snapshot := map[string]any{
		"question_text": input.QuestionText,
		"asset_ids":     input.AssetIDs,
		"strictness":    input.Strictness,
	}

	var parsed answerReviewJSON
	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeAnswerReview.String(),
		vo.JobTypeAnswerReview, snapshot,
		buildMessages(userPrompt), defaultModel, 4096,
		jsonValidator(&parsed),
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(result.Response.Content), &parsed)

	return &AnswerReviewOutput{
		GenerationID:       result.GenerationID,
		Issues:             parsed.Issues,
		Improvements:       parsed.Improvements,
		ImprovedAnswerText: parsed.ImprovedAnswerText,
		ReferencedAssetIDs: parsed.ReferencedAssetIDs,
	}, nil
}
