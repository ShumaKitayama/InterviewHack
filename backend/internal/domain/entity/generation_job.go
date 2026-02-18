// DesignDocument/7_BackendDesign.md 7.9 / DesignDocument/8_DBDesign.dbml generation_jobs
package entity

import (
	"time"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type GenerationJobStatus string

const (
	GenerationJobStatusSuccess GenerationJobStatus = "success"
	GenerationJobStatusFailed  GenerationJobStatus = "failed"
)

type GenerationJob struct {
	ID            string
	ExternalRef   string
	UserID        string
	JobType       vo.JobType
	InputSnapshot map[string]any
	ModelName     string
	TokenUsed     *int
	CostEstimate  *float64
	Status        GenerationJobStatus
	ErrorCode     *string
	ErrorMessage  *string
	CreatedAt     time.Time
}

func NewGenerationJob(id, externalRef, userID string, jobType vo.JobType, input map[string]any, modelName string) *GenerationJob {
	return &GenerationJob{
		ID:            id,
		ExternalRef:   externalRef,
		UserID:        userID,
		JobType:       jobType,
		InputSnapshot: input,
		ModelName:     modelName,
		Status:        GenerationJobStatusSuccess,
		CreatedAt:     time.Now(),
	}
}

func (j *GenerationJob) MarkFailed(code, message string) {
	j.Status = GenerationJobStatusFailed
	j.ErrorCode = &code
	j.ErrorMessage = &message
}

func (j *GenerationJob) SetTokenUsage(tokens int, cost float64) {
	j.TokenUsed = &tokens
	j.CostEstimate = &cost
}
