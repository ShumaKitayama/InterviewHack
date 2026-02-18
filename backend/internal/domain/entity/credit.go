// DesignDocument/7_BackendDesign.md 7.6 / DesignDocument/8_DBDesign.dbml credit_balances, credit_transactions
package entity

import (
	"time"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type CreditBalance struct {
	UserID    string
	Balance   int
	UpdatedAt time.Time
}

func (b *CreditBalance) HasEnough(cost int) bool {
	return b.Balance >= cost
}

func (b *CreditBalance) Deduct(cost int) {
	b.Balance -= cost
	b.UpdatedAt = time.Now()
}

func (b *CreditBalance) Restore(cost int) {
	b.Balance += cost
	b.UpdatedAt = time.Now()
}

func (b *CreditBalance) Add(credits int) {
	b.Balance += credits
	b.UpdatedAt = time.Now()
}

type CreditTransaction struct {
	ID          string
	UserID      string
	Type        vo.TransactionType
	Amount      int
	Status      vo.CreditStatus
	Reason      string
	ExternalRef string
	CreatedAt   time.Time
}

func NewConsumeTransaction(id, userID string, cost int, externalRef string) *CreditTransaction {
	return &CreditTransaction{
		ID:          id,
		UserID:      userID,
		Type:        vo.TransactionTypeConsume,
		Amount:      -cost,
		Status:      vo.CreditStatusPending,
		Reason:      "credit_consume",
		ExternalRef: externalRef,
		CreatedAt:   time.Now(),
	}
}

func NewPurchaseTransaction(id, userID string, credits int, externalRef string) *CreditTransaction {
	return &CreditTransaction{
		ID:          id,
		UserID:      userID,
		Type:        vo.TransactionTypePurchase,
		Amount:      credits,
		Status:      vo.CreditStatusCommitted,
		Reason:      "stripe_purchase",
		ExternalRef: externalRef,
		CreatedAt:   time.Now(),
	}
}

func NewBonusTransaction(id, userID string, credits int, reason string) *CreditTransaction {
	return &CreditTransaction{
		ID:          id,
		UserID:      userID,
		Type:        vo.TransactionTypeBonus,
		Amount:      credits,
		Status:      vo.CreditStatusCommitted,
		Reason:      reason,
		ExternalRef: "onboarding_bonus",
		CreatedAt:   time.Now(),
	}
}

func (t *CreditTransaction) Commit() error {
	if !t.Status.CanTransitionTo(vo.CreditStatusCommitted) {
		return &statusTransitionError{from: t.Status, to: vo.CreditStatusCommitted}
	}
	t.Status = vo.CreditStatusCommitted
	return nil
}

func (t *CreditTransaction) Reverse() error {
	if !t.Status.CanTransitionTo(vo.CreditStatusReversed) {
		return &statusTransitionError{from: t.Status, to: vo.CreditStatusReversed}
	}
	t.Status = vo.CreditStatusReversed
	return nil
}

type statusTransitionError struct {
	from vo.CreditStatus
	to   vo.CreditStatus
}

func (e *statusTransitionError) Error() string {
	return "invalid status transition: " + e.from.String() + " -> " + e.to.String()
}
