package postgres

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
)

// CompanyRepo implements repository.CompanyRepository.
type CompanyRepo struct{ DB *DB }

const companyCols = `id, user_id, name, job_title, selection_phase, motivation_memo, reference_url, created_at, updated_at`

func scanCompany(row interface{ Scan(...any) error }) (entity.Company, error) {
	var c entity.Company
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.JobTitle, &c.SelectionPhase, &c.MotivationMemo, &c.ReferenceURL, &c.CreatedAt, &c.UpdatedAt)
	return c, mapErr(err)
}

// List lists non-deleted companies.
func (r *CompanyRepo) List(ctx context.Context, userID string, f repository.CompanyFilter, p repository.PageParams) (repository.Page[entity.Company], error) {
	args := []any{userID}
	where := ` WHERE user_id = $1 AND deleted_at IS NULL`
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where += fmt.Sprintf(" AND (name ILIKE $%d OR job_title ILIKE $%d)", len(args), len(args))
	}
	if f.SelectionPhase != "" {
		args = append(args, f.SelectionPhase)
		where += fmt.Sprintf(" AND selection_phase = $%d", len(args))
	}
	cond, args, err := byCreated.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.Company]{}, err
	}
	args = append(args, p.Limit+1)
	rows, err := r.DB.q(ctx).Query(ctx, `SELECT `+companyCols+` FROM companies`+where+cond+byCreated.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.Company]{}, mapErr(err)
	}
	defer rows.Close()
	var items []entity.Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return repository.Page[entity.Company]{}, err
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return repository.Page[entity.Company]{}, mapErr(err)
	}
	return page(items, p.Limit, func(c entity.Company) string { return encodeCursor(ts(c.CreatedAt), c.ID) }), nil
}

// Get returns one non-deleted company.
func (r *CompanyRepo) Get(ctx context.Context, userID, id string) (entity.Company, error) {
	return scanCompany(r.DB.q(ctx).QueryRow(ctx,
		`SELECT `+companyCols+` FROM companies WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`, userID, id))
}

// Create inserts a company.
func (r *CompanyRepo) Create(ctx context.Context, c entity.Company) (entity.Company, error) {
	return scanCompany(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO companies (id, user_id, name, job_title, selection_phase, motivation_memo, reference_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+companyCols,
		c.ID, c.UserID, c.Name, c.JobTitle, c.SelectionPhase, c.MotivationMemo, c.ReferenceURL))
}

// Update updates a company.
func (r *CompanyRepo) Update(ctx context.Context, c entity.Company) (entity.Company, error) {
	return scanCompany(r.DB.q(ctx).QueryRow(ctx,
		`UPDATE companies SET name = $3, job_title = $4, selection_phase = $5, motivation_memo = $6, reference_url = $7, updated_at = now()
		 WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL RETURNING `+companyCols,
		c.UserID, c.ID, c.Name, c.JobTitle, c.SelectionPhase, c.MotivationMemo, c.ReferenceURL))
}

// SoftDelete sets deleted_at.
func (r *CompanyRepo) SoftDelete(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx,
		`UPDATE companies SET deleted_at = now() WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`, userID, id))
}

// ResearchRepo implements repository.ResearchRepository.
type ResearchRepo struct{ DB *DB }

const researchCols = `id, user_id, company_id, title, type, source_url, content_raw, summary, created_at, updated_at`

func scanResearch(row interface{ Scan(...any) error }) (entity.ResearchMaterial, error) {
	var m entity.ResearchMaterial
	err := row.Scan(&m.ID, &m.UserID, &m.CompanyID, &m.Title, &m.Type, &m.SourceURL, &m.ContentRaw, &m.Summary, &m.CreatedAt, &m.UpdatedAt)
	return m, mapErr(err)
}

// List lists materials of a company.
func (r *ResearchRepo) List(ctx context.Context, userID, companyID string, p repository.PageParams) (repository.Page[entity.ResearchMaterial], error) {
	cond, args, err := byCreated.where(p.Cursor, []any{userID, companyID})
	if err != nil {
		return repository.Page[entity.ResearchMaterial]{}, err
	}
	args = append(args, p.Limit+1)
	rows, err := r.DB.q(ctx).Query(ctx,
		`SELECT `+researchCols+` FROM research_materials WHERE user_id = $1 AND company_id = $2`+cond+byCreated.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.ResearchMaterial]{}, mapErr(err)
	}
	defer rows.Close()
	var items []entity.ResearchMaterial
	for rows.Next() {
		m, err := scanResearch(rows)
		if err != nil {
			return repository.Page[entity.ResearchMaterial]{}, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return repository.Page[entity.ResearchMaterial]{}, mapErr(err)
	}
	return page(items, p.Limit, func(m entity.ResearchMaterial) string { return encodeCursor(ts(m.CreatedAt), m.ID) }), nil
}

// Get returns one material.
func (r *ResearchRepo) Get(ctx context.Context, userID, id string) (entity.ResearchMaterial, error) {
	return scanResearch(r.DB.q(ctx).QueryRow(ctx, `SELECT `+researchCols+` FROM research_materials WHERE user_id = $1 AND id = $2`, userID, id))
}

// GetMany returns the user's materials among ids, in the order of ids.
func (r *ResearchRepo) GetMany(ctx context.Context, userID string, ids []string) ([]entity.ResearchMaterial, error) {
	rows, err := r.DB.q(ctx).Query(ctx,
		`SELECT `+researchCols+` FROM research_materials WHERE user_id = $1 AND id = ANY($2::uuid[]) ORDER BY array_position($2::uuid[], id)`,
		userID, nonNil(ids))
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.ResearchMaterial
	for rows.Next() {
		m, err := scanResearch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, mapErr(rows.Err())
}

// Create inserts a material.
func (r *ResearchRepo) Create(ctx context.Context, m entity.ResearchMaterial) (entity.ResearchMaterial, error) {
	return scanResearch(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO research_materials (id, user_id, company_id, title, type, source_url, content_raw, summary)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+researchCols,
		m.ID, m.UserID, m.CompanyID, m.Title, m.Type, m.SourceURL, m.ContentRaw, m.Summary))
}

// Update updates a material.
func (r *ResearchRepo) Update(ctx context.Context, m entity.ResearchMaterial) (entity.ResearchMaterial, error) {
	return scanResearch(r.DB.q(ctx).QueryRow(ctx,
		`UPDATE research_materials SET title = $3, type = $4, source_url = $5, content_raw = $6, summary = $7, updated_at = now()
		 WHERE user_id = $1 AND id = $2 RETURNING `+researchCols,
		m.UserID, m.ID, m.Title, m.Type, m.SourceURL, m.ContentRaw, m.Summary))
}

// Delete deletes a material.
func (r *ResearchRepo) Delete(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM research_materials WHERE user_id = $1 AND id = $2`, userID, id))
}

// CountOwned counts the user's materials among ids.
func (r *ResearchRepo) CountOwned(ctx context.Context, userID string, ids []string) (int, error) {
	var n int
	err := r.DB.q(ctx).QueryRow(ctx, `SELECT count(*) FROM research_materials WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, nonNil(ids)).Scan(&n)
	return n, mapErr(err)
}
