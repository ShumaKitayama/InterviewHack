// DesignDocument/7_BackendDesign.md 7.5.4 / 7.6
// DesignDocument/4_LLMDesign.md 4.3 プロンプト設計方針
// AI共通: クレジット予約 -> AI実行(リトライ付き) -> 確定/取消
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/credit"
)

const (
	maxRetries   = 2
	defaultModel = "gpt-4o-mini"

	// DesignDocument/4_LLMDesign.md 4.3.1 System ロール
	systemPrompt = `あなたは凄腕の面接官・キャリアコーチです。就活生・転職者の面接準備と書類作成を支援します。

制約:
- 回答には必ず具体例・数値・学びを含めること
- 一般論テンプレートにならず、ユーザーの自己資産データに基づくこと
- ユーザーの経歴に存在しない事実を捏造しないこと
- STAR法（状況・課題・行動・結果）を意識した構成にすること
- 出力はすべてJSON形式で返すこと。JSON以外のテキストを含めないこと
- 日本語で回答すること`
)

type BaseAIUsecase struct {
	consumeUC *credit.ConsumeUsecase
	llmClient external.LLMClient
	jobRepo   repository.GenerationJobRepository
	idGen     func() string
}

func NewBaseAIUsecase(
	consumeUC *credit.ConsumeUsecase,
	llmClient external.LLMClient,
	jobRepo repository.GenerationJobRepository,
	idGen func() string,
) *BaseAIUsecase {
	return &BaseAIUsecase{
		consumeUC: consumeUC,
		llmClient: llmClient,
		jobRepo:   jobRepo,
		idGen:     idGen,
	}
}

type AIExecutionResult struct {
	GenerationID string
	Response     *external.LLMResponse
}

// ExecuteWithCredit handles the Reserve -> LLM call (with retry) -> Commit/Reverse flow.
// DesignDocument/7_BackendDesign.md 7.5.4 共通仕様: 出力はJSONスキーマで検証し、壊れていたらリトライ
func (b *BaseAIUsecase) ExecuteWithCredit(
	ctx context.Context,
	userID string,
	feature string,
	jobType vo.JobType,
	inputSnapshot map[string]any,
	messages []external.LLMMessage,
	model string,
	maxTokens int,
	validator func(content string) error,
) (*AIExecutionResult, error) {
	generationID := b.idGen()

	reserved, err := b.consumeUC.Reserve(ctx, &credit.ConsumeInput{
		UserID:      userID,
		Feature:     feature,
		ExternalRef: generationID,
	})
	if err != nil {
		return nil, err
	}

	if model == "" {
		model = defaultModel
	}

	var resp *external.LLMResponse
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, lastErr = b.llmClient.Generate(ctx, &external.LLMRequest{
			Messages:  messages,
			Model:     model,
			MaxTokens: maxTokens,
			JSONMode:  true,
		})
		if lastErr != nil {
			slog.Warn("LLM call failed", "attempt", attempt, "error", lastErr, "job_id", generationID)
			continue
		}

		if validator != nil {
			if validErr := validator(resp.Content); validErr != nil {
				slog.Warn("LLM output schema validation failed, retrying",
					"attempt", attempt, "error", validErr, "job_id", generationID)
				lastErr = validErr
				continue
			}
		}

		lastErr = nil
		break
	}

	job := entity.NewGenerationJob(generationID, generationID, userID, jobType, inputSnapshot, model)

	if lastErr != nil {
		slog.Error("LLM execution exhausted retries, reversing credit",
			"error", lastErr, "job_id", generationID, "retry_count", maxRetries)
		job.MarkFailed(vo.ErrorCodeOpenAIError.String(), lastErr.Error())
		_ = b.jobRepo.Create(ctx, job)

		if reverseErr := b.consumeUC.Reverse(ctx, userID, reserved.TransactionID, reserved.Cost); reverseErr != nil {
			slog.Error("failed to reverse credit", "error", reverseErr, "tx_id", reserved.TransactionID)
		}
		return nil, fmt.Errorf("LLM execution: %w", lastErr)
	}

	job.SetTokenUsage(resp.TokensUsed, resp.CostEstimate)
	if err := b.jobRepo.Create(ctx, job); err != nil {
		slog.Error("failed to save generation job", "error", err)
	}

	if err := b.consumeUC.Commit(ctx, reserved.TransactionID); err != nil {
		slog.Error("failed to commit credit, attempting reverse",
			"error", err, "tx_id", reserved.TransactionID)
		if reverseErr := b.consumeUC.Reverse(ctx, userID, reserved.TransactionID, reserved.Cost); reverseErr != nil {
			slog.Error("failed to reverse credit after commit failure",
				"error", reverseErr, "tx_id", reserved.TransactionID)
		}
		return nil, fmt.Errorf("commit credit transaction: %w", err)
	}

	return &AIExecutionResult{GenerationID: generationID, Response: resp}, nil
}

// buildMessages creates the standard system + user message pair.
func buildMessages(userPrompt string) []external.LLMMessage {
	return []external.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

// jsonValidator returns a validator that tries to unmarshal JSON into the given target.
func jsonValidator(target any) func(string) error {
	return func(content string) error {
		if err := json.Unmarshal([]byte(content), target); err != nil {
			return fmt.Errorf("invalid JSON output: %w", err)
		}
		return nil
	}
}
