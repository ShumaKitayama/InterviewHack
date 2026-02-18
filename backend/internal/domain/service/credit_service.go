// DesignDocument/7_BackendDesign.md 7.6 クレジット消費のトランザクション設計
package service

import (
	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
)

type CreditService struct{}

func NewCreditService() *CreditService {
	return &CreditService{}
}

// ValidateBalance checks if the user has enough credits and returns an error if not.
// DesignDocument/7_BackendDesign.md 7.6.2: 残高チェック (balance >= cost)
func (s *CreditService) ValidateBalance(balance *entity.CreditBalance, cost int) error {
	if cost <= 0 {
		return domainerror.ErrValidation("消費クレジット数は1以上でなければなりません")
	}
	if !balance.HasEnough(cost) {
		return domainerror.ErrInsufficientCredits(balance.Balance, cost)
	}
	return nil
}

// ValidateCostConfig checks that the requested feature is active.
func (s *CreditService) ValidateCostConfig(costConfig *entity.CreditCost) error {
	if !costConfig.IsActive {
		return domainerror.ErrFeatureDisabled(costConfig.Feature)
	}
	if costConfig.Cost <= 0 {
		return domainerror.ErrValidation("消費クレジット設定が不正です")
	}
	return nil
}

// ValidatePackage checks that the package is valid and active for purchase.
func (s *CreditService) ValidatePackage(pkg *entity.CreditPackage) error {
	if pkg == nil {
		return domainerror.ErrInvalidPackage("")
	}
	if !pkg.IsActive {
		return domainerror.ErrInvalidPackage(pkg.ID)
	}
	return nil
}
