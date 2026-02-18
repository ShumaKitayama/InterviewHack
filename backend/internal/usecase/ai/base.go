// DesignDocument/7_BackendDesign.md 7.5.4 / 7.6
// AI共通: クレジット予約 -> AI実行 -> 確定/取消
package ai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/credit"
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

// ExecuteWithCredit handles the Reserve -> LLM call -> Commit/Reverse flow.
func (b *BaseAIUsecase) ExecuteWithCredit(
	ctx context.Context,
	userID string,
	feature string,
	jobType vo.JobType,
	inputSnapshot map[string]any,
	prompt string,
	model string,
	maxTokens int,
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

	resp, err := b.llmClient.Generate(ctx, &external.LLMRequest{
		Prompt:    prompt,
		Model:     model,
		MaxTokens: maxTokens,
	})

	job := entity.NewGenerationJob(generationID, generationID, userID, jobType, inputSnapshot, model)

	if err != nil {
		slog.Error("LLM execution failed, reversing credit", "error", err, "job_id", generationID)
		job.MarkFailed(vo.ErrorCodeOpenAIError.String(), err.Error())
		_ = b.jobRepo.Create(ctx, job)

		if reverseErr := b.consumeUC.Reverse(ctx, userID, reserved.TransactionID, reserved.Cost); reverseErr != nil {
			slog.Error("failed to reverse credit", "error", reverseErr, "tx_id", reserved.TransactionID)
		}
		return nil, fmt.Errorf("LLM execution: %w", err)
	}

	job.SetTokenUsage(resp.TokensUsed, resp.CostEstimate)
	if err := b.jobRepo.Create(ctx, job); err != nil {
		slog.Error("failed to save generation job", "error", err)
	}

	if err := b.consumeUC.Commit(ctx, reserved.TransactionID); err != nil {
		slog.Error("failed to commit credit transaction", "error", err, "tx_id", reserved.TransactionID)
	}

	return &AIExecutionResult{GenerationID: generationID, Response: resp}, nil
}
