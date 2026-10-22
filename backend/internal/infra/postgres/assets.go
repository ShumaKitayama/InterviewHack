package postgres

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/repository"
)

// AssetRepo implements repository.AssetRepository.
type AssetRepo struct{ DB *DB }

const assetSelect = `SELECT a.id, a.user_id, a.type, a.title, a.content, a.created_at, a.updated_at,
  COALESCE(array_agg(t.id::text ORDER BY t.name) FILTER (WHERE t.id IS NOT NULL), '{}') AS tag_ids,
  COALESCE(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.id IS NOT NULL), '{}') AS tag_names
FROM assets a
LEFT JOIN asset_tag_relations r ON r.asset_id = a.id
LEFT JOIN asset_tags t ON t.id = r.tag_id AND t.user_id = a.user_id`

func scanAsset(row interface{ Scan(...any) error }) (entity.Asset, error) {
	var a entity.Asset
	err := row.Scan(&a.ID, &a.UserID, &a.Type, &a.Title, &a.Content, &a.CreatedAt, &a.UpdatedAt, &a.TagIDs, &a.TagNames)
	return a, mapErr(err)
}

// List lists non-deleted assets of the user.
func (r *AssetRepo) List(ctx context.Context, userID string, f repository.AssetFilter, p repository.PageParams) (repository.Page[entity.Asset], error) {
	ks := byCreated.prefixed("a")
	if f.SortBy == "updated" {
		ks = byUpdated.prefixed("a")
	}
	args := []any{userID}
	where := ` WHERE a.user_id = $1 AND a.deleted_at IS NULL`
	if f.Type != "" {
		args = append(args, f.Type)
		where += fmt.Sprintf(" AND a.type = $%d", len(args))
	}
	if f.TagID != "" {
		args = append(args, f.TagID)
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM asset_tag_relations x WHERE x.asset_id = a.id AND x.tag_id = $%d::uuid)", len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where += fmt.Sprintf(" AND (a.title ILIKE $%d OR a.content ILIKE $%d)", len(args), len(args))
	}
	cond, args, err := ks.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.Asset]{}, err
	}
	args = append(args, p.Limit+1)
	rows, err := r.DB.q(ctx).Query(ctx, assetSelect+where+cond+" GROUP BY a.id"+ks.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.Asset]{}, mapErr(err)
	}
	defer rows.Close()
	var items []entity.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return repository.Page[entity.Asset]{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return repository.Page[entity.Asset]{}, mapErr(err)
	}
	return page(items, p.Limit, func(a entity.Asset) string {
		if f.SortBy == "updated" {
			return encodeCursor(ts(a.UpdatedAt), a.ID)
		}
		return encodeCursor(ts(a.CreatedAt), a.ID)
	}), nil
}

// Get returns one non-deleted asset.
func (r *AssetRepo) Get(ctx context.Context, userID, id string) (entity.Asset, error) {
	return scanAsset(r.DB.q(ctx).QueryRow(ctx,
		assetSelect+` WHERE a.user_id = $1 AND a.id = $2 AND a.deleted_at IS NULL GROUP BY a.id`, userID, id))
}

// GetMany returns the user's non-deleted assets among ids, in the order of ids.
func (r *AssetRepo) GetMany(ctx context.Context, userID string, ids []string) ([]entity.Asset, error) {
	rows, err := r.DB.q(ctx).Query(ctx,
		assetSelect+` WHERE a.user_id = $1 AND a.id = ANY($2::uuid[]) AND a.deleted_at IS NULL
		GROUP BY a.id ORDER BY array_position($2::uuid[], a.id)`, userID, nonNil(ids))
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []entity.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, mapErr(rows.Err())
}

func (r *AssetRepo) replaceTags(ctx context.Context, assetID string, tagIDs []string) error {
	q := r.DB.q(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM asset_tag_relations WHERE asset_id = $1`, assetID); err != nil {
		return mapErr(err)
	}
	if len(tagIDs) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `INSERT INTO asset_tag_relations (asset_id, tag_id) SELECT $1, unnest($2::uuid[])`, assetID, tagIDs)
	return mapErr(err)
}

// Create inserts an asset and its tag relations.
func (r *AssetRepo) Create(ctx context.Context, a entity.Asset) (entity.Asset, error) {
	if _, err := r.DB.q(ctx).Exec(ctx,
		`INSERT INTO assets (id, user_id, type, title, content) VALUES ($1, $2, $3, $4, $5)`,
		a.ID, a.UserID, a.Type, a.Title, a.Content); err != nil {
		return entity.Asset{}, mapErr(err)
	}
	if err := r.replaceTags(ctx, a.ID, a.TagIDs); err != nil {
		return entity.Asset{}, err
	}
	return r.Get(ctx, a.UserID, a.ID)
}

// Update updates an asset. TagIDs == nil keeps the current tags.
func (r *AssetRepo) Update(ctx context.Context, a entity.Asset) (entity.Asset, error) {
	if err := affected(r.DB.q(ctx).Exec(ctx,
		`UPDATE assets SET type = $3, title = $4, content = $5, updated_at = now()
		 WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`,
		a.UserID, a.ID, a.Type, a.Title, a.Content)); err != nil {
		return entity.Asset{}, err
	}
	if a.TagIDs != nil {
		if err := r.replaceTags(ctx, a.ID, a.TagIDs); err != nil {
			return entity.Asset{}, err
		}
	}
	return r.Get(ctx, a.UserID, a.ID)
}

// SoftDelete sets deleted_at.
func (r *AssetRepo) SoftDelete(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx,
		`UPDATE assets SET deleted_at = now() WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`, userID, id))
}

// CountOwned counts the user's non-deleted assets among ids.
func (r *AssetRepo) CountOwned(ctx context.Context, userID string, ids []string) (int, error) {
	var n int
	err := r.DB.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM assets WHERE user_id = $1 AND id = ANY($2::uuid[]) AND deleted_at IS NULL`, userID, nonNil(ids)).Scan(&n)
	return n, mapErr(err)
}

// TagRepo implements repository.TagRepository.
type TagRepo struct{ DB *DB }

// List lists the user's tags by name.
func (r *TagRepo) List(ctx context.Context, userID string) ([]entity.AssetTag, error) {
	rows, err := r.DB.q(ctx).Query(ctx, `SELECT id, user_id, name FROM asset_tags WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []entity.AssetTag{}
	for rows.Next() {
		var t entity.AssetTag
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, t)
	}
	return out, mapErr(rows.Err())
}

// Create inserts a tag. Duplicate names map to ErrConflict.
func (r *TagRepo) Create(ctx context.Context, t entity.AssetTag) (entity.AssetTag, error) {
	_, err := r.DB.q(ctx).Exec(ctx, `INSERT INTO asset_tags (id, user_id, name) VALUES ($1, $2, $3)`, t.ID, t.UserID, t.Name)
	return t, mapErr(err)
}

// Update renames a tag.
func (r *TagRepo) Update(ctx context.Context, t entity.AssetTag) (entity.AssetTag, error) {
	return t, affected(r.DB.q(ctx).Exec(ctx, `UPDATE asset_tags SET name = $3 WHERE user_id = $1 AND id = $2`, t.UserID, t.ID, t.Name))
}

// Delete deletes a tag (relations cascade).
func (r *TagRepo) Delete(ctx context.Context, userID, id string) error {
	return affected(r.DB.q(ctx).Exec(ctx, `DELETE FROM asset_tags WHERE user_id = $1 AND id = $2`, userID, id))
}

// CountOwned counts the user's tags among ids.
func (r *TagRepo) CountOwned(ctx context.Context, userID string, ids []string) (int, error) {
	var n int
	err := r.DB.q(ctx).QueryRow(ctx, `SELECT count(*) FROM asset_tags WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, nonNil(ids)).Scan(&n)
	return n, mapErr(err)
}
