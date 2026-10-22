package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
)

// QARepo implements repository.QARepository.
type QARepo struct{ DB *DB }

const qaSelect = `SELECT q.id, q.user_id, q.company_id, q.question_text, q.answer_text, q.source_type, q.is_favorite, q.review_status,
  q.created_at, q.updated_at,
  COALESCE((SELECT array_agg(r.asset_id::text ORDER BY r.asset_id) FROM qa_asset_relations r WHERE r.qa_id = q.id), '{}')
FROM qa q`

// Favorites first, then newest (2.2.9).
var qaKeyset = keyset{cols: []string{"q.is_favorite::int", "q.created_at", "q.id"}, casts: []string{"int", "timestamptz", "uuid"}}

func scanQA(row interface{ Scan(...any) error }) (entity.QA, error) {
	var q entity.QA
	var source, review string
	err := row.Scan(&q.ID, &q.UserID, &q.CompanyID, &q.QuestionText, &q.AnswerText, &source, &q.IsFavorite, &review, &q.CreatedAt, &q.UpdatedAt, &q.AssetIDs)
	q.SourceType, q.ReviewStatus = vo.QASourceType(source), vo.ReviewStatus(review)
	return q, mapErr(err)
}

func (r *QARepo) query(ctx context.Context, sql string, args ...any) ([]entity.QA, error) {
	rows, err := r.DB.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.QA
	for rows.Next() {
		q, err := scanQA(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, mapErr(rows.Err())
}

// List lists Q&A, favorites first.
func (r *QARepo) List(ctx context.Context, userID string, f repository.QAFilter, p repository.PageParams) (repository.Page[entity.QA], error) {
	args := []any{userID}
	where := ` WHERE q.user_id = $1`
	if f.CompanyID != "" {
		args = append(args, f.CompanyID)
		where += fmt.Sprintf(" AND q.company_id = $%d::uuid", len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where += fmt.Sprintf(" AND (q.question_text ILIKE $%d OR q.answer_text ILIKE $%d)", len(args), len(args))
	}
	if f.IsFavorite != nil {
		args = append(args, *f.IsFavorite)
		where += fmt.Sprintf(" AND q.is_favorite = $%d", len(args))
	}
	if f.ReviewStatus != "" {
		args = append(args, f.ReviewStatus)
		where += fmt.Sprintf(" AND q.review_status = $%d", len(args))
	}
	cond, args, err := qaKeyset.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.QA]{}, err
	}
	args = append(args, p.Limit+1)
	items, err := r.query(ctx, qaSelect+where+cond+qaKeyset.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.QA]{}, err
	}
	return page(items, p.Limit, func(q entity.QA) string {
		fav := 0
		if q.IsFavorite {
			fav = 1
		}
		return encodeCursor(strconv.Itoa(fav), ts(q.CreatedAt), q.ID)
	}), nil
}

// Get returns one Q&A.
func (r *QARepo) Get(ctx context.Context, userID, id string) (entity.QA, error) {
	return scanQA(r.DB.q(ctx).QueryRow(ctx, qaSelect+` WHERE q.user_id = $1 AND q.id = $2`, userID, id))
}

func (r *QARepo) replaceAssets(ctx context.Context, id string, assetIDs []string) error {
	q := r.DB.q(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM qa_asset_relations WHERE qa_id = $1`, id); err != nil {
		return mapErr(err)
	}
	if len(assetIDs) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `INSERT INTO qa_asset_relations (qa_id, asset_id) SELECT $1, unnest($2::uuid[])`, id, assetIDs)
	return mapErr(err)
}

// Create inserts a Q&A and its asset relations.
func (r *QARepo) Create(ctx context.Context, q entity.QA) (entity.QA, error) {
	if _, err := r.DB.q(ctx).Exec(ctx,
		`INSERT INTO qa (id, user_id, company_id, question_text, answer_text, source_type, is_favorite, review_status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		q.ID, q.UserID, q.CompanyID, q.QuestionText, q.AnswerText, string(q.SourceType), q.IsFavorite, string(q.ReviewStatus)); err != nil {
		return entity.QA{}, mapErr(err)
	}
	if err := r.replaceAssets(ctx, q.ID, q.AssetIDs); err != nil {
		return entity.QA{}, err
	}
	return r.Get(ctx, q.UserID, q.ID)
}

// Update updates a Q&A. AssetIDs == nil keeps relations.
func (r *QARepo) Update(ctx context.Context, q entity.QA) (entity.QA, error) {
	if err := affected(r.DB.q(ctx).Exec(ctx,
		`UPDATE qa SET company_id = $3, question_text = $4, answer_text = $5, source_type = $6, is_favorite = $7, review_status = $8, updated_at = now()
		 WHERE user_id = $1 AND id = $2`,
		q.UserID, q.ID, q.CompanyID, q.QuestionText, q.AnswerText, string(q.SourceType), q.IsFavorite, string(q.ReviewStatus))); err != nil {
		return entity.QA{}, err
	}
	if q.AssetIDs != nil {
		if err := r.replaceAssets(ctx, q.ID, q.AssetIDs); err != nil {
			return entity.QA{}, err
		}
	}
	return r.Get(ctx, q.UserID, q.ID)
}

// Delete deletes a Q&A.
func (r *QARepo) Delete(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM qa WHERE user_id = $1 AND id = $2`, userID, id))
}

// Recent returns the newest Q&A.
func (r *QARepo) Recent(ctx context.Context, userID string, limit int) ([]entity.QA, error) {
	out, err := r.query(ctx, qaSelect+` WHERE q.user_id = $1 ORDER BY q.created_at DESC, q.id DESC LIMIT $2`, userID, limit)
	if out == nil {
		out = []entity.QA{}
	}
	return out, err
}
