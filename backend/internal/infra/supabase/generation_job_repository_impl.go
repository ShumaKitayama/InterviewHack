// DesignDocument/7_BackendDesign.md 7.9 / DesignDocument/8_DBDesign.dbml generation_jobs
package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/jackc/pgx/v5"
)

type generationJobRepositoryImpl struct {
	client *Client
}

func NewGenerationJobRepository(client *Client) repository.GenerationJobRepository {
	return &generationJobRepositoryImpl{client: client}
}

func (r *generationJobRepositoryImpl) Create(ctx context.Context, job *entity.GenerationJob) error {
	snapshot, err := json.Marshal(job.InputSnapshot)
	if err != nil {
		return fmt.Errorf("marshal input snapshot: %w", err)
	}

	_, err = r.client.Pool.Exec(ctx,
		`INSERT INTO generation_jobs
		 (id, external_ref, user_id, job_type, input_snapshot, model_name,
		  token_used, cost_estimate, status, error_code, error_message, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		job.ID, job.ExternalRef, job.UserID, job.JobType.String(),
		snapshot, job.ModelName, job.TokenUsed, job.CostEstimate,
		string(job.Status), job.ErrorCode, job.ErrorMessage, job.CreatedAt,
	)
	return err
}

func (r *generationJobRepositoryImpl) Update(ctx context.Context, job *entity.GenerationJob) error {
	_, err := r.client.Pool.Exec(ctx,
		`UPDATE generation_jobs SET status = $1, error_code = $2, error_message = $3,
		 token_used = $4, cost_estimate = $5 WHERE id = $6`,
		string(job.Status), job.ErrorCode, job.ErrorMessage,
		job.TokenUsed, job.CostEstimate, job.ID,
	)
	return err
}

func (r *generationJobRepositoryImpl) GetByID(ctx context.Context, id string) (*entity.GenerationJob, error) {
	row := r.client.Pool.QueryRow(ctx,
		`SELECT id, external_ref, user_id, job_type, input_snapshot, model_name,
		 token_used, cost_estimate, status, error_code, error_message, created_at
		 FROM generation_jobs WHERE id = $1`, id,
	)

	var job entity.GenerationJob
	var jobType, status string
	var snapshot []byte
	err := row.Scan(&job.ID, &job.ExternalRef, &job.UserID, &jobType,
		&snapshot, &job.ModelName, &job.TokenUsed, &job.CostEstimate,
		&status, &job.ErrorCode, &job.ErrorMessage, &job.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan generation job: %w", err)
	}

	job.JobType = vo.JobType(jobType)
	job.Status = entity.GenerationJobStatus(status)
	_ = json.Unmarshal(snapshot, &job.InputSnapshot)

	return &job, nil
}
