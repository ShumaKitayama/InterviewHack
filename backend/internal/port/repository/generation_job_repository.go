// DesignDocument/7_BackendDesign.md 7.9 / DesignDocument/8_DBDesign.dbml generation_jobs
package repository

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
)

type GenerationJobRepository interface {
	Create(ctx context.Context, job *entity.GenerationJob) error
	Update(ctx context.Context, job *entity.GenerationJob) error
	GetByID(ctx context.Context, id string) (*entity.GenerationJob, error)
}
