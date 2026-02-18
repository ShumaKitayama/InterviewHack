// DesignDocument/11_TestDesign.md 11.2.1 クレジット残高チェック
package service

import (
	"testing"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

func TestValidateBalance(t *testing.T) {
	svc := NewCreditService()

	tests := []struct {
		name    string
		balance int
		cost    int
		wantErr bool
		errCode vo.ErrorCode
	}{
		// UT-CR-001: balance > cost
		{name: "sufficient balance", balance: 100, cost: 10, wantErr: false},
		// UT-CR-002: balance = cost
		{name: "exact balance", balance: 10, cost: 10, wantErr: false},
		// UT-CR-003: balance < cost
		{name: "insufficient balance", balance: 5, cost: 10, wantErr: true, errCode: vo.ErrorCodeInsufficientCredit},
		// UT-CR-004: cost <= 0
		{name: "zero cost", balance: 100, cost: 0, wantErr: true, errCode: vo.ErrorCodeValidationError},
		{name: "negative cost", balance: 100, cost: -5, wantErr: true, errCode: vo.ErrorCodeValidationError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bal := &entity.CreditBalance{UserID: "u1", Balance: tt.balance}
			err := svc.ValidateBalance(bal, tt.cost)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domErr, ok := err.(*domainerror.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domErr.Code != tt.errCode {
					t.Fatalf("expected code %s, got %s", tt.errCode, domErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateCostConfig(t *testing.T) {
	svc := NewCreditService()

	// UT-CR-005: feature disabled
	t.Run("disabled feature", func(t *testing.T) {
		cost := &entity.CreditCost{Feature: "qa_generate", Cost: 10, IsActive: false}
		err := svc.ValidateCostConfig(cost)
		if err == nil {
			t.Fatal("expected error for disabled feature")
		}
		domErr, ok := err.(*domainerror.DomainError)
		if !ok || domErr.Code != vo.ErrorCodeFeatureDisabled {
			t.Fatalf("expected FEATURE_DISABLED, got %v", err)
		}
	})

	t.Run("active feature", func(t *testing.T) {
		cost := &entity.CreditCost{Feature: "qa_generate", Cost: 10, IsActive: true}
		if err := svc.ValidateCostConfig(cost); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
