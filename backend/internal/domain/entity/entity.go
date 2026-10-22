// DesignDocument/2_DomainDesign.md 2.2 エンティティ定義
package entity

import (
	"time"

	"github.com/interviewhack/backend/internal/domain/vo"
)

// User is the application user, keyed by Firebase uid (2.2.1).
type User struct {
	ID          string
	FirebaseUID string
	Email       *string
	DisplayName *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Asset is a self asset (セット1, 2.2.2).
type Asset struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Content   string
	TagIDs    []string
	TagNames  []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AssetTag is a per-user tag name.
type AssetTag struct {
	ID     string
	UserID string
	Name   string
}

// Company is a target company (セット2, 2.2.3).
type Company struct {
	ID             string
	UserID         string
	Name           string
	JobTitle       *string
	SelectionPhase *string
	MotivationMemo *string
	ReferenceURL   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ResearchMaterial holds text extracted on device (2.2.4, D11).
type ResearchMaterial struct {
	ID         string
	UserID     string
	CompanyID  string
	Title      string
	Type       string
	SourceURL  *string
	ContentRaw *string
	Summary    *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ESProject is an ES project per company (2.2.5).
type ESProject struct {
	ID                  string
	UserID              string
	CompanyID           string
	Status              vo.ESStatus
	Deadline            *time.Time
	AssetIDs            []string
	ResearchMaterialIDs []string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ESSection is one ES question (2.2.6).
type ESSection struct {
	ID             string
	ESProjectID    string
	UserID         string
	Title          string
	ContentDraft   *string
	ContentFinal   *string
	CharacterLimit *int
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ESSectionVersion is a saved past content_draft (2.2.6.1).
type ESSectionVersion struct {
	ID          string
	ESSectionID string
	UserID      string
	Version     int
	Content     string
	Source      vo.VersionSource
	CreatedAt   time.Time
}

// Interview is one interview event (2.2.7).
type Interview struct {
	ID            string
	UserID        string
	CompanyID     *string
	InterviewDate time.Time
	Phase         *string
	Result        *string
	Memo          *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// InterviewLog is one question/answer pair from an interview (2.2.8).
type InterviewLog struct {
	ID                   string
	InterviewID          string
	UserID               string
	QuestionText         string
	AnswerText           string
	SelfEvaluation       *string
	AIFeedbackText       *string
	AIImprovedAnswerText *string
	Score                *int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// QA is a Q&A bank entry (2.2.9).
type QA struct {
	ID           string
	UserID       string
	CompanyID    *string
	QuestionText string
	AnswerText   string
	SourceType   vo.QASourceType
	IsFavorite   bool
	ReviewStatus vo.ReviewStatus
	AssetIDs     []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// InputRefs is what prompt_sessions.input_refs stores: IDs and options only (D13).
type InputRefs struct {
	AssetIDs            []string       `json:"assetIds,omitempty"`
	ResearchMaterialIDs []string       `json:"researchMaterialIds,omitempty"`
	Options             map[string]any `json:"options,omitempty"`
}

// PromptSession is one AI連携 round trip (2.2.10).
type PromptSession struct {
	ID              string
	UserID          string
	UseCase         vo.UseCase
	TemplateVersion string
	CompanyID       *string
	ESProjectID     *string
	ESSectionID     *string
	InterviewID     *string
	InterviewLogID  *string
	InputRefs       InputRefs
	Status          vo.SessionStatus
	ParseResult     *vo.ParseResult
	CreatedAt       time.Time
	PastedAt        *time.Time
	CompletedAt     *time.Time
	// TargetLabel is a read-only display name of the target (list views only).
	TargetLabel string
}

// ShareSnapshotSection is one section frozen into a share link.
type ShareSnapshotSection struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ShareSnapshot is share_links.snapshot (D12). It never contains the user name.
type ShareSnapshot struct {
	CompanyName *string                `json:"companyName,omitempty"`
	Sections    []ShareSnapshotSection `json:"sections"`
}

// ShareLink is an ES share link (2.2.11).
type ShareLink struct {
	ID              string
	UserID          string
	ESProjectID     string
	TokenHash       string
	Snapshot        ShareSnapshot
	ShowCompanyName bool
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	CreatedAt       time.Time
}

// DashboardSummary is a summary_generate result (2.2.12).
type DashboardSummary struct {
	ID                string
	UserID            string
	Content           string
	SourceInterviewID *string
	CreatedAt         time.Time
}
