package dto

import (
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	promptsvc "github.com/interviewhack/backend/internal/domain/service/prompt"
	"github.com/interviewhack/backend/internal/domain/vo"
)

// CreateSessionRequest is POST /prompt-sessions.
type CreateSessionRequest struct {
	UseCase             string   `json:"useCase"`
	CompanyID           *string  `json:"companyId"`
	InterviewID         *string  `json:"interviewId"`
	InterviewLogID      *string  `json:"interviewLogId"`
	ESProjectID         *string  `json:"esProjectId"`
	ESSectionID         *string  `json:"esSectionId"`
	AssetIDs            []string `json:"assetIds"`
	ResearchMaterialIDs []string `json:"researchMaterialIds"`
	Options             struct {
		QuestionCount int    `json:"questionCount"`
		Strictness    string `json:"strictness"`
		Instructions  string `json:"instructions"`
	} `json:"options"`
}

// PromptResponse is the response of POST /prompt-sessions and GET .../prompt.
type PromptResponse struct {
	SessionID       string   `json:"sessionId"`
	TemplateVersion string   `json:"templateVersion"`
	PromptText      string   `json:"promptText"`
	Warnings        []string `json:"warnings"`
}

// Session is a prompt session summary.
type Session struct {
	SessionID       string     `json:"sessionId"`
	UseCase         string     `json:"useCase"`
	TemplateVersion string     `json:"templateVersion"`
	Status          string     `json:"status"`
	ParseResult     *string    `json:"parseResult"`
	CompanyID       *string    `json:"companyId"`
	ESProjectID     *string    `json:"esProjectId"`
	ESSectionID     *string    `json:"esSectionId"`
	InterviewID     *string    `json:"interviewId"`
	InterviewLogID  *string    `json:"interviewLogId"`
	TargetLabel     string     `json:"targetLabel"`
	CreatedAt       time.Time  `json:"createdAt"`
	PastedAt        *time.Time `json:"pastedAt"`
	CompletedAt     *time.Time `json:"completedAt"`
}

// SessionFrom converts a session (input_refs are not exposed).
func SessionFrom(s entity.PromptSession) Session {
	var pr *string
	if s.ParseResult != nil {
		v := string(*s.ParseResult)
		pr = &v
	}
	return Session{SessionID: s.ID, UseCase: string(s.UseCase), TemplateVersion: s.TemplateVersion, Status: string(s.Status), ParseResult: pr,
		CompanyID: s.CompanyID, ESProjectID: s.ESProjectID, ESSectionID: s.ESSectionID, InterviewID: s.InterviewID, InterviewLogID: s.InterviewLogID,
		TargetLabel: s.TargetLabel, CreatedAt: s.CreatedAt, PastedAt: s.PastedAt, CompletedAt: s.CompletedAt}
}

// ParseRequest is POST /prompt-sessions/{id}/parse.
type ParseRequest struct {
	ResponseText *string `json:"responseText"`
}

// QAPreviewItem is one parsed Q&A. intentText is preview-only.
type QAPreviewItem struct {
	QuestionText string `json:"questionText"`
	AnswerText   string `json:"answerText"`
	IntentText   string `json:"intentText"`
}

// ParseResponse is the preview (7.5.5 の表).
type ParseResponse struct {
	SessionID               string         `json:"sessionId"`
	ParseResult             string         `json:"parseResult"`
	DetectedTemplateVersion string         `json:"detectedTemplateVersion"`
	Preview                 map[string]any `json:"preview"`
	Warnings                []string       `json:"warnings"`
}

// PreviewFor renders only the fields of the use case.
func PreviewFor(uc vo.UseCase, p promptsvc.Preview) map[string]any {
	m := map[string]any{}
	switch uc {
	case vo.UseCaseQAGenerate:
		items := make([]QAPreviewItem, len(p.Items))
		for i, it := range p.Items {
			items[i] = QAPreviewItem{QuestionText: it.QuestionText, AnswerText: it.AnswerText, IntentText: it.IntentText}
		}
		m["items"] = items
	case vo.UseCaseAnswerReview:
		m["issues"] = nn(p.Issues)
		m["improvements"] = nn(p.Improvements)
		m["improvedAnswerText"] = p.ImprovedAnswerText
	case vo.UseCaseResearchSummarize:
		m["highlights"] = nn(p.Highlights)
		m["idealCandidate"] = p.IdealCandidate
		m["recentTopics"] = nn(p.RecentTopics)
		m["summaryText"] = p.Text
	default:
		m["text"] = p.Text
	}
	if p.RawText != "" {
		m["rawText"] = p.RawText
	}
	return m
}

// CommitRequest is POST /prompt-sessions/{id}/commit. Unknown fields are rejected
// so a mismatched shape is detected (UT-VAL-006).
type CommitRequest struct {
	Items              *[]CommitQAItem `json:"items"`
	AssetIDs           *[]string       `json:"assetIds"`
	FeedbackText       *string         `json:"feedbackText"`
	ImprovedAnswerText *string         `json:"improvedAnswerText"`
	SaveToQA           *bool           `json:"saveToQa"`
	SummaryText        *string         `json:"summaryText"`
	Text               *string         `json:"text"`
}

// CommitQAItem is one Q&A to save.
type CommitQAItem struct {
	QuestionText string `json:"questionText"`
	AnswerText   string `json:"answerText"`
}

// CommitResponse is the commit result.
type CommitResponse struct {
	SessionID string              `json:"sessionId"`
	Status    string              `json:"status"`
	SavedRefs map[string][]string `json:"savedRefs"`
}

// CreateShareRequest is POST /es-projects/{id}/share-links.
type CreateShareRequest struct {
	SectionIDs      []string `json:"sectionIds"`
	ExpiresInDays   *int     `json:"expiresInDays"`
	ShowCompanyName *bool    `json:"showCompanyName"`
}

// ShareCreated is the creation response; url carries the raw token only here.
type ShareCreated struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ShareLink is a list item (no URL / token).
type ShareLink struct {
	ID              string     `json:"id"`
	SectionTitles   []string   `json:"sectionTitles"`
	ShowCompanyName bool       `json:"showCompanyName"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	RevokedAt       *time.Time `json:"revokedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	IsActive        bool       `json:"isActive"`
}

// ShareLinkFrom converts a link.
func ShareLinkFrom(l entity.ShareLink, active bool) ShareLink {
	titles := make([]string, len(l.Snapshot.Sections))
	for i, s := range l.Snapshot.Sections {
		titles[i] = s.Title
	}
	return ShareLink{ID: l.ID, SectionTitles: titles, ShowCompanyName: l.ShowCompanyName, ExpiresAt: l.ExpiresAt, RevokedAt: l.RevokedAt, CreatedAt: l.CreatedAt, IsActive: active}
}

// Dashboard is GET /dashboard.
type Dashboard struct {
	UpcomingInterviews    []Interview       `json:"upcomingInterviews"`
	LatestSummary         *DashboardSummary `json:"latestSummary"`
	RecentQA              []QA              `json:"recentQa"`
	RecentInterviewLogs   []InterviewLog    `json:"recentInterviewLogs"`
	PendingPromptSessions PendingSessions   `json:"pendingPromptSessions"`
}

// DashboardSummary is the latest summary.
type DashboardSummary struct {
	ID                string    `json:"id"`
	Content           string    `json:"content"`
	SourceInterviewID *string   `json:"sourceInterviewId"`
	CreatedAt         time.Time `json:"createdAt"`
}

// PendingSessions is the count and the newest unfinished sessions.
type PendingSessions struct {
	Count int       `json:"count"`
	Items []Session `json:"items"`
}
