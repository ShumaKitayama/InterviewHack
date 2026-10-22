package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/interviewhack/backend/internal/domain/entity"
)

// UserRepo implements repository.UserRepository.
type UserRepo struct{ DB *DB }

const userCols = `id, firebase_uid, email, display_name, created_at, updated_at, deleted_at`

func scanUser(row interface{ Scan(...any) error }) (entity.User, error) {
	var u entity.User
	err := row.Scan(&u.ID, &u.FirebaseUID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	return u, mapErr(err)
}

// GetOrCreateByFirebaseUID upserts by firebase_uid. The email is synced only for
// users without deleted_at (7.4 step 4, IT-AUTH-008).
func (r *UserRepo) GetOrCreateByFirebaseUID(ctx context.Context, uid string, email *string) (entity.User, error) {
	q := r.DB.q(ctx)
	if _, err := q.Exec(ctx,
		`INSERT INTO users (id, firebase_uid, email) VALUES ($1, $2, $3) ON CONFLICT (firebase_uid) DO NOTHING`,
		uuid.NewString(), uid, email); err != nil {
		return entity.User{}, mapErr(err)
	}
	u, err := scanUser(q.QueryRow(ctx,
		`UPDATE users SET email = $2, updated_at = now()
		 WHERE firebase_uid = $1 AND deleted_at IS NULL AND email IS DISTINCT FROM $2
		 RETURNING `+userCols, uid, email))
	if err == nil {
		return u, nil
	}
	return scanUser(q.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE firebase_uid = $1`, uid))
}

// Get returns a non-deleted user.
func (r *UserRepo) Get(ctx context.Context, userID string) (entity.User, error) {
	return scanUser(r.DB.q(ctx).QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1 AND deleted_at IS NULL`, userID))
}

// UpdateDisplayName updates display_name.
func (r *UserRepo) UpdateDisplayName(ctx context.Context, userID string, displayName *string) (entity.User, error) {
	return scanUser(r.DB.q(ctx).QueryRow(ctx,
		`UPDATE users SET display_name = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL RETURNING `+userCols,
		userID, displayName))
}

// DeleteAccount physically deletes every row of the user's domain data, then marks
// the users row deleted with PII cleared. Callers run it inside a transaction (6.3.3).
func (r *UserRepo) DeleteAccount(ctx context.Context, userID string, now time.Time) error {
	q := r.DB.q(ctx)
	stmts := []string{
		`DELETE FROM prompt_sessions WHERE user_id = $1`,
		`DELETE FROM dashboard_summaries WHERE user_id = $1`,
		`DELETE FROM share_links WHERE user_id = $1`,
		`DELETE FROM qa WHERE user_id = $1`,
		`DELETE FROM interview_logs WHERE user_id = $1`,
		`DELETE FROM interviews WHERE user_id = $1`,
		`DELETE FROM es_section_versions WHERE user_id = $1`,
		`DELETE FROM es_sections WHERE user_id = $1`,
		`DELETE FROM es_projects WHERE user_id = $1`,
		`DELETE FROM research_materials WHERE user_id = $1`,
		`DELETE FROM asset_tags WHERE user_id = $1`,
		`DELETE FROM assets WHERE user_id = $1`,
		`DELETE FROM companies WHERE user_id = $1`,
	}
	for _, s := range stmts {
		if _, err := q.Exec(ctx, s, userID); err != nil {
			return mapErr(err)
		}
	}
	return affected(q.Exec(ctx,
		`UPDATE users SET deleted_at = $2, email = NULL, display_name = NULL, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL`,
		userID, now))
}
