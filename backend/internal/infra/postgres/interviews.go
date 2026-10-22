package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
)

// InterviewRepo implements repository.InterviewRepository.
type InterviewRepo struct{ DB *DB }

const interviewCols = `id, user_id, company_id, interview_date, phase, result, memo, created_at, updated_at`

func scanInterview(row interface{ Scan(...any) error }) (entity.Interview, error) {
	var i entity.Interview
	err := row.Scan(&i.ID, &i.UserID, &i.CompanyID, &i.InterviewDate, &i.Phase, &i.Result, &i.Memo, &i.CreatedAt, &i.UpdatedAt)
	return i, mapErr(err)
}

func collectInterviews(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]entity.Interview, error) {
	defer rows.Close()
	var out []entity.Interview
	for rows.Next() {
		i, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, mapErr(rows.Err())
}

// ListInterviews lists interviews.
func (r *InterviewRepo) ListInterviews(ctx context.Context, userID string, f repository.InterviewFilter, p repository.PageParams) (repository.Page[entity.Interview], error) {
	args := []any{userID}
	where := ` WHERE user_id = $1`
	if f.CompanyID != "" {
		args = append(args, f.CompanyID)
		where += fmt.Sprintf(" AND company_id = $%d::uuid", len(args))
	}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(" AND interview_date >= $%d", len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(" AND interview_date <= $%d", len(args))
	}
	cond, args, err := byCreated.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.Interview]{}, err
	}
	args = append(args, p.Limit+1)
	rows, err := r.DB.q(ctx).Query(ctx, `SELECT `+interviewCols+` FROM interviews`+where+cond+byCreated.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.Interview]{}, mapErr(err)
	}
	items, err := collectInterviews(rows)
	if err != nil {
		return repository.Page[entity.Interview]{}, err
	}
	return page(items, p.Limit, func(i entity.Interview) string { return encodeCursor(ts(i.CreatedAt), i.ID) }), nil
}

// GetInterview returns one interview.
func (r *InterviewRepo) GetInterview(ctx context.Context, userID, id string) (entity.Interview, error) {
	return scanInterview(r.DB.q(ctx).QueryRow(ctx, `SELECT `+interviewCols+` FROM interviews WHERE user_id = $1 AND id = $2`, userID, id))
}

// CreateInterview inserts an interview.
func (r *InterviewRepo) CreateInterview(ctx context.Context, i entity.Interview) (entity.Interview, error) {
	return scanInterview(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO interviews (id, user_id, company_id, interview_date, phase, result, memo) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+interviewCols,
		i.ID, i.UserID, i.CompanyID, i.InterviewDate, i.Phase, i.Result, i.Memo))
}

// UpdateInterview updates an interview.
func (r *InterviewRepo) UpdateInterview(ctx context.Context, i entity.Interview) (entity.Interview, error) {
	return scanInterview(r.DB.q(ctx).QueryRow(ctx,
		`UPDATE interviews SET company_id = $3, interview_date = $4, phase = $5, result = $6, memo = $7, updated_at = now()
		 WHERE user_id = $1 AND id = $2 RETURNING `+interviewCols,
		i.UserID, i.ID, i.CompanyID, i.InterviewDate, i.Phase, i.Result, i.Memo))
}

// DeleteInterview deletes an interview; logs cascade.
func (r *InterviewRepo) DeleteInterview(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM interviews WHERE user_id = $1 AND id = $2`, userID, id))
}

// UpcomingInterviews lists interviews on or after from, soonest first.
func (r *InterviewRepo) UpcomingInterviews(ctx context.Context, userID string, from time.Time, limit int) ([]entity.Interview, error) {
	rows, err := r.DB.q(ctx).Query(ctx,
		`SELECT `+interviewCols+` FROM interviews WHERE user_id = $1 AND interview_date >= $2::date ORDER BY interview_date, created_at LIMIT $3`,
		userID, from, limit)
	if err != nil {
		return nil, mapErr(err)
	}
	out, err := collectInterviews(rows)
	if out == nil {
		out = []entity.Interview{}
	}
	return out, err
}

const logCols = `id, interview_id, user_id, question_text, answer_text, self_evaluation, ai_feedback_text, ai_improved_answer_text, score, created_at, updated_at`

func scanLog(row interface{ Scan(...any) error }) (entity.InterviewLog, error) {
	var l entity.InterviewLog
	err := row.Scan(&l.ID, &l.InterviewID, &l.UserID, &l.QuestionText, &l.AnswerText, &l.SelfEvaluation, &l.AIFeedbackText, &l.AIImprovedAnswerText, &l.Score, &l.CreatedAt, &l.UpdatedAt)
	return l, mapErr(err)
}

func (r *InterviewRepo) queryLogs(ctx context.Context, sql string, args ...any) ([]entity.InterviewLog, error) {
	rows, err := r.DB.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []entity.InterviewLog{}
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, mapErr(rows.Err())
}

// ListLogs lists logs of an interview.
func (r *InterviewRepo) ListLogs(ctx context.Context, userID, interviewID string, p repository.PageParams) (repository.Page[entity.InterviewLog], error) {
	args := []any{userID}
	where := ` WHERE user_id = $1`
	if interviewID != "" {
		args = append(args, interviewID)
		where += fmt.Sprintf(" AND interview_id = $%d::uuid", len(args))
	}
	cond, args, err := byCreated.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.InterviewLog]{}, err
	}
	args = append(args, p.Limit+1)
	items, err := r.queryLogs(ctx, `SELECT `+logCols+` FROM interview_logs`+where+cond+byCreated.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.InterviewLog]{}, err
	}
	return page(items, p.Limit, func(l entity.InterviewLog) string { return encodeCursor(ts(l.CreatedAt), l.ID) }), nil
}

// GetLog returns one log.
func (r *InterviewRepo) GetLog(ctx context.Context, userID, id string) (entity.InterviewLog, error) {
	return scanLog(r.DB.q(ctx).QueryRow(ctx, `SELECT `+logCols+` FROM interview_logs WHERE user_id = $1 AND id = $2`, userID, id))
}

// CreateLog inserts a log.
func (r *InterviewRepo) CreateLog(ctx context.Context, l entity.InterviewLog) (entity.InterviewLog, error) {
	return scanLog(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO interview_logs (id, interview_id, user_id, question_text, answer_text, self_evaluation, ai_feedback_text, ai_improved_answer_text, score)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING `+logCols,
		l.ID, l.InterviewID, l.UserID, l.QuestionText, l.AnswerText, l.SelfEvaluation, l.AIFeedbackText, l.AIImprovedAnswerText, l.Score))
}

// UpdateLog updates a log.
func (r *InterviewRepo) UpdateLog(ctx context.Context, l entity.InterviewLog) (entity.InterviewLog, error) {
	return scanLog(r.DB.q(ctx).QueryRow(ctx,
		`UPDATE interview_logs SET question_text = $3, answer_text = $4, self_evaluation = $5, ai_feedback_text = $6,
		   ai_improved_answer_text = $7, score = $8, updated_at = now()
		 WHERE user_id = $1 AND id = $2 RETURNING `+logCols,
		l.UserID, l.ID, l.QuestionText, l.AnswerText, l.SelfEvaluation, l.AIFeedbackText, l.AIImprovedAnswerText, l.Score))
}

// DeleteLog deletes a log.
func (r *InterviewRepo) DeleteLog(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM interview_logs WHERE user_id = $1 AND id = $2`, userID, id))
}

// ReviewedLogs returns logs that have ai_feedback_text, newest first.
func (r *InterviewRepo) ReviewedLogs(ctx context.Context, userID string, interviewID *string, limit int) ([]entity.InterviewLog, error) {
	if interviewID != nil {
		return r.queryLogs(ctx,
			`SELECT `+logCols+` FROM interview_logs WHERE user_id = $1 AND interview_id = $2 AND ai_feedback_text IS NOT NULL AND ai_feedback_text <> ''
			 ORDER BY created_at DESC, id DESC LIMIT $3`, userID, *interviewID, limit)
	}
	return r.queryLogs(ctx,
		`SELECT `+logCols+` FROM interview_logs WHERE user_id = $1 AND ai_feedback_text IS NOT NULL AND ai_feedback_text <> ''
		 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
}

// RecentLogs returns the newest logs.
func (r *InterviewRepo) RecentLogs(ctx context.Context, userID string, limit int) ([]entity.InterviewLog, error) {
	return r.queryLogs(ctx, `SELECT `+logCols+` FROM interview_logs WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
}
