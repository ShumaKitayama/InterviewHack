// DesignDocument/7_BackendDesign.md 7.6 / DesignDocument/6_AuthAccountCahrgingDesign.md 6.7
// クレジットトランザクションの状態遷移: pending -> committed | reversed
package vo

type CreditStatus string

const (
	CreditStatusPending   CreditStatus = "pending"
	CreditStatusCommitted CreditStatus = "committed"
	CreditStatusReversed  CreditStatus = "reversed"
)

func (s CreditStatus) String() string {
	return string(s)
}

func (s CreditStatus) IsValid() bool {
	switch s {
	case CreditStatusPending, CreditStatusCommitted, CreditStatusReversed:
		return true
	}
	return false
}

func (s CreditStatus) CanTransitionTo(next CreditStatus) bool {
	switch s {
	case CreditStatusPending:
		return next == CreditStatusCommitted || next == CreditStatusReversed
	default:
		return false
	}
}
