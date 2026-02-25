// DesignDocument/7_BackendDesign.md 7.5.4 POST /ai/es-refine
// DesignDocument/4_LLMDesign.md 4.2.6 ES推敲・ブラッシュアップ
package ai

import (
	"context"
	"encoding/json"
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
	RefinedText  string
	Changes      []string
}

type esRefineJSON struct {
	RefinedText string   `json:"refinedText"`
	Changes     []string `json:"changes"`
}

type ESRefineUsecase struct {
	base *BaseAIUsecase
}

func NewESRefineUsecase(base *BaseAIUsecase) *ESRefineUsecase {
	return &ESRefineUsecase{base: base}
}

func (u *ESRefineUsecase) Execute(ctx context.Context, input *ESRefineInput) (*ESRefineOutput, error) {
	userPrompt := fmt.Sprintf(`以下のES本文を修正指示に従ってブラッシュアップしてください。

【現在のES本文】
%s

【修正指示】
%s

【出力要件】
- 指示を的確に反映しつつ、文章の自然さを保つこと
- 元の文章の良い部分は残すこと
- 変更点は具体的にリスト化すること
- 文字数は元の文章と大きく変えないこと

以下のJSON形式のみで出力してください:
{
  "refinedText": "修正後のES本文",
  "changes": ["変更点1: ○○を△△に修正", "変更点2: ..."]
}`, input.CurrentText, input.Instructions)

	snapshot := map[string]any{
		"current_text": "(omitted for log safety)",
		"instructions": input.Instructions,
	}

	var parsed esRefineJSON
	result, err := u.base.ExecuteWithCredit(
		ctx, input.UserID, vo.JobTypeESRefine.String(),
		vo.JobTypeESRefine, snapshot,
		buildMessages(userPrompt), defaultModel, 4096,
		jsonValidator(&parsed),
	)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(result.Response.Content), &parsed)

	return &ESRefineOutput{
		GenerationID: result.GenerationID,
		RefinedText:  parsed.RefinedText,
		Changes:      parsed.Changes,
	}, nil
}
