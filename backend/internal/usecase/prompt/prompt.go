// Package prompt implements the AI連携 flow: create → parse → commit / abandon.
// DesignDocument/7_BackendDesign.md 7.5.5 / 7.6.1 / 7.9, 4_LLMDesign.md
package prompt

import (
	"context"
	"errors"
	"strings"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	promptsvc "github.com/interviewhack/backend/internal/domain/service/prompt"
	"github.com/interviewhack/backend/internal/domain/service/session"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
	esuc "github.com/interviewhack/backend/internal/usecase/es"
)

// Service is the AI連携 use case.
type Service struct {
	Sessions   repository.PromptSessionRepository
	Companies  repository.CompanyRepository
	Assets     repository.AssetRepository
	Materials  repository.ResearchRepository
	ES         repository.ESRepository
	Interviews repository.InterviewRepository
	QA         repository.QARepository
	Dashboard  repository.DashboardRepository
	ESService  *esuc.Service
	Tx         repository.TxManager
	IDs        external.IDGenerator
	Clock      external.Clock
	// DisabledCommit lists use cases whose commit is stopped by a feature flag (12.11).
	DisabledCommit map[vo.UseCase]bool
}

// CreateInput is POST /prompt-sessions.
type CreateInput struct {
	UseCase             string
	CompanyID           *string
	InterviewID         *string
	InterviewLogID      *string
	ESProjectID         *string
	ESSectionID         *string
	AssetIDs            []string
	ResearchMaterialIDs []string
	Options             promptsvc.Options
}

// Created is the result of Create.
type Created struct {
	Session entity.PromptSession
	Prompt  promptsvc.Prompt
}

// CommitInput is the user-confirmed content. Only the part of the session's use case is used.
type CommitInput struct {
	QAItems            *[]QAItem
	AssetIDs           *[]string
	FeedbackText       *string
	ImprovedAnswerText *string
	SaveToQA           bool
	SummaryText        *string
	Text               *string
}

// QAItem is one Q&A to save. The intent is not accepted (not saved, 4.2.1).
type QAItem struct {
	QuestionText string
	AnswerText   string
}

// Committed is the result of Commit.
type Committed struct {
	SessionID string
	SavedRefs map[string][]string
}

func optionsMap(o promptsvc.Options) map[string]any {
	m := map[string]any{}
	if o.QuestionCount != 0 {
		m["questionCount"] = o.QuestionCount
	}
	if o.Strictness != "" {
		m["strictness"] = o.Strictness
	}
	if o.Instructions != "" {
		m["instructions"] = o.Instructions
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func optionsFrom(m map[string]any) promptsvc.Options {
	var o promptsvc.Options
	if v, ok := m["questionCount"].(float64); ok {
		o.QuestionCount = int(v)
	}
	if v, ok := m["questionCount"].(int); ok {
		o.QuestionCount = v
	}
	if v, ok := m["strictness"].(string); ok {
		o.Strictness = v
	}
	if v, ok := m["instructions"].(string); ok {
		o.Instructions = v
	}
	return o
}

func validateOptions(o promptsvc.Options) error {
	if o.Strictness != "" && o.Strictness != "normal" && o.Strictness != "hard" {
		return domainerror.ErrUnprocessable("strictness は normal / hard のいずれかです")
	}
	return common.MaxLen("instructions", o.Instructions, 2000)
}

func required(field string, v *string) error {
	if v == nil || *v == "" {
		return domainerror.ErrValidation(field + " は必須です")
	}
	return nil
}

// loadInput gathers the user's own data for a template. Anything not owned by
// the user is NOT_FOUND, so another user's data never enters a prompt (IT-AZ-006).
func (s *Service) loadInput(ctx context.Context, userID string, sess entity.PromptSession) (promptsvc.BuildInput, error) {
	in := promptsvc.BuildInput{UseCase: sess.UseCase, Options: optionsFrom(sess.InputRefs.Options)}
	if sess.CompanyID != nil {
		c, err := s.Companies.Get(ctx, userID, *sess.CompanyID)
		if err != nil {
			return in, common.MapRepoErr(err, "企業")
		}
		in.Company = &c
	}
	if ids := sess.InputRefs.AssetIDs; len(ids) > 0 {
		assets, err := s.Assets.GetMany(ctx, userID, ids)
		if err != nil {
			return in, err
		}
		if err := common.EnsureOwned(len(assets), ids, "自己資産"); err != nil {
			return in, err
		}
		in.Assets = assets
	}
	if ids := sess.InputRefs.ResearchMaterialIDs; len(ids) > 0 {
		ms, err := s.Materials.GetMany(ctx, userID, ids)
		if err != nil {
			return in, err
		}
		if err := common.EnsureOwned(len(ms), ids, "企業研究資料"); err != nil {
			return in, err
		}
		in.Materials = ms
	}
	if sess.InterviewLogID != nil {
		l, err := s.Interviews.GetLog(ctx, userID, *sess.InterviewLogID)
		if err != nil {
			return in, common.MapRepoErr(err, "面接ログ")
		}
		in.InterviewLog = &l
	}
	if sess.InterviewID != nil {
		i, err := s.Interviews.GetInterview(ctx, userID, *sess.InterviewID)
		if err != nil {
			return in, common.MapRepoErr(err, "面接")
		}
		in.Interview = &i
	}
	if sess.UseCase == vo.UseCaseSummaryGenerate {
		logs, err := s.Interviews.ReviewedLogs(ctx, userID, sess.InterviewID, promptsvc.RecentLogsForSummary)
		if err != nil {
			return in, err
		}
		in.FeedbackLogs = logs
	}
	if sess.ESSectionID != nil {
		sec, err := s.ES.GetSection(ctx, userID, *sess.ESSectionID)
		if err != nil {
			return in, common.MapRepoErr(err, "ESセクション")
		}
		if sess.ESProjectID != nil && sec.ESProjectID != *sess.ESProjectID {
			return in, domainerror.ErrNotFound("ESセクション")
		}
		in.Section = &sec
	}
	return in, nil
}

// resolveTargets fills derived targets (e.g. company of an ES project or interview).
func (s *Service) resolveTargets(ctx context.Context, userID string, sess *entity.PromptSession) error {
	if sess.InterviewLogID != nil && sess.InterviewID == nil {
		l, err := s.Interviews.GetLog(ctx, userID, *sess.InterviewLogID)
		if err != nil {
			return common.MapRepoErr(err, "面接ログ")
		}
		sess.InterviewID = &l.InterviewID
	}
	if sess.InterviewID != nil && sess.CompanyID == nil && sess.UseCase == vo.UseCaseAnswerReview {
		i, err := s.Interviews.GetInterview(ctx, userID, *sess.InterviewID)
		if err != nil {
			return common.MapRepoErr(err, "面接")
		}
		sess.CompanyID = i.CompanyID
	}
	if sess.ESProjectID != nil && sess.CompanyID == nil {
		p, err := s.ES.GetProject(ctx, userID, *sess.ESProjectID)
		if err != nil {
			return common.MapRepoErr(err, "ESプロジェクト")
		}
		sess.CompanyID = &p.CompanyID
	}
	return nil
}

// Create creates a session (status=created) and returns the generated prompt.
func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (Created, error) {
	uc := vo.UseCase(in.UseCase)
	if !uc.Valid() {
		return Created{}, domainerror.ErrValidation("未知の useCase です")
	}
	if err := validateOptions(in.Options); err != nil {
		return Created{}, err
	}
	assetIDs := common.Unique(in.AssetIDs)
	materialIDs := common.Unique(in.ResearchMaterialIDs)
	if err := common.IDCount("assetIds", assetIDs); err != nil {
		return Created{}, err
	}
	if err := common.IDCount("researchMaterialIds", materialIDs); err != nil {
		return Created{}, err
	}
	sess := entity.PromptSession{
		ID: s.IDs.NewID(), UserID: userID, UseCase: uc, Status: vo.SessionCreated,
		InputRefs: entity.InputRefs{AssetIDs: assetIDs, ResearchMaterialIDs: materialIDs, Options: optionsMap(in.Options)},
	}
	switch uc {
	case vo.UseCaseQAGenerate:
		if err := required("companyId", in.CompanyID); err != nil {
			return Created{}, err
		}
		sess.CompanyID = in.CompanyID
	case vo.UseCaseAnswerReview:
		if err := required("interviewLogId", in.InterviewLogID); err != nil {
			return Created{}, err
		}
		sess.InterviewLogID = in.InterviewLogID
	case vo.UseCaseSummaryGenerate:
		sess.InterviewID = in.InterviewID
	case vo.UseCaseResearchSummarize:
		if err := required("companyId", in.CompanyID); err != nil {
			return Created{}, err
		}
		if len(materialIDs) != 1 {
			return Created{}, domainerror.ErrValidation("research_summarize は researchMaterialIds を 1 件だけ指定してください")
		}
		sess.CompanyID = in.CompanyID
	case vo.UseCaseESGenerate, vo.UseCaseESRefine:
		if err := required("esProjectId", in.ESProjectID); err != nil {
			return Created{}, err
		}
		if err := required("esSectionId", in.ESSectionID); err != nil {
			return Created{}, err
		}
		sess.ESProjectID, sess.ESSectionID = in.ESProjectID, in.ESSectionID
	}
	if err := s.resolveTargets(ctx, userID, &sess); err != nil {
		return Created{}, err
	}
	bi, err := s.loadInput(ctx, userID, sess)
	if err != nil {
		return Created{}, err
	}
	if uc == vo.UseCaseResearchSummarize && len(bi.Materials) == 1 && sess.CompanyID != nil && bi.Materials[0].CompanyID != *sess.CompanyID {
		return Created{}, domainerror.ErrNotFound("企業研究資料")
	}
	p, err := promptsvc.Build(bi)
	if err != nil {
		return Created{}, err
	}
	sess.TemplateVersion = p.TemplateVersion
	created, err := s.Sessions.Create(ctx, sess)
	if err != nil {
		return Created{}, err
	}
	return Created{Session: created, Prompt: p}, nil
}

func (s *Service) get(ctx context.Context, userID, id string) (entity.PromptSession, error) {
	sess, err := s.Sessions.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return entity.PromptSession{}, domainerror.ErrSessionNotFound()
		}
		return entity.PromptSession{}, err
	}
	return sess, nil
}

// expire marks this user's old unfinished sessions as abandoned (7.9).
func (s *Service) expire(ctx context.Context, userID string) error {
	now := s.Clock.Now()
	return s.Sessions.ExpireUnfinished(ctx, userID, session.ExpiryCutoff(now), now)
}

// List lists sessions by status (GET /prompt-sessions?status=created,pasted).
func (s *Service) List(ctx context.Context, userID string, statuses []string, p repository.PageParams) (repository.Page[entity.PromptSession], error) {
	var st []vo.SessionStatus
	for _, v := range statuses {
		if v == "" {
			continue
		}
		if !vo.SessionStatus(v).Valid() {
			return repository.Page[entity.PromptSession]{}, domainerror.ErrValidation("status が不正です")
		}
		st = append(st, vo.SessionStatus(v))
	}
	if err := s.expire(ctx, userID); err != nil {
		return repository.Page[entity.PromptSession]{}, err
	}
	p.Limit = common.PageLimit(p.Limit)
	return s.Sessions.List(ctx, userID, st, p)
}

// Get returns a session (used by clients after a 409 on commit, 7.6.1).
func (s *Service) Get(ctx context.Context, userID, id string) (entity.PromptSession, error) {
	return s.get(ctx, userID, id)
}

// Prompt re-generates the prompt from input_refs with the session's template version.
func (s *Service) Prompt(ctx context.Context, userID, id string) (entity.PromptSession, promptsvc.Prompt, error) {
	sess, err := s.get(ctx, userID, id)
	if err != nil {
		return sess, promptsvc.Prompt{}, err
	}
	if err := session.Check(sess.Status, session.ActionPrompt); err != nil {
		return sess, promptsvc.Prompt{}, err
	}
	bi, err := s.loadInput(ctx, userID, sess)
	if err != nil {
		return sess, promptsvc.Prompt{}, err
	}
	p, err := promptsvc.BuildVersion(bi, sess.TemplateVersion)
	return sess, p, err
}

// conflict reports the right error after a conditional transition matched no row.
func (s *Service) conflict(ctx context.Context, userID, id string) error {
	sess, err := s.get(ctx, userID, id)
	if err != nil {
		return err
	}
	return domainerror.ErrSessionStateConflict(string(sess.Status))
}

// Parse parses a pasted response and returns a preview. Nothing but the session's
// status / parse_result / pasted_at is stored (D13).
func (s *Service) Parse(ctx context.Context, userID, id, responseText string) (entity.PromptSession, promptsvc.ParseOutput, error) {
	sess, err := s.get(ctx, userID, id)
	if err != nil {
		return sess, promptsvc.ParseOutput{}, err
	}
	if err := session.Check(sess.Status, session.ActionParse); err != nil {
		return sess, promptsvc.ParseOutput{}, err
	}
	hints := promptsvc.ParseHints{QuestionCount: optionsFrom(sess.InputRefs.Options).QuestionCount}
	if sess.UseCase == vo.UseCaseQAGenerate && hints.QuestionCount == 0 {
		hints.QuestionCount = promptsvc.DefaultQuestionCount
	}
	if sess.ESSectionID != nil {
		if sec, err := s.ES.GetSection(ctx, userID, *sess.ESSectionID); err == nil {
			hints.CharacterLimit = sec.CharacterLimit
		}
	}
	out, err := promptsvc.Parse(sess.UseCase, sess.TemplateVersion, responseText, hints)
	if err != nil {
		return sess, out, err
	}
	result := out.Result
	ok, err := s.Sessions.Transition(ctx, userID, id, repository.TransitionParams{
		From: session.AllowedFrom(session.ActionParse), To: vo.SessionPasted,
		ParseResult: &result, SetPastedAt: true, Now: s.Clock.Now(),
	})
	if err != nil {
		return sess, out, err
	}
	if !ok {
		return sess, out, s.conflict(ctx, userID, id)
	}
	sess.Status, sess.ParseResult = vo.SessionPasted, &result
	return sess, out, nil
}

func validateCommit(uc vo.UseCase, in CommitInput) error {
	shapeErr := domainerror.ErrUnprocessable("保存内容の形がユースケースと一致しません")
	switch uc {
	case vo.UseCaseQAGenerate:
		if in.QAItems == nil || len(*in.QAItems) == 0 {
			return shapeErr
		}
		if len(*in.QAItems) > common.MaxIDs {
			return domainerror.ErrUnprocessable("保存する Q&A が多すぎます")
		}
		for _, it := range *in.QAItems {
			if err := common.RequireText("questionText", it.QuestionText, common.MaxTextChars); err != nil {
				return err
			}
			if err := common.RequireText("answerText", it.AnswerText, common.MaxTextChars); err != nil {
				return err
			}
		}
	case vo.UseCaseAnswerReview:
		if in.FeedbackText == nil || in.ImprovedAnswerText == nil {
			return shapeErr
		}
		if err := common.RequireText("feedbackText", *in.FeedbackText, common.MaxTextChars); err != nil {
			return err
		}
		if err := common.RequireText("improvedAnswerText", *in.ImprovedAnswerText, common.MaxTextChars); err != nil {
			return err
		}
	case vo.UseCaseResearchSummarize:
		if in.SummaryText == nil {
			return shapeErr
		}
		return common.RequireText("summaryText", *in.SummaryText, common.MaxTextChars)
	case vo.UseCaseSummaryGenerate, vo.UseCaseESGenerate, vo.UseCaseESRefine:
		if in.Text == nil {
			return shapeErr
		}
		return common.RequireText("text", *in.Text, common.MaxTextChars)
	}
	return nil
}

// Commit saves the confirmed content and moves the session to saved, in one
// transaction guarded by a conditional UPDATE (status = pasted) so it succeeds once.
func (s *Service) Commit(ctx context.Context, userID, id string, in CommitInput) (Committed, error) {
	sess, err := s.get(ctx, userID, id)
	if err != nil {
		return Committed{}, err
	}
	if err := session.Check(sess.Status, session.ActionCommit); err != nil {
		return Committed{}, err
	}
	if s.DisabledCommit[sess.UseCase] {
		return Committed{}, domainerror.ErrFeatureDisabled("commit:" + string(sess.UseCase))
	}
	if err := validateCommit(sess.UseCase, in); err != nil {
		return Committed{}, err
	}
	refs := map[string][]string{}
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		ok, err := s.Sessions.Transition(ctx, userID, id, repository.TransitionParams{
			From: session.AllowedFrom(session.ActionCommit), To: vo.SessionSaved,
			SetCompleted: true, Now: s.Clock.Now(),
		})
		if err != nil {
			return err
		}
		if !ok {
			return s.conflict(ctx, userID, id)
		}
		return s.save(ctx, userID, sess, in, refs)
	})
	if err != nil {
		return Committed{}, err
	}
	return Committed{SessionID: id, SavedRefs: refs}, nil
}

func (s *Service) save(ctx context.Context, userID string, sess entity.PromptSession, in CommitInput, refs map[string][]string) error {
	switch sess.UseCase {
	case vo.UseCaseQAGenerate:
		assetIDs := sess.InputRefs.AssetIDs
		if in.AssetIDs != nil {
			assetIDs = common.Unique(*in.AssetIDs)
			if len(assetIDs) > 0 {
				n, err := s.Assets.CountOwned(ctx, userID, assetIDs)
				if err != nil {
					return err
				}
				if err := common.EnsureOwned(n, assetIDs, "自己資産"); err != nil {
					return err
				}
			}
		}
		for _, it := range *in.QAItems {
			q, err := s.QA.Create(ctx, entity.QA{
				ID: s.IDs.NewID(), UserID: userID, CompanyID: sess.CompanyID,
				QuestionText: strings.TrimSpace(it.QuestionText), AnswerText: strings.TrimSpace(it.AnswerText),
				SourceType: vo.QASourcePreGenerated, ReviewStatus: vo.ReviewNew, AssetIDs: assetIDs,
			})
			if err != nil {
				return err
			}
			refs["qa"] = append(refs["qa"], q.ID)
		}
	case vo.UseCaseAnswerReview:
		l, err := s.Interviews.GetLog(ctx, userID, *sess.InterviewLogID)
		if err != nil {
			return common.MapRepoErr(err, "面接ログ")
		}
		l.AIFeedbackText, l.AIImprovedAnswerText = in.FeedbackText, in.ImprovedAnswerText
		if _, err := s.Interviews.UpdateLog(ctx, l); err != nil {
			return err
		}
		refs["interviewLogs"] = []string{l.ID}
		if in.SaveToQA {
			q, err := s.QA.Create(ctx, entity.QA{
				ID: s.IDs.NewID(), UserID: userID, CompanyID: sess.CompanyID,
				QuestionText: l.QuestionText, AnswerText: *in.ImprovedAnswerText,
				SourceType: vo.QASourceReviewImproved, ReviewStatus: vo.ReviewNew, AssetIDs: sess.InputRefs.AssetIDs,
			})
			if err != nil {
				return err
			}
			refs["qa"] = []string{q.ID}
		}
	case vo.UseCaseResearchSummarize:
		id := sess.InputRefs.ResearchMaterialIDs[0]
		m, err := s.Materials.Get(ctx, userID, id)
		if err != nil {
			return common.MapRepoErr(err, "企業研究資料")
		}
		m.Summary = in.SummaryText
		if _, err := s.Materials.Update(ctx, m); err != nil {
			return err
		}
		refs["researchMaterials"] = []string{m.ID}
	case vo.UseCaseSummaryGenerate:
		d, err := s.Dashboard.CreateSummary(ctx, entity.DashboardSummary{
			ID: s.IDs.NewID(), UserID: userID, Content: strings.TrimSpace(*in.Text), SourceInterviewID: sess.InterviewID,
		})
		if err != nil {
			return err
		}
		refs["dashboardSummaries"] = []string{d.ID}
	case vo.UseCaseESGenerate, vo.UseCaseESRefine:
		sec, err := s.ESService.ReplaceDraftFromAI(ctx, userID, *sess.ESSectionID, sess.UseCase, strings.TrimSpace(*in.Text))
		if err != nil {
			return err
		}
		refs["esSections"] = []string{sec.ID}
		if sess.UseCase == vo.UseCaseESGenerate && sess.ESProjectID != nil {
			if err := s.linkESRelations(ctx, userID, *sess.ESProjectID, sess.InputRefs); err != nil {
				return err
			}
		}
	}
	return nil
}

// linkESRelations adds the assets/materials used for a draft to the project (4.2.5).
func (s *Service) linkESRelations(ctx context.Context, userID, projectID string, refs entity.InputRefs) error {
	if len(refs.AssetIDs) == 0 && len(refs.ResearchMaterialIDs) == 0 {
		return nil
	}
	p, err := s.ES.GetProject(ctx, userID, projectID)
	if err != nil {
		return common.MapRepoErr(err, "ESプロジェクト")
	}
	p.AssetIDs = common.Unique(append(p.AssetIDs, refs.AssetIDs...))
	p.ResearchMaterialIDs = common.Unique(append(p.ResearchMaterialIDs, refs.ResearchMaterialIDs...))
	_, err = s.ES.UpdateProject(ctx, p)
	return err
}

// Abandon stops a session.
func (s *Service) Abandon(ctx context.Context, userID, id string) error {
	if _, err := s.get(ctx, userID, id); err != nil {
		return err
	}
	ok, err := s.Sessions.Transition(ctx, userID, id, repository.TransitionParams{
		From: session.AllowedFrom(session.ActionAbandon), To: vo.SessionAbandoned,
		SetCompleted: true, Now: s.Clock.Now(),
	})
	if err != nil {
		return err
	}
	if !ok {
		return s.conflict(ctx, userID, id)
	}
	return nil
}
