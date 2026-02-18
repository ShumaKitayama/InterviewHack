// DesignDocument/11_TestDesign.md 11.2.2 クレジット状態遷移
package vo

import "testing"

func TestCreditStatusTransition(t *testing.T) {
	tests := []struct {
		name  string
		from  CreditStatus
		to    CreditStatus
		canDo bool
	}{
		// UT-TX-001/002: pending -> committed
		{name: "pending to committed", from: CreditStatusPending, to: CreditStatusCommitted, canDo: true},
		// UT-TX-002: pending -> reversed
		{name: "pending to reversed", from: CreditStatusPending, to: CreditStatusReversed, canDo: true},
		// Invalid transitions
		{name: "committed to reversed", from: CreditStatusCommitted, to: CreditStatusReversed, canDo: false},
		{name: "reversed to committed", from: CreditStatusReversed, to: CreditStatusCommitted, canDo: false},
		{name: "committed to pending", from: CreditStatusCommitted, to: CreditStatusPending, canDo: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.canDo {
				t.Fatalf("expected %v for %s -> %s, got %v", tt.canDo, tt.from, tt.to, result)
			}
		})
	}
}
