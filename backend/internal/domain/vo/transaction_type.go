// DesignDocument/8_DBDesign.dbml credit_transactions.type
package vo

type TransactionType string

const (
	TransactionTypePurchase   TransactionType = "purchase"
	TransactionTypeConsume    TransactionType = "consume"
	TransactionTypeBonus      TransactionType = "bonus"
	TransactionTypeAdjustment TransactionType = "adjustment"
)

func (t TransactionType) String() string {
	return string(t)
}

func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypePurchase, TransactionTypeConsume,
		TransactionTypeBonus, TransactionTypeAdjustment:
		return true
	}
	return false
}
