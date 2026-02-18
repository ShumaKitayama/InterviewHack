// DesignDocument/8_DBDesign.dbml credit_packages
package supabase

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/jackc/pgx/v5"
)

type creditPackageRepositoryImpl struct {
	client *Client
}

func NewCreditPackageRepository(client *Client) repository.CreditPackageRepository {
	return &creditPackageRepositoryImpl{client: client}
}

func (r *creditPackageRepositoryImpl) GetByID(ctx context.Context, id string) (*entity.CreditPackage, error) {
	row := r.client.Pool.QueryRow(ctx,
		"SELECT id, name, credits, stripe_price_id, is_active FROM credit_packages WHERE id = $1", id,
	)

	var pkg entity.CreditPackage
	err := row.Scan(&pkg.ID, &pkg.Name, &pkg.Credits, &pkg.StripePriceID, &pkg.IsActive)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan credit package: %w", err)
	}
	return &pkg, nil
}

func (r *creditPackageRepositoryImpl) ListActive(ctx context.Context) ([]*entity.CreditPackage, error) {
	rows, err := r.client.Pool.Query(ctx,
		"SELECT id, name, credits, stripe_price_id, is_active FROM credit_packages WHERE is_active = true ORDER BY credits ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("query credit packages: %w", err)
	}
	defer rows.Close()

	var packages []*entity.CreditPackage
	for rows.Next() {
		var pkg entity.CreditPackage
		if err := rows.Scan(&pkg.ID, &pkg.Name, &pkg.Credits, &pkg.StripePriceID, &pkg.IsActive); err != nil {
			return nil, fmt.Errorf("scan credit package: %w", err)
		}
		packages = append(packages, &pkg)
	}
	return packages, nil
}
