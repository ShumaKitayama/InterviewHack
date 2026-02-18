// DesignDocument/8_DBDesign.dbml credit_costs
package supabase

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/jackc/pgx/v5"
)

type creditCostRepositoryImpl struct {
	client *Client
}

func NewCreditCostRepository(client *Client) repository.CreditCostRepository {
	return &creditCostRepositoryImpl{client: client}
}

func (r *creditCostRepositoryImpl) GetByFeature(ctx context.Context, feature string) (*entity.CreditCost, error) {
	row := r.client.Pool.QueryRow(ctx,
		"SELECT feature, cost, is_active FROM credit_costs WHERE feature = $1", feature,
	)

	var cost entity.CreditCost
	err := row.Scan(&cost.Feature, &cost.Cost, &cost.IsActive)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan credit cost: %w", err)
	}
	return &cost, nil
}
