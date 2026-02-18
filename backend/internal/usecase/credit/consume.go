// DesignDocument/7_BackendDesign.md 7.6 クレジット消費のトランザクション設計
// Reserve -> AI実行 -> Commit / Reversed
package credit

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/service"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
)

type ConsumeInput struct {
	UserID      string
	Feature     string
	ExternalRef string
}

type ConsumeOutput struct {
	TransactionID string
	Cost          int
}

type ConsumeUsecase struct {
	creditRepo    repository.CreditRepository
	costRepo      repository.CreditCostRepository
	creditService *service.CreditService
	idGenerator   func() string
}

func NewConsumeUsecase(
	creditRepo repository.CreditRepository,
	costRepo repository.CreditCostRepository,
	creditService *service.CreditService,
	idGenerator func() string,
) *ConsumeUsecase {
	return &ConsumeUsecase{
		creditRepo:    creditRepo,
		costRepo:      costRepo,
		creditService: creditService,
		idGenerator:   idGenerator,
	}
}

// Reserve deducts credits in a pending state.
// DesignDocument/7_BackendDesign.md 7.6.2 steps 1-4
func (u *ConsumeUsecase) Reserve(ctx context.Context, input *ConsumeInput) (*ConsumeOutput, error) {
	costConfig, err := u.costRepo.GetByFeature(ctx, input.Feature)
	if err != nil {
		return nil, fmt.Errorf("get cost config: %w", err)
	}
	if err := u.creditService.ValidateCostConfig(costConfig); err != nil {
		return nil, err
	}

	var txID string
	err = u.creditRepo.WithTx(ctx, func(txCtx context.Context) error {
		balance, err := u.creditRepo.GetBalanceForUpdate(txCtx, input.UserID)
		if err != nil {
			return fmt.Errorf("get balance for update: %w", err)
		}

		if err := u.creditService.ValidateBalance(balance, costConfig.Cost); err != nil {
			return err
		}

		txID = u.idGenerator()
		tx := entity.NewConsumeTransaction(txID, input.UserID, costConfig.Cost, input.ExternalRef)
		if err := u.creditRepo.CreateTransaction(txCtx, tx); err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		balance.Deduct(costConfig.Cost)
		if err := u.creditRepo.UpsertBalance(txCtx, balance); err != nil {
			return fmt.Errorf("update balance: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ConsumeOutput{TransactionID: txID, Cost: costConfig.Cost}, nil
}

// Commit marks a pending transaction as committed.
// DesignDocument/7_BackendDesign.md 7.6.2 step 6
func (u *ConsumeUsecase) Commit(ctx context.Context, transactionID string) error {
	return u.creditRepo.UpdateTransactionStatus(ctx, transactionID, vo.CreditStatusCommitted)
}

// Reverse marks a pending transaction as reversed and restores the balance.
// DesignDocument/7_BackendDesign.md 7.6.2 step 7
func (u *ConsumeUsecase) Reverse(ctx context.Context, userID, transactionID string, cost int) error {
	return u.creditRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := u.creditRepo.UpdateTransactionStatus(txCtx, transactionID, vo.CreditStatusReversed); err != nil {
			return fmt.Errorf("update transaction status: %w", err)
		}

		balance, err := u.creditRepo.GetBalanceForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("get balance for update: %w", err)
		}
		balance.Restore(cost)
		if err := u.creditRepo.UpsertBalance(txCtx, balance); err != nil {
			return fmt.Errorf("restore balance: %w", err)
		}

		return nil
	})
}
