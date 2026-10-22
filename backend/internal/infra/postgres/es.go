package postgres

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
)

// ESRepo implements repository.ESRepository.
type ESRepo struct{ DB *DB }

const esProjectSelect = `SELECT p.id, p.user_id, p.company_id, p.status, p.deadline, p.created_at, p.updated_at,
  COALESCE((SELECT array_agg(ar.asset_id::text ORDER BY ar.asset_id) FROM es_project_asset_relations ar WHERE ar.es_project_id = p.id), '{}'),
  COALESCE((SELECT array_agg(mr.research_material_id::text ORDER BY mr.research_material_id) FROM es_project_research_material_relations mr WHERE mr.es_project_id = p.id), '{}')
FROM es_projects p`

func scanESProject(row interface{ Scan(...any) error }) (entity.ESProject, error) {
	var e entity.ESProject
	var status string
	err := row.Scan(&e.ID, &e.UserID, &e.CompanyID, &status, &e.Deadline, &e.CreatedAt, &e.UpdatedAt, &e.AssetIDs, &e.ResearchMaterialIDs)
	e.Status = statusOf(status)
	return e, mapErr(err)
}

// ListProjects lists projects.
func (r *ESRepo) ListProjects(ctx context.Context, userID string, f repository.ESFilter, p repository.PageParams) (repository.Page[entity.ESProject], error) {
	ks := byCreated.prefixed("p")
	args := []any{userID}
	where := ` WHERE p.user_id = $1`
	if f.CompanyID != "" {
		args = append(args, f.CompanyID)
		where += fmt.Sprintf(" AND p.company_id = $%d::uuid", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND p.status = $%d", len(args))
	}
	cond, args, err := ks.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.ESProject]{}, err
	}
	args = append(args, p.Limit+1)
	rows, err := r.DB.q(ctx).Query(ctx, esProjectSelect+where+cond+ks.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.ESProject]{}, mapErr(err)
	}
	defer rows.Close()
	var items []entity.ESProject
	for rows.Next() {
		e, err := scanESProject(rows)
		if err != nil {
			return repository.Page[entity.ESProject]{}, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return repository.Page[entity.ESProject]{}, mapErr(err)
	}
	return page(items, p.Limit, func(e entity.ESProject) string { return encodeCursor(ts(e.CreatedAt), e.ID) }), nil
}

// GetProject returns one project.
func (r *ESRepo) GetProject(ctx context.Context, userID, id string) (entity.ESProject, error) {
	return scanESProject(r.DB.q(ctx).QueryRow(ctx, esProjectSelect+` WHERE p.user_id = $1 AND p.id = $2`, userID, id))
}

func (r *ESRepo) replaceRelations(ctx context.Context, e entity.ESProject) error {
	q := r.DB.q(ctx)
	if e.AssetIDs != nil {
		if _, err := q.Exec(ctx, `DELETE FROM es_project_asset_relations WHERE es_project_id = $1`, e.ID); err != nil {
			return mapErr(err)
		}
		if len(e.AssetIDs) > 0 {
			if _, err := q.Exec(ctx, `INSERT INTO es_project_asset_relations (es_project_id, asset_id) SELECT $1, unnest($2::uuid[])`, e.ID, e.AssetIDs); err != nil {
				return mapErr(err)
			}
		}
	}
	if e.ResearchMaterialIDs != nil {
		if _, err := q.Exec(ctx, `DELETE FROM es_project_research_material_relations WHERE es_project_id = $1`, e.ID); err != nil {
			return mapErr(err)
		}
		if len(e.ResearchMaterialIDs) > 0 {
			if _, err := q.Exec(ctx, `INSERT INTO es_project_research_material_relations (es_project_id, research_material_id) SELECT $1, unnest($2::uuid[])`, e.ID, e.ResearchMaterialIDs); err != nil {
				return mapErr(err)
			}
		}
	}
	return nil
}

// CreateProject inserts a project and its relations.
func (r *ESRepo) CreateProject(ctx context.Context, e entity.ESProject) (entity.ESProject, error) {
	if _, err := r.DB.q(ctx).Exec(ctx,
		`INSERT INTO es_projects (id, user_id, company_id, status, deadline) VALUES ($1, $2, $3, $4, $5)`,
		e.ID, e.UserID, e.CompanyID, string(e.Status), e.Deadline); err != nil {
		return entity.ESProject{}, mapErr(err)
	}
	if err := r.replaceRelations(ctx, e); err != nil {
		return entity.ESProject{}, err
	}
	return r.GetProject(ctx, e.UserID, e.ID)
}

// UpdateProject updates a project. Nil relation slices are kept.
func (r *ESRepo) UpdateProject(ctx context.Context, e entity.ESProject) (entity.ESProject, error) {
	if err := affected(r.DB.q(ctx).Exec(ctx,
		`UPDATE es_projects SET status = $3, deadline = $4, updated_at = now() WHERE user_id = $1 AND id = $2`,
		e.UserID, e.ID, string(e.Status), e.Deadline)); err != nil {
		return entity.ESProject{}, err
	}
	if err := r.replaceRelations(ctx, e); err != nil {
		return entity.ESProject{}, err
	}
	return r.GetProject(ctx, e.UserID, e.ID)
}

// DeleteProject deletes a project. Sections, versions and share links cascade.
func (r *ESRepo) DeleteProject(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM es_projects WHERE user_id = $1 AND id = $2`, userID, id))
}

const esSectionCols = `id, es_project_id, user_id, title, content_draft, content_final, character_limit, version, created_at, updated_at`

func scanESSection(row interface{ Scan(...any) error }) (entity.ESSection, error) {
	var s entity.ESSection
	err := row.Scan(&s.ID, &s.ESProjectID, &s.UserID, &s.Title, &s.ContentDraft, &s.ContentFinal, &s.CharacterLimit, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	return s, mapErr(err)
}

// ListSections lists sections of a project in creation order.
func (r *ESRepo) ListSections(ctx context.Context, userID, projectID string) ([]entity.ESSection, error) {
	rows, err := r.DB.q(ctx).Query(ctx,
		`SELECT `+esSectionCols+` FROM es_sections WHERE user_id = $1 AND es_project_id = $2 ORDER BY created_at, id`, userID, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []entity.ESSection{}
	for rows.Next() {
		s, err := scanESSection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, mapErr(rows.Err())
}

// GetSection returns one section.
func (r *ESRepo) GetSection(ctx context.Context, userID, id string) (entity.ESSection, error) {
	return scanESSection(r.DB.q(ctx).QueryRow(ctx, `SELECT `+esSectionCols+` FROM es_sections WHERE user_id = $1 AND id = $2`, userID, id))
}

// CreateSection inserts a section.
func (r *ESRepo) CreateSection(ctx context.Context, s entity.ESSection) (entity.ESSection, error) {
	return scanESSection(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO es_sections (id, es_project_id, user_id, title, content_draft, content_final, character_limit, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+esSectionCols,
		s.ID, s.ESProjectID, s.UserID, s.Title, s.ContentDraft, s.ContentFinal, s.CharacterLimit, s.Version))
}

// UpdateSection updates a section.
func (r *ESRepo) UpdateSection(ctx context.Context, s entity.ESSection) (entity.ESSection, error) {
	return scanESSection(r.DB.q(ctx).QueryRow(ctx,
		`UPDATE es_sections SET title = $3, content_draft = $4, content_final = $5, character_limit = $6, version = $7, updated_at = now()
		 WHERE user_id = $1 AND id = $2 RETURNING `+esSectionCols,
		s.UserID, s.ID, s.Title, s.ContentDraft, s.ContentFinal, s.CharacterLimit, s.Version))
}

// DeleteSection deletes a section; its versions cascade.
func (r *ESRepo) DeleteSection(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM es_sections WHERE user_id = $1 AND id = $2`, userID, id))
}

const esVersionCols = `id, es_section_id, user_id, version, content, source, created_at`

func scanESVersion(row interface{ Scan(...any) error }) (entity.ESSectionVersion, error) {
	var v entity.ESSectionVersion
	var source string
	err := row.Scan(&v.ID, &v.ESSectionID, &v.UserID, &v.Version, &v.Content, &source, &v.CreatedAt)
	v.Source = sourceOf(source)
	return v, mapErr(err)
}

// ListVersions lists versions newest first.
func (r *ESRepo) ListVersions(ctx context.Context, userID, sectionID string) ([]entity.ESSectionVersion, error) {
	rows, err := r.DB.q(ctx).Query(ctx,
		`SELECT `+esVersionCols+` FROM es_section_versions WHERE user_id = $1 AND es_section_id = $2 ORDER BY created_at DESC, id DESC`,
		userID, sectionID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []entity.ESSectionVersion{}
	for rows.Next() {
		v, err := scanESVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, mapErr(rows.Err())
}

// GetVersion returns one version of a section.
func (r *ESRepo) GetVersion(ctx context.Context, userID, sectionID, versionID string) (entity.ESSectionVersion, error) {
	return scanESVersion(r.DB.q(ctx).QueryRow(ctx,
		`SELECT `+esVersionCols+` FROM es_section_versions WHERE user_id = $1 AND es_section_id = $2 AND id = $3`,
		userID, sectionID, versionID))
}

// CreateVersion inserts a version. created_at uses clock_timestamp() so versions
// created in one transaction keep their order.
func (r *ESRepo) CreateVersion(ctx context.Context, v entity.ESSectionVersion) (entity.ESSectionVersion, error) {
	return scanESVersion(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO es_section_versions (id, es_section_id, user_id, version, content, source, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, clock_timestamp()) RETURNING `+esVersionCols,
		v.ID, v.ESSectionID, v.UserID, v.Version, v.Content, string(v.Source)))
}

// PruneVersions keeps only the newest keep versions.
func (r *ESRepo) PruneVersions(ctx context.Context, userID, sectionID string, keep int) error {
	_, err := r.DB.q(ctx).Exec(ctx,
		`DELETE FROM es_section_versions WHERE user_id = $1 AND es_section_id = $2 AND id NOT IN (
		   SELECT id FROM es_section_versions WHERE user_id = $1 AND es_section_id = $2
		   ORDER BY created_at DESC, id DESC LIMIT $3)`,
		userID, sectionID, keep)
	return mapErr(err)
}
