// DesignDocument/8_DBDesign.dbml credit_packages
package repository

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
)

type CreditPackageRepository interface {
	GetByID(ctx context.Context, id string) (*entity.CreditPackage, error)
	ListActive(ctx context.Context) ([]*entity.CreditPackage, error)
}
