// DesignDocument/7_BackendDesign.md 7.6 / DesignDocument/8_DBDesign.dbml
package supabase

import (
	"context"
	"fmt"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type creditRepositoryImpl struct {
	client *Client
}

func NewCreditRepository(client *Client) repository.CreditRepository {
	return &creditRepositoryImpl{client: client}
}

type txKey struct{}

func (r *creditRepositoryImpl) conn(ctx context.Context) interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
} {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return &txWrapper{tx: tx}
	}
	return &poolWrapper{pool: r.client.Pool}
}

func (r *creditRepositoryImpl) GetBalanceForUpdate(ctx context.Context, userID string) (*entity.CreditBalance, error) {
	c := r.conn(ctx)
	row := c.QueryRow(ctx,
		"SELECT user_id, balance, updated_at FROM credit_balances WHERE user_id = $1 FOR UPDATE",
		userID,
	)

	var b entity.CreditBalance
	err := row.Scan(&b.UserID, &b.Balance, &b.UpdatedAt)
	if err == pgx.ErrNoRows {
		return &entity.CreditBalance{UserID: userID, Balance: 0, UpdatedAt: time.Now()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan balance: %w", err)
	}
	return &b, nil
}

func (r *creditRepositoryImpl) GetBalance(ctx context.Context, userID string) (*entity.CreditBalance, error) {
	row := r.client.Pool.QueryRow(ctx,
		"SELECT user_id, balance, updated_at FROM credit_balances WHERE user_id = $1",
		userID,
	)

	var b entity.CreditBalance
	err := row.Scan(&b.UserID, &b.Balance, &b.UpdatedAt)
	if err == pgx.ErrNoRows {
		return &entity.CreditBalance{UserID: userID, Balance: 0, UpdatedAt: time.Now()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan balance: %w", err)
	}
	return &b, nil
}

func (r *creditRepositoryImpl) UpsertBalance(ctx context.Context, balance *entity.CreditBalance) error {
	c := r.conn(ctx)
	_, err := c.Exec(ctx,
		`INSERT INTO credit_balances (user_id, balance, updated_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id)
		 DO UPDATE SET balance = $2, updated_at = $3`,
		balance.UserID, balance.Balance, balance.UpdatedAt,
	)
	return err
}

func (r *creditRepositoryImpl) CreateTransaction(ctx context.Context, tx *entity.CreditTransaction) error {
	c := r.conn(ctx)
	_, err := c.Exec(ctx,
		`INSERT INTO credit_transactions (id, user_id, type, amount, status, reason, external_ref, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tx.ID, tx.UserID, tx.Type.String(), tx.Amount, tx.Status.String(),
		tx.Reason, tx.ExternalRef, tx.CreatedAt,
	)
	return err
}

func (r *creditRepositoryImpl) UpdateTransactionStatus(ctx context.Context, txID string, status vo.CreditStatus) error {
	c := r.conn(ctx)
	_, err := c.Exec(ctx,
		"UPDATE credit_transactions SET status = $1 WHERE id = $2",
		status.String(), txID,
	)
	return err
}

func (r *creditRepositoryImpl) ListTransactions(ctx context.Context, userID string, limit int, cursor *string) ([]*entity.CreditTransaction, *string, error) {
	query := `SELECT id, user_id, type, amount, status, reason, external_ref, created_at
	          FROM credit_transactions
	          WHERE user_id = $1`
	args := []any{userID}

	if cursor != nil {
		query += " AND created_at < $2"
		args = append(args, *cursor)
	}

	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit+1)

	rows, err := r.client.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var txs []*entity.CreditTransaction
	for rows.Next() {
		var tx entity.CreditTransaction
		var txType, status string
		if err := rows.Scan(&tx.ID, &tx.UserID, &txType, &tx.Amount, &status,
			&tx.Reason, &tx.ExternalRef, &tx.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("scan transaction: %w", err)
		}
		tx.Type = vo.TransactionType(txType)
		tx.Status = vo.CreditStatus(status)
		txs = append(txs, &tx)
	}

	var nextCursor *string
	if len(txs) > limit {
		c := txs[limit].CreatedAt.Format(time.RFC3339Nano)
		nextCursor = &c
		txs = txs[:limit]
	}

	return txs, nextCursor, nil
}

func (r *creditRepositoryImpl) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.client.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// Adapter wrappers for pgx.Tx and pgxpool.Pool
type txWrapper struct {
	tx pgx.Tx
}

type execResult struct {
	ct int64
}

func (e *execResult) RowsAffected() int64 { return e.ct }

func (w *txWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return w.tx.QueryRow(ctx, sql, args...)
}

func (w *txWrapper) Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error) {
	ct, err := w.tx.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &execResult{ct: ct.RowsAffected()}, nil
}

func (w *txWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.tx.Query(ctx, sql, args...)
}

type poolWrapper struct {
	pool *pgxpool.Pool
}

func (w *poolWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return w.pool.QueryRow(ctx, sql, args...)
}

func (w *poolWrapper) Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error) {
	ct, err := w.pool.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &execResult{ct: ct.RowsAffected()}, nil
}

func (w *poolWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.pool.Query(ctx, sql, args...)
}
