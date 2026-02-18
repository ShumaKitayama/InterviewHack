// DesignDocument/7_BackendDesign.md 7.5.2 / DesignDocument/6_AuthAccountCahrgingDesign.md 6.7.2
// Webhook受信後のクレジット付与
package credit

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
)

type PurchaseInput struct {
	UserID      string
	Credits     int
	ExternalRef string
}

type PurchaseUsecase struct {
	creditRepo  repository.CreditRepository
	idGenerator func() string
}

func NewPurchaseUsecase(
	creditRepo repository.CreditRepository,
	idGenerator func() string,
) *PurchaseUsecase {
	return &PurchaseUsecase{
		creditRepo:  creditRepo,
		idGenerator: idGenerator,
	}
}

// Execute grants purchased credits. Called after Stripe webhook validation.
// DesignDocument/7_BackendDesign.md 7.5.2: purchase は直接 committed で確定
func (u *PurchaseUsecase) Execute(ctx context.Context, input *PurchaseInput) error {
	return u.creditRepo.WithTx(ctx, func(txCtx context.Context) error {
		txID := u.idGenerator()
		tx := entity.NewPurchaseTransaction(txID, input.UserID, input.Credits, input.ExternalRef)
		if err := u.creditRepo.CreateTransaction(txCtx, tx); err != nil {
			return fmt.Errorf("create purchase transaction: %w", err)
		}

		balance, err := u.creditRepo.GetBalanceForUpdate(txCtx, input.UserID)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}
		balance.Add(input.Credits)
		if err := u.creditRepo.UpsertBalance(txCtx, balance); err != nil {
			return fmt.Errorf("update balance: %w", err)
		}

		return nil
	})
}
