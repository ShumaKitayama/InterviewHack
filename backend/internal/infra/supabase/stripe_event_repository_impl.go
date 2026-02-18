// DesignDocument/9_SecurityDesign.md 9.5.3 / DesignDocument/8_DBDesign.dbml stripe_events
package supabase

import (
	"context"

	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/jackc/pgx/v5"
)

type stripeEventRepositoryImpl struct {
	client *Client
}

func NewStripeEventRepository(client *Client) repository.StripeEventRepository {
	return &stripeEventRepositoryImpl{client: client}
}

func (r *stripeEventRepositoryImpl) Exists(ctx context.Context, eventID string) (bool, error) {
	row := r.client.Pool.QueryRow(ctx,
		"SELECT 1 FROM stripe_events WHERE id = $1", eventID,
	)
	var dummy int
	err := row.Scan(&dummy)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *stripeEventRepositoryImpl) Save(ctx context.Context, eventID string) error {
	_, err := r.client.Pool.Exec(ctx,
		"INSERT INTO stripe_events (id) VALUES ($1) ON CONFLICT (id) DO NOTHING",
		eventID,
	)
	return err
}
