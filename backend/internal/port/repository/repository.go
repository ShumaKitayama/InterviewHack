// Package repository defines persistence interfaces. Every method that reads or
// writes user data takes userID and must scope its query by it (D6, 6.4).
// DesignDocument/7_BackendDesign.md 7.2.2 / 7.3.1
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/vo"
)

// ErrNotFound is returned when a row does not exist or belongs to another user.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on unique-constraint violations.
var ErrConflict = errors.New("conflict")

// ErrInvalidCursor is returned for a malformed pagination cursor.
var ErrInvalidCursor = errors.New("invalid cursor")

// TxManager runs fn in one DB transaction. Repositories called with the ctx
// passed to fn take part in that transaction.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PageParams is cursor pagination input (7.5).
type PageParams struct {
	Limit  int
	Cursor string
}

// Page is one page of results.
type Page[T any] struct {
	Items      []T
	NextCursor *string
}

// UserRepository manages users.
type UserRepository interface {
	// GetOrCreateByFirebaseUID returns the user for uid, creating it on first access.
	// The email is synced only for users that are not deleted.
	GetOrCreateByFirebaseUID(ctx context.Context, uid string, email *string) (entity.User, error)
	Get(ctx context.Context, userID string) (entity.User, error)
	UpdateDisplayName(ctx context.Context, userID string, displayName *string) (entity.User, error)
	// DeleteAccount physically deletes all domain data of the user and marks the
	// users row deleted with email/display_name cleared (6.3.3).
	DeleteAccount(ctx context.Context, userID string, now time.Time) error
}

// AssetFilter filters assets.
type AssetFilter struct {
	Type   string
	TagID  string
	Query  string
	SortBy string // created | updated
}

// AssetRepository manages assets and their tag relations.
type AssetRepository interface {
	List(ctx context.Context, userID string, f AssetFilter, p PageParams) (Page[entity.Asset], error)
	Get(ctx context.Context, userID, id string) (entity.Asset, error)
	GetMany(ctx context.Context, userID string, ids []string) ([]entity.Asset, error)
	Create(ctx context.Context, a entity.Asset) (entity.Asset, error)
	Update(ctx context.Context, a entity.Asset) (entity.Asset, error)
	SoftDelete(ctx context.Context, userID, id string) error
	CountOwned(ctx context.Context, userID string, ids []string) (int, error)
}

// TagRepository manages asset tags.
type TagRepository interface {
	List(ctx context.Context, userID string) ([]entity.AssetTag, error)
	Create(ctx context.Context, t entity.AssetTag) (entity.AssetTag, error)
	Update(ctx context.Context, t entity.AssetTag) (entity.AssetTag, error)
	Delete(ctx context.Context, userID, id string) error
	CountOwned(ctx context.Context, userID string, ids []string) (int, error)
}

// CompanyFilter filters companies.
type CompanyFilter struct {
	Query          string
	SelectionPhase string
}

// CompanyRepository manages companies.
type CompanyRepository interface {
	List(ctx context.Context, userID string, f CompanyFilter, p PageParams) (Page[entity.Company], error)
	Get(ctx context.Context, userID, id string) (entity.Company, error)
	Create(ctx context.Context, c entity.Company) (entity.Company, error)
	Update(ctx context.Context, c entity.Company) (entity.Company, error)
	SoftDelete(ctx context.Context, userID, id string) error
}

// ResearchRepository manages research materials.
type ResearchRepository interface {
	List(ctx context.Context, userID, companyID string, p PageParams) (Page[entity.ResearchMaterial], error)
	Get(ctx context.Context, userID, id string) (entity.ResearchMaterial, error)
	GetMany(ctx context.Context, userID string, ids []string) ([]entity.ResearchMaterial, error)
	Create(ctx context.Context, m entity.ResearchMaterial) (entity.ResearchMaterial, error)
	Update(ctx context.Context, m entity.ResearchMaterial) (entity.ResearchMaterial, error)
	Delete(ctx context.Context, userID, id string) error
	CountOwned(ctx context.Context, userID string, ids []string) (int, error)
}

// ESFilter filters ES projects.
type ESFilter struct {
	CompanyID string
	Status    string
}

// ESRepository manages ES projects, sections and section versions.
type ESRepository interface {
	ListProjects(ctx context.Context, userID string, f ESFilter, p PageParams) (Page[entity.ESProject], error)
	GetProject(ctx context.Context, userID, id string) (entity.ESProject, error)
	CreateProject(ctx context.Context, e entity.ESProject) (entity.ESProject, error)
	UpdateProject(ctx context.Context, e entity.ESProject) (entity.ESProject, error)
	DeleteProject(ctx context.Context, userID, id string) error

	ListSections(ctx context.Context, userID, projectID string) ([]entity.ESSection, error)
	GetSection(ctx context.Context, userID, id string) (entity.ESSection, error)
	CreateSection(ctx context.Context, s entity.ESSection) (entity.ESSection, error)
	UpdateSection(ctx context.Context, s entity.ESSection) (entity.ESSection, error)
	DeleteSection(ctx context.Context, userID, id string) error

	ListVersions(ctx context.Context, userID, sectionID string) ([]entity.ESSectionVersion, error)
	GetVersion(ctx context.Context, userID, sectionID, versionID string) (entity.ESSectionVersion, error)
	CreateVersion(ctx context.Context, v entity.ESSectionVersion) (entity.ESSectionVersion, error)
	// PruneVersions keeps only the newest keep versions of a section.
	PruneVersions(ctx context.Context, userID, sectionID string, keep int) error
}

// InterviewFilter filters interviews.
type InterviewFilter struct {
	CompanyID string
	From      *time.Time
	To        *time.Time
}

// InterviewRepository manages interviews and interview logs.
type InterviewRepository interface {
	ListInterviews(ctx context.Context, userID string, f InterviewFilter, p PageParams) (Page[entity.Interview], error)
	GetInterview(ctx context.Context, userID, id string) (entity.Interview, error)
	CreateInterview(ctx context.Context, i entity.Interview) (entity.Interview, error)
	UpdateInterview(ctx context.Context, i entity.Interview) (entity.Interview, error)
	DeleteInterview(ctx context.Context, userID, id string) error
	UpcomingInterviews(ctx context.Context, userID string, from time.Time, limit int) ([]entity.Interview, error)

	ListLogs(ctx context.Context, userID, interviewID string, p PageParams) (Page[entity.InterviewLog], error)
	GetLog(ctx context.Context, userID, id string) (entity.InterviewLog, error)
	CreateLog(ctx context.Context, l entity.InterviewLog) (entity.InterviewLog, error)
	UpdateLog(ctx context.Context, l entity.InterviewLog) (entity.InterviewLog, error)
	DeleteLog(ctx context.Context, userID, id string) error
	// ReviewedLogs returns logs with ai_feedback_text, newest first, optionally of one interview.
	ReviewedLogs(ctx context.Context, userID string, interviewID *string, limit int) ([]entity.InterviewLog, error)
	RecentLogs(ctx context.Context, userID string, limit int) ([]entity.InterviewLog, error)
}

// QAFilter filters Q&A.
type QAFilter struct {
	CompanyID    string
	Query        string
	IsFavorite   *bool
	ReviewStatus string
}

// QARepository manages the Q&A bank.
type QARepository interface {
	List(ctx context.Context, userID string, f QAFilter, p PageParams) (Page[entity.QA], error)
	Get(ctx context.Context, userID, id string) (entity.QA, error)
	Create(ctx context.Context, q entity.QA) (entity.QA, error)
	Update(ctx context.Context, q entity.QA) (entity.QA, error)
	Delete(ctx context.Context, userID, id string) error
	Recent(ctx context.Context, userID string, limit int) ([]entity.QA, error)
}

// TransitionParams describes a conditional status update (7.6.1).
type TransitionParams struct {
	From         []vo.SessionStatus
	To           vo.SessionStatus
	ParseResult  *vo.ParseResult
	SetPastedAt  bool
	SetCompleted bool
	Now          time.Time
}

// PromptSessionRepository manages prompt_sessions.
type PromptSessionRepository interface {
	Create(ctx context.Context, s entity.PromptSession) (entity.PromptSession, error)
	Get(ctx context.Context, userID, id string) (entity.PromptSession, error)
	List(ctx context.Context, userID string, statuses []vo.SessionStatus, p PageParams) (Page[entity.PromptSession], error)
	// Transition updates the status only when the current status is in From.
	// It returns false when no row matched.
	Transition(ctx context.Context, userID, id string, t TransitionParams) (bool, error)
	// ExpireUnfinished marks created/pasted sessions created at or before cutoff as abandoned.
	ExpireUnfinished(ctx context.Context, userID string, cutoff, now time.Time) error
	CountUnfinished(ctx context.Context, userID string) (int, error)
}

// ShareLinkRepository manages share_links.
type ShareLinkRepository interface {
	Create(ctx context.Context, l entity.ShareLink) (entity.ShareLink, error)
	ListByProject(ctx context.Context, userID, projectID string) ([]entity.ShareLink, error)
	Revoke(ctx context.Context, userID, id string, now time.Time) error
	// GetByTokenHash is the only unauthenticated lookup (7.3.1).
	GetByTokenHash(ctx context.Context, tokenHash string) (entity.ShareLink, error)
}

// DashboardRepository manages dashboard_summaries.
type DashboardRepository interface {
	CreateSummary(ctx context.Context, s entity.DashboardSummary) (entity.DashboardSummary, error)
	LatestSummary(ctx context.Context, userID string) (*entity.DashboardSummary, error)
}
