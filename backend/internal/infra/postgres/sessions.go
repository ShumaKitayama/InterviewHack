package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
)

// PromptSessionRepo implements repository.PromptSessionRepository.
type PromptSessionRepo struct{ DB *DB }

const sessionCols = `id, user_id, use_case, template_version, company_id, es_project_id, es_section_id, interview_id, interview_log_id,
  input_refs, status, parse_result, created_at, pasted_at, completed_at`

func scanSession(row interface{ Scan(...any) error }) (entity.PromptSession, error) {
	var s entity.PromptSession
	var uc, status string
	var parse *string
	var refs []byte
	err := row.Scan(&s.ID, &s.UserID, &uc, &s.TemplateVersion, &s.CompanyID, &s.ESProjectID, &s.ESSectionID, &s.InterviewID, &s.InterviewLogID,
		&refs, &status, &parse, &s.CreatedAt, &s.PastedAt, &s.CompletedAt)
	if err != nil {
		return s, mapErr(err)
	}
	s.UseCase, s.Status = vo.UseCase(uc), vo.SessionStatus(status)
	if parse != nil {
		pr := vo.ParseResult(*parse)
		s.ParseResult = &pr
	}
	if len(refs) > 0 {
		if err := json.Unmarshal(refs, &s.InputRefs); err != nil {
			return s, fmt.Errorf("decode input_refs: %w", err)
		}
	}
	return s, nil
}

// Create inserts a session. input_refs holds only IDs and options (D13).
func (r *PromptSessionRepo) Create(ctx context.Context, s entity.PromptSession) (entity.PromptSession, error) {
	refs, err := json.Marshal(s.InputRefs)
	if err != nil {
		return entity.PromptSession{}, err
	}
	return scanSession(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO prompt_sessions (id, user_id, use_case, template_version, company_id, es_project_id, es_section_id, interview_id, interview_log_id, input_refs, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING `+sessionCols,
		s.ID, s.UserID, string(s.UseCase), s.TemplateVersion, s.CompanyID, s.ESProjectID, s.ESSectionID, s.InterviewID, s.InterviewLogID,
		refs, string(s.Status)))
}

// Get returns one session.
func (r *PromptSessionRepo) Get(ctx context.Context, userID, id string) (entity.PromptSession, error) {
	return scanSession(r.DB.q(ctx).QueryRow(ctx, `SELECT `+sessionCols+` FROM prompt_sessions WHERE user_id = $1 AND id = $2`, userID, id))
}

// sessionListSelect adds the target's display name (7.5.5 対象の表示名).
var sessionListSelect = `SELECT ` + prefixCols("ps", sessionCols) + `,
  COALESCE(sec.title, il.question_text, c.name, '')
FROM prompt_sessions ps
LEFT JOIN es_sections sec ON sec.id = ps.es_section_id AND sec.user_id = ps.user_id
LEFT JOIN interview_logs il ON il.id = ps.interview_log_id AND il.user_id = ps.user_id
LEFT JOIN companies c ON c.id = ps.company_id AND c.user_id = ps.user_id`

func prefixCols(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for i, c := range parts {
		parts[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(parts, ", ")
}

type labeledRow struct {
	row   interface{ Scan(...any) error }
	label *string
}

func (l labeledRow) Scan(dest ...any) error { return l.row.Scan(append(dest, l.label)...) }

// List lists sessions with their target labels, newest first.
func (r *PromptSessionRepo) List(ctx context.Context, userID string, statuses []vo.SessionStatus, p repository.PageParams) (repository.Page[entity.PromptSession], error) {
	ks := byCreated.prefixed("ps")
	args := []any{userID}
	where := ` WHERE ps.user_id = $1`
	if len(statuses) > 0 {
		args = append(args, statusStrings(statuses))
		where += fmt.Sprintf(" AND ps.status = ANY($%d::text[])", len(args))
	}
	cond, args, err := ks.where(p.Cursor, args)
	if err != nil {
		return repository.Page[entity.PromptSession]{}, err
	}
	args = append(args, p.Limit+1)
	rows, err := r.DB.q(ctx).Query(ctx, sessionListSelect+where+cond+ks.orderBy()+fmt.Sprintf(" LIMIT $%d", len(args)), args...)
	if err != nil {
		return repository.Page[entity.PromptSession]{}, mapErr(err)
	}
	defer rows.Close()
	var items []entity.PromptSession
	for rows.Next() {
		var label string
		s, err := scanSession(labeledRow{row: rows, label: &label})
		if err != nil {
			return repository.Page[entity.PromptSession]{}, err
		}
		s.TargetLabel = label
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return repository.Page[entity.PromptSession]{}, mapErr(err)
	}
	return page(items, p.Limit, func(s entity.PromptSession) string { return encodeCursor(ts(s.CreatedAt), s.ID) }), nil
}

// Transition is the conditional UPDATE of 7.6.1. It returns false when the row
// does not exist, belongs to another user, or is not in one of t.From.
func (r *PromptSessionRepo) Transition(ctx context.Context, userID, id string, t repository.TransitionParams) (bool, error) {
	sets := []string{"status = $3"}
	args := []any{userID, id, string(t.To), statusStrings(t.From)}
	if t.ParseResult != nil {
		args = append(args, string(*t.ParseResult))
		sets = append(sets, fmt.Sprintf("parse_result = $%d", len(args)))
	}
	if t.SetPastedAt {
		args = append(args, t.Now)
		sets = append(sets, fmt.Sprintf("pasted_at = $%d", len(args)))
	}
	if t.SetCompleted {
		args = append(args, t.Now)
		sets = append(sets, fmt.Sprintf("completed_at = $%d", len(args)))
	}
	tag, err := r.DB.q(ctx).Exec(ctx,
		`UPDATE prompt_sessions SET `+strings.Join(sets, ", ")+` WHERE user_id = $1 AND id = $2 AND status = ANY($4::text[])`, args...)
	if err != nil {
		return false, mapErr(err)
	}
	return tag.RowsAffected() == 1, nil
}

// ExpireUnfinished abandons created/pasted sessions created at or before cutoff (7.9).
func (r *PromptSessionRepo) ExpireUnfinished(ctx context.Context, userID string, cutoff, now time.Time) error {
	_, err := r.DB.q(ctx).Exec(ctx,
		`UPDATE prompt_sessions SET status = 'abandoned', completed_at = $3
		 WHERE user_id = $1 AND status IN ('created', 'pasted') AND created_at <= $2`, userID, cutoff, now)
	return mapErr(err)
}

// CountUnfinished counts created/pasted sessions.
func (r *PromptSessionRepo) CountUnfinished(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.DB.q(ctx).QueryRow(ctx, `SELECT count(*) FROM prompt_sessions WHERE user_id = $1 AND status IN ('created', 'pasted')`, userID).Scan(&n)
	return n, mapErr(err)
}

// ShareLinkRepo implements repository.ShareLinkRepository.
type ShareLinkRepo struct{ DB *DB }

const shareCols = `id, user_id, es_project_id, token_hash, snapshot, show_company_name, expires_at, revoked_at, created_at`

func scanShare(row interface{ Scan(...any) error }) (entity.ShareLink, error) {
	var l entity.ShareLink
	var snap []byte
	if err := row.Scan(&l.ID, &l.UserID, &l.ESProjectID, &l.TokenHash, &snap, &l.ShowCompanyName, &l.ExpiresAt, &l.RevokedAt, &l.CreatedAt); err != nil {
		return l, mapErr(err)
	}
	if err := json.Unmarshal(snap, &l.Snapshot); err != nil {
		return l, fmt.Errorf("decode snapshot: %w", err)
	}
	return l, nil
}

// Create inserts a link (only the token hash).
func (r *ShareLinkRepo) Create(ctx context.Context, l entity.ShareLink) (entity.ShareLink, error) {
	snap, err := json.Marshal(l.Snapshot)
	if err != nil {
		return entity.ShareLink{}, err
	}
	return scanShare(r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO share_links (id, user_id, es_project_id, token_hash, snapshot, show_company_name, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+shareCols,
		l.ID, l.UserID, l.ESProjectID, l.TokenHash, snap, l.ShowCompanyName, l.ExpiresAt))
}

// ListByProject lists links of a project, newest first.
func (r *ShareLinkRepo) ListByProject(ctx context.Context, userID, projectID string) ([]entity.ShareLink, error) {
	rows, err := r.DB.q(ctx).Query(ctx,
		`SELECT `+shareCols+` FROM share_links WHERE user_id = $1 AND es_project_id = $2 ORDER BY created_at DESC, id DESC`, userID, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []entity.ShareLink{}
	for rows.Next() {
		l, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, mapErr(rows.Err())
}

// Revoke sets revoked_at; revoking an already-revoked link succeeds.
func (r *ShareLinkRepo) Revoke(ctx context.Context, userID, id string, now time.Time) error {
	return affected(r.DB.q(ctx).Exec(ctx,
		`UPDATE share_links SET revoked_at = COALESCE(revoked_at, $3) WHERE user_id = $1 AND id = $2`, userID, id, now))
}

// GetByTokenHash looks a link up by token hash (unauthenticated share page).
func (r *ShareLinkRepo) GetByTokenHash(ctx context.Context, tokenHash string) (entity.ShareLink, error) {
	return scanShare(r.DB.q(ctx).QueryRow(ctx, `SELECT `+shareCols+` FROM share_links WHERE token_hash = $1`, tokenHash))
}

// DashboardRepo implements repository.DashboardRepository.
type DashboardRepo struct{ DB *DB }

// CreateSummary inserts a dashboard summary.
func (r *DashboardRepo) CreateSummary(ctx context.Context, s entity.DashboardSummary) (entity.DashboardSummary, error) {
	err := r.DB.q(ctx).QueryRow(ctx,
		`INSERT INTO dashboard_summaries (id, user_id, content, source_interview_id) VALUES ($1, $2, $3, $4) RETURNING created_at`,
		s.ID, s.UserID, s.Content, s.SourceInterviewID).Scan(&s.CreatedAt)
	return s, mapErr(err)
}

// LatestSummary returns the newest summary or nil.
func (r *DashboardRepo) LatestSummary(ctx context.Context, userID string) (*entity.DashboardSummary, error) {
	var s entity.DashboardSummary
	err := r.DB.q(ctx).QueryRow(ctx,
		`SELECT id, user_id, content, source_interview_id, created_at FROM dashboard_summaries WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`,
		userID).Scan(&s.ID, &s.UserID, &s.Content, &s.SourceInterviewID, &s.CreatedAt)
	if err != nil {
		if mapErr(err) == repository.ErrNotFound {
			return nil, nil
		}
		return nil, mapErr(err)
	}
	return &s, nil
}

func statusOf(s string) vo.ESStatus      { return vo.ESStatus(s) }
func sourceOf(s string) vo.VersionSource { return vo.VersionSource(s) }
