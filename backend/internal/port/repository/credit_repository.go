// DesignDocument/7_BackendDesign.md 7.2.2 / 7.6
// 依存性逆転: Usecase/Domain からのみ参照し、実装は Infra 層に置く
package repository

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
)

type CreditRepository interface {
	// GetBalanceForUpdate acquires a row lock (SELECT ... FOR UPDATE) and returns the balance.
	// DesignDocument/7_BackendDesign.md 7.6.2: 残高行ロック
	GetBalanceForUpdate(ctx context.Context, userID string) (*entity.CreditBalance, error)

	// GetBalance returns the balance without locking (read-only).
	GetBalance(ctx context.Context, userID string) (*entity.CreditBalance, error)

	// BalanceExists checks if a credit_balances row exists for the given user.
	BalanceExists(ctx context.Context, userID string) (bool, error)

	// UpsertBalance creates or updates the balance row.
	UpsertBalance(ctx context.Context, balance *entity.CreditBalance) error

	// CreateTransaction inserts a new credit transaction.
	CreateTransaction(ctx context.Context, tx *entity.CreditTransaction) error

	// UpdateTransactionStatus updates the status of an existing transaction.
	UpdateTransactionStatus(ctx context.Context, txID string, status vo.CreditStatus) error

	// ListTransactions returns paginated transactions for a user.
	ListTransactions(ctx context.Context, userID string, limit int, cursor *string) ([]*entity.CreditTransaction, *string, error)

	// WithTx executes fn within a database transaction.
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
