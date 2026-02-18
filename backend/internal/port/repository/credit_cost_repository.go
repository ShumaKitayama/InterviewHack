// DesignDocument/8_DBDesign.dbml credit_costs
package repository

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
)

type CreditCostRepository interface {
	GetByFeature(ctx context.Context, feature string) (*entity.CreditCost, error)
}
