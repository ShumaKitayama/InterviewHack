package handler

import (
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	promptsvc "github.com/interviewhack/backend/internal/domain/service/prompt"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/usecase/prompt"
)

// CreatePromptSession is POST /prompt-sessions (7.5.5).
func (h *Handler) CreatePromptSession(c *gin.Context) {
	var in dto.CreateSessionRequest
	if !bind(c, &in) {
		return
	}
	for _, f := range []struct {
		n string
		v *string
	}{{"companyId", in.CompanyID}, {"interviewId", in.InterviewID}, {"interviewLogId", in.InterviewLogID}, {"esProjectId", in.ESProjectID}, {"esSectionId", in.ESSectionID}} {
		if !uuidField(c, f.n, f.v) {
			return
		}
	}
	if !uuidList(c, "assetIds", in.AssetIDs) || !uuidList(c, "researchMaterialIds", in.ResearchMaterialIDs) {
		return
	}
	res, err := h.Prompts.Create(c.Request.Context(), userID(c), prompt.CreateInput{
		UseCase: in.UseCase, CompanyID: in.CompanyID, InterviewID: in.InterviewID, InterviewLogID: in.InterviewLogID,
		ESProjectID: in.ESProjectID, ESSectionID: in.ESSectionID, AssetIDs: in.AssetIDs, ResearchMaterialIDs: in.ResearchMaterialIDs,
		Options: promptsvc.Options{QuestionCount: in.Options.QuestionCount, Strictness: in.Options.Strictness, Instructions: in.Options.Instructions},
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	c.Set("session_id", res.Session.ID)
	c.Set("use_case", string(res.Session.UseCase))
	c.Set("template_version", res.Session.TemplateVersion)
	presenter.RespondCreated(c, dto.PromptResponse{SessionID: res.Session.ID, TemplateVersion: res.Prompt.TemplateVersion,
		PromptText: res.Prompt.Text, Warnings: nonNilStrings(res.Prompt.Warnings)})
}

// ListPromptSessions is GET /prompt-sessions?status=created,pasted.
func (h *Handler) ListPromptSessions(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	page, err := h.Prompts.List(c.Request.Context(), userID(c), splitCSV(c.Query("status")), p)
	respondList(c, page, err, dto.SessionFrom)
}

// GetPromptSession is GET /prompt-sessions/:id (used after a commit 409, 7.6.1).
func (h *Handler) GetPromptSession(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	s, err := h.Prompts.Get(c.Request.Context(), userID(c), id)
	respond(c, s, err, dto.SessionFrom, false)
}

// GetPrompt is GET /prompt-sessions/:id/prompt.
func (h *Handler) GetPrompt(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	s, p, err := h.Prompts.Prompt(c.Request.Context(), userID(c), id)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondOK(c, dto.PromptResponse{SessionID: s.ID, TemplateVersion: p.TemplateVersion, PromptText: p.Text, Warnings: nonNilStrings(p.Warnings)})
}

// ParseResponse is POST /prompt-sessions/:id/parse. Nothing but the session
// status is stored; the response text is not logged.
func (h *Handler) ParseResponse(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.ParseRequest
	if !bind(c, &in) {
		return
	}
	if in.ResponseText == nil {
		presenter.RespondError(c, domainerror.ErrValidation("responseText は必須です"))
		return
	}
	c.Set("session_id", id)
	c.Set("response_length", utf8.RuneCountInString(*in.ResponseText))
	s, out, err := h.Prompts.Parse(c.Request.Context(), userID(c), id, *in.ResponseText)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	c.Set("use_case", string(s.UseCase))
	c.Set("template_version", s.TemplateVersion)
	c.Set("detected_template_version", out.DetectedVersion)
	c.Set("parse_result", string(out.Result))
	presenter.RespondOK(c, dto.ParseResponse{SessionID: s.ID, ParseResult: string(out.Result), DetectedTemplateVersion: out.DetectedVersion,
		Preview: dto.PreviewFor(s.UseCase, out.Preview), Warnings: nonNilStrings(out.Warnings)})
}

// CommitSession is POST /prompt-sessions/:id/commit.
func (h *Handler) CommitSession(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.CommitRequest
	if !bindStrict(c, &in) {
		return
	}
	if in.AssetIDs != nil && !uuidList(c, "assetIds", *in.AssetIDs) {
		return
	}
	ci := prompt.CommitInput{AssetIDs: in.AssetIDs, FeedbackText: in.FeedbackText, ImprovedAnswerText: in.ImprovedAnswerText,
		SummaryText: in.SummaryText, Text: in.Text, SaveToQA: in.SaveToQA != nil && *in.SaveToQA}
	if in.Items != nil {
		items := make([]prompt.QAItem, len(*in.Items))
		for i, it := range *in.Items {
			items[i] = prompt.QAItem{QuestionText: it.QuestionText, AnswerText: it.AnswerText}
		}
		ci.QAItems = &items
	}
	c.Set("session_id", id)
	res, err := h.Prompts.Commit(c.Request.Context(), userID(c), id, ci)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondOK(c, dto.CommitResponse{SessionID: res.SessionID, Status: string(vo.SessionSaved), SavedRefs: res.SavedRefs})
}

// AbandonSession is POST /prompt-sessions/:id/abandon.
func (h *Handler) AbandonSession(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.Prompts.Abandon(c.Request.Context(), userID(c), id); err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondOK(c, gin.H{"sessionId": id, "status": string(vo.SessionAbandoned)})
}

// GetDashboard is GET /dashboard (7.5.4).
func (h *Handler) GetDashboard(c *gin.Context) {
	v, err := h.Dashboard.Get(c.Request.Context(), userID(c))
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	out := dto.Dashboard{
		UpcomingInterviews:    dto.Map(v.UpcomingInterviews, dto.InterviewFrom),
		RecentQA:              dto.Map(v.RecentQA, dto.QAFrom),
		RecentInterviewLogs:   dto.Map(v.RecentInterviewLogs, dto.InterviewLogFrom),
		PendingPromptSessions: dto.PendingSessions{Count: v.PendingCount, Items: dto.Map(v.PendingSessions, dto.SessionFrom)},
	}
	if s := v.LatestSummary; s != nil {
		out.LatestSummary = &dto.DashboardSummary{ID: s.ID, Content: s.Content, SourceInterviewID: s.SourceInterviewID, CreatedAt: s.CreatedAt}
	}
	presenter.RespondOK(c, out)
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
