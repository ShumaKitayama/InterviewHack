package dto

import (
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
)

// List is the common list response.
type List[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// Me is GET /me.
type Me struct {
	ID          string    `json:"id"`
	Email       *string   `json:"email"`
	DisplayName *string   `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// MeFrom converts a user.
func MeFrom(u entity.User) Me {
	return Me{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, CreatedAt: u.CreatedAt}
}

// Asset is an asset response.
type Asset struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	TagIDs    []string  `json:"tagIds"`
	TagNames  []string  `json:"tagNames"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AssetFrom converts an asset.
func AssetFrom(a entity.Asset) Asset {
	return Asset{ID: a.ID, Type: a.Type, Title: a.Title, Content: a.Content, TagIDs: nn(a.TagIDs), TagNames: nn(a.TagNames), CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}

// AssetInput is POST/PATCH /assets.
type AssetInput struct {
	Type    Opt[string]   `json:"type"`
	Title   Opt[string]   `json:"title"`
	Content Opt[string]   `json:"content"`
	TagIDs  Opt[[]string] `json:"tagIds"`
}

// Tag is a tag response.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TagInput is POST/PATCH /asset-tags.
type TagInput struct {
	Name string `json:"name"`
}

// Company is a company response.
type Company struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	JobTitle       *string   `json:"jobTitle"`
	SelectionPhase *string   `json:"selectionPhase"`
	MotivationMemo *string   `json:"motivationMemo"`
	ReferenceURL   *string   `json:"referenceUrl"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// CompanyFrom converts a company.
func CompanyFrom(c entity.Company) Company {
	return Company{ID: c.ID, Name: c.Name, JobTitle: c.JobTitle, SelectionPhase: c.SelectionPhase, MotivationMemo: c.MotivationMemo, ReferenceURL: c.ReferenceURL, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

// CompanyInput is POST/PATCH /companies.
type CompanyInput struct {
	Name           Opt[string] `json:"name"`
	JobTitle       Opt[string] `json:"jobTitle"`
	SelectionPhase Opt[string] `json:"selectionPhase"`
	MotivationMemo Opt[string] `json:"motivationMemo"`
	ReferenceURL   Opt[string] `json:"referenceUrl"`
}

// ResearchMaterial is a research material response.
type ResearchMaterial struct {
	ID         string    `json:"id"`
	CompanyID  string    `json:"companyId"`
	Title      string    `json:"title"`
	Type       string    `json:"type"`
	SourceURL  *string   `json:"sourceUrl"`
	ContentRaw *string   `json:"contentRaw"`
	Summary    *string   `json:"summary"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// ResearchFrom converts a material.
func ResearchFrom(m entity.ResearchMaterial) ResearchMaterial {
	return ResearchMaterial{ID: m.ID, CompanyID: m.CompanyID, Title: m.Title, Type: m.Type, SourceURL: m.SourceURL, ContentRaw: m.ContentRaw, Summary: m.Summary, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}

// ResearchInput is POST/PATCH /research-materials.
type ResearchInput struct {
	CompanyID  Opt[string] `json:"companyId"`
	Title      Opt[string] `json:"title"`
	Type       Opt[string] `json:"type"`
	SourceURL  Opt[string] `json:"sourceUrl"`
	ContentRaw Opt[string] `json:"contentRaw"`
	Summary    Opt[string] `json:"summary"`
}

// ESProject is an ES project response.
type ESProject struct {
	ID                  string    `json:"id"`
	CompanyID           string    `json:"companyId"`
	Status              string    `json:"status"`
	Deadline            *string   `json:"deadline"`
	AssetIDs            []string  `json:"assetIds"`
	ResearchMaterialIDs []string  `json:"researchMaterialIds"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// ESProjectFrom converts a project.
func ESProjectFrom(e entity.ESProject) ESProject {
	return ESProject{ID: e.ID, CompanyID: e.CompanyID, Status: string(e.Status), Deadline: FormatDatePtr(e.Deadline),
		AssetIDs: nn(e.AssetIDs), ResearchMaterialIDs: nn(e.ResearchMaterialIDs), CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt}
}

// ESProjectInput is POST/PATCH /es-projects.
type ESProjectInput struct {
	CompanyID           Opt[string]   `json:"companyId"`
	Status              Opt[string]   `json:"status"`
	Deadline            Opt[string]   `json:"deadline"`
	AssetIDs            Opt[[]string] `json:"assetIds"`
	ResearchMaterialIDs Opt[[]string] `json:"researchMaterialIds"`
}

// ESSection is an ES section response.
type ESSection struct {
	ID             string    `json:"id"`
	ESProjectID    string    `json:"esProjectId"`
	Title          string    `json:"title"`
	ContentDraft   *string   `json:"contentDraft"`
	ContentFinal   *string   `json:"contentFinal"`
	CharacterLimit *int      `json:"characterLimit"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ESSectionFrom converts a section.
func ESSectionFrom(s entity.ESSection) ESSection {
	return ESSection{ID: s.ID, ESProjectID: s.ESProjectID, Title: s.Title, ContentDraft: s.ContentDraft, ContentFinal: s.ContentFinal,
		CharacterLimit: s.CharacterLimit, Version: s.Version, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}
}

// ESSectionInput is POST/PATCH sections.
type ESSectionInput struct {
	Title          Opt[string] `json:"title"`
	ContentDraft   Opt[string] `json:"contentDraft"`
	ContentFinal   Opt[string] `json:"contentFinal"`
	CharacterLimit Opt[int]    `json:"characterLimit"`
}

// ESSectionVersion is a version response.
type ESSectionVersion struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// ESSectionVersionFrom converts a version.
func ESSectionVersionFrom(v entity.ESSectionVersion) ESSectionVersion {
	return ESSectionVersion{ID: v.ID, Version: v.Version, Source: string(v.Source), Content: v.Content, CreatedAt: v.CreatedAt}
}

// Interview is an interview response.
type Interview struct {
	ID            string    `json:"id"`
	CompanyID     *string   `json:"companyId"`
	InterviewDate string    `json:"interviewDate"`
	Phase         *string   `json:"phase"`
	Result        *string   `json:"result"`
	Memo          *string   `json:"memo"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// InterviewFrom converts an interview.
func InterviewFrom(i entity.Interview) Interview {
	return Interview{ID: i.ID, CompanyID: i.CompanyID, InterviewDate: FormatDate(i.InterviewDate), Phase: i.Phase, Result: i.Result, Memo: i.Memo, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

// InterviewInput is POST/PATCH /interviews.
type InterviewInput struct {
	CompanyID     Opt[string] `json:"companyId"`
	InterviewDate Opt[string] `json:"interviewDate"`
	Phase         Opt[string] `json:"phase"`
	Result        Opt[string] `json:"result"`
	Memo          Opt[string] `json:"memo"`
}

// InterviewLog is an interview log response.
type InterviewLog struct {
	ID                   string    `json:"id"`
	InterviewID          string    `json:"interviewId"`
	QuestionText         string    `json:"questionText"`
	AnswerText           string    `json:"answerText"`
	SelfEvaluation       *string   `json:"selfEvaluation"`
	AIFeedbackText       *string   `json:"aiFeedbackText"`
	AIImprovedAnswerText *string   `json:"aiImprovedAnswerText"`
	Score                *int      `json:"score"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// InterviewLogFrom converts a log.
func InterviewLogFrom(l entity.InterviewLog) InterviewLog {
	return InterviewLog{ID: l.ID, InterviewID: l.InterviewID, QuestionText: l.QuestionText, AnswerText: l.AnswerText, SelfEvaluation: l.SelfEvaluation,
		AIFeedbackText: l.AIFeedbackText, AIImprovedAnswerText: l.AIImprovedAnswerText, Score: l.Score, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}

// InterviewLogInput is POST/PATCH /interview-logs.
type InterviewLogInput struct {
	InterviewID          Opt[string] `json:"interviewId"`
	QuestionText         Opt[string] `json:"questionText"`
	AnswerText           Opt[string] `json:"answerText"`
	SelfEvaluation       Opt[string] `json:"selfEvaluation"`
	AIFeedbackText       Opt[string] `json:"aiFeedbackText"`
	AIImprovedAnswerText Opt[string] `json:"aiImprovedAnswerText"`
	Score                Opt[int]    `json:"score"`
}

// QA is a Q&A response.
type QA struct {
	ID           string    `json:"id"`
	CompanyID    *string   `json:"companyId"`
	QuestionText string    `json:"questionText"`
	AnswerText   string    `json:"answerText"`
	SourceType   string    `json:"sourceType"`
	IsFavorite   bool      `json:"isFavorite"`
	ReviewStatus string    `json:"reviewStatus"`
	AssetIDs     []string  `json:"assetIds"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// QAFrom converts a Q&A.
func QAFrom(q entity.QA) QA {
	return QA{ID: q.ID, CompanyID: q.CompanyID, QuestionText: q.QuestionText, AnswerText: q.AnswerText, SourceType: string(q.SourceType),
		IsFavorite: q.IsFavorite, ReviewStatus: string(q.ReviewStatus), AssetIDs: nn(q.AssetIDs), CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt}
}

// QAInput is POST/PATCH /qa.
type QAInput struct {
	CompanyID    Opt[string]   `json:"companyId"`
	QuestionText Opt[string]   `json:"questionText"`
	AnswerText   Opt[string]   `json:"answerText"`
	SourceType   Opt[string]   `json:"sourceType"`
	IsFavorite   Opt[bool]     `json:"isFavorite"`
	ReviewStatus Opt[string]   `json:"reviewStatus"`
	AssetIDs     Opt[[]string] `json:"assetIds"`
}

// Map converts a slice.
func Map[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func nn(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
