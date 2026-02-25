// DesignDocument/6_AuthAccountCahrgingDesign.md 6.8 無料クレジット（オンボーディング）
// 初回登録時にボーナスクレジット付与（20 credits）
package credit

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
)

const onboardingBonusCredits = 20

type BonusUsecase struct {
	creditRepo  repository.CreditRepository
	idGenerator func() string
}

func NewBonusUsecase(
	creditRepo repository.CreditRepository,
	idGenerator func() string,
) *BonusUsecase {
	return &BonusUsecase{
		creditRepo:  creditRepo,
		idGenerator: idGenerator,
	}
}

type BonusResult struct {
	Granted bool `json:"granted"`
	Credits int  `json:"credits"`
	Balance int  `json:"balance"`
}

// GrantOnboardingBonus checks if the user already has a credit_balance row.
// If not, creates one with the bonus credits and records the transaction.
func (u *BonusUsecase) GrantOnboardingBonus(ctx context.Context, userID string) (*BonusResult, error) {
	exists, err := u.creditRepo.BalanceExists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check balance existence: %w", err)
	}

	if exists {
		balance, err := u.creditRepo.GetBalance(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get balance: %w", err)
		}
		return &BonusResult{Granted: false, Credits: 0, Balance: balance.Balance}, nil
	}

	var finalBalance int
	err = u.creditRepo.WithTx(ctx, func(txCtx context.Context) error {
		balance, err := u.creditRepo.GetBalanceForUpdate(txCtx, userID)
		if err != nil {
			return fmt.Errorf("get balance: %w", err)
		}

		txID := u.idGenerator()
		tx := entity.NewBonusTransaction(txID, userID, onboardingBonusCredits, "onboarding_bonus")
		if err := u.creditRepo.CreateTransaction(txCtx, tx); err != nil {
			return fmt.Errorf("create bonus transaction: %w", err)
		}

		balance.Add(onboardingBonusCredits)
		if err := u.creditRepo.UpsertBalance(txCtx, balance); err != nil {
			return fmt.Errorf("update balance: %w", err)
		}

		finalBalance = balance.Balance
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &BonusResult{Granted: true, Credits: onboardingBonusCredits, Balance: finalBalance}, nil
}
