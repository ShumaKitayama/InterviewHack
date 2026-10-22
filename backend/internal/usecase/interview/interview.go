// Package interview implements interviews and interview logs.
// DesignDocument/1_Requirementdefinition.md 1.1.6 / 7_BackendDesign.md 7.5.3
package interview

import (
	"context"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the interview use case.
type Service struct {
	Interviews repository.InterviewRepository
	Companies  repository.CompanyRepository
	IDs        external.IDGenerator
}

// InterviewInput is a create/patch payload.
type InterviewInput struct {
	CompanyID     **string
	InterviewDate *time.Time
	Phase         *string
	Result        *string
	Memo          *string
}

// LogInput is a create/patch payload for a log. InterviewID is used on create only.
type LogInput struct {
	InterviewID          *string
	QuestionText         *string
	AnswerText           *string
	SelfEvaluation       *string
	AIFeedbackText       *string
	AIImprovedAnswerText *string
	Score                **int
}

func validResult(r *string) bool {
	if r == nil {
		return true
	}
	switch *r {
	case "pass", "fail", "pending", "":
		return true
	}
	return false
}

func (s *Service) applyInterview(ctx context.Context, userID string, i *entity.Interview, in InterviewInput) error {
	if in.CompanyID != nil {
		if *in.CompanyID != nil {
			if _, err := s.Companies.Get(ctx, userID, **in.CompanyID); err != nil {
				return common.MapRepoErr(err, "企業")
			}
		}
		i.CompanyID = *in.CompanyID
	}
	if in.InterviewDate != nil {
		i.InterviewDate = *in.InterviewDate
	}
	if in.Phase != nil {
		i.Phase = in.Phase
	}
	if in.Result != nil {
		i.Result = in.Result
		if *in.Result == "" {
			i.Result = nil // an empty result means "not set"
		}
	}
	if in.Memo != nil {
		i.Memo = in.Memo
	}
	if !validResult(i.Result) {
		return domainerror.ErrUnprocessable("result は pass / fail / pending のいずれかです")
	}
	if err := common.MaxLenPtr("phase", i.Phase, common.MaxTitleChars); err != nil {
		return err
	}
	return common.MaxLenPtr("memo", i.Memo, common.MaxTextChars)
}

// ListInterviews lists interviews.
func (s *Service) ListInterviews(ctx context.Context, userID string, f repository.InterviewFilter, p repository.PageParams) (repository.Page[entity.Interview], error) {
	p.Limit = common.PageLimit(p.Limit)
	return s.Interviews.ListInterviews(ctx, userID, f, p)
}

// GetInterview returns one interview.
func (s *Service) GetInterview(ctx context.Context, userID, id string) (entity.Interview, error) {
	i, err := s.Interviews.GetInterview(ctx, userID, id)
	return i, common.MapRepoErr(err, "面接")
}

// CreateInterview creates an interview.
func (s *Service) CreateInterview(ctx context.Context, userID string, in InterviewInput) (entity.Interview, error) {
	if in.InterviewDate == nil {
		return entity.Interview{}, domainerror.ErrValidation("interviewDate は必須です")
	}
	i := entity.Interview{ID: s.IDs.NewID(), UserID: userID}
	if err := s.applyInterview(ctx, userID, &i, in); err != nil {
		return entity.Interview{}, err
	}
	return s.Interviews.CreateInterview(ctx, i)
}

// UpdateInterview patches an interview.
func (s *Service) UpdateInterview(ctx context.Context, userID, id string, in InterviewInput) (entity.Interview, error) {
	i, err := s.GetInterview(ctx, userID, id)
	if err != nil {
		return entity.Interview{}, err
	}
	if err := s.applyInterview(ctx, userID, &i, in); err != nil {
		return entity.Interview{}, err
	}
	out, err := s.Interviews.UpdateInterview(ctx, i)
	return out, common.MapRepoErr(err, "面接")
}

// DeleteInterview deletes an interview and its logs.
func (s *Service) DeleteInterview(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Interviews.DeleteInterview(ctx, userID, id), "面接")
}

func applyLog(l *entity.InterviewLog, in LogInput) error {
	if in.QuestionText != nil {
		l.QuestionText = *in.QuestionText
	}
	if in.AnswerText != nil {
		l.AnswerText = *in.AnswerText
	}
	if in.SelfEvaluation != nil {
		l.SelfEvaluation = in.SelfEvaluation
	}
	if in.AIFeedbackText != nil {
		l.AIFeedbackText = in.AIFeedbackText
	}
	if in.AIImprovedAnswerText != nil {
		l.AIImprovedAnswerText = in.AIImprovedAnswerText
	}
	if in.Score != nil {
		l.Score = *in.Score
	}
	if err := common.RequireText("questionText", l.QuestionText, common.MaxTextChars); err != nil {
		return err
	}
	if err := common.RequireText("answerText", l.AnswerText, common.MaxTextChars); err != nil {
		return err
	}
	for _, f := range []struct {
		n string
		v *string
	}{{"selfEvaluation", l.SelfEvaluation}, {"aiFeedbackText", l.AIFeedbackText}, {"aiImprovedAnswerText", l.AIImprovedAnswerText}} {
		if err := common.MaxLenPtr(f.n, f.v, common.MaxTextChars); err != nil {
			return err
		}
	}
	if l.Score != nil && (*l.Score < 0 || *l.Score > 100) {
		return domainerror.ErrUnprocessable("score は 0〜100 で指定してください")
	}
	return nil
}

// ListLogs lists logs of an interview.
func (s *Service) ListLogs(ctx context.Context, userID, interviewID string, p repository.PageParams) (repository.Page[entity.InterviewLog], error) {
	if _, err := s.GetInterview(ctx, userID, interviewID); err != nil {
		return repository.Page[entity.InterviewLog]{}, err
	}
	p.Limit = common.PageLimit(p.Limit)
	return s.Interviews.ListLogs(ctx, userID, interviewID, p)
}

// GetLog returns one log.
func (s *Service) GetLog(ctx context.Context, userID, id string) (entity.InterviewLog, error) {
	l, err := s.Interviews.GetLog(ctx, userID, id)
	return l, common.MapRepoErr(err, "面接ログ")
}

// CreateLog creates a log.
func (s *Service) CreateLog(ctx context.Context, userID string, in LogInput) (entity.InterviewLog, error) {
	if in.InterviewID == nil {
		return entity.InterviewLog{}, domainerror.ErrValidation("interviewId は必須です")
	}
	if _, err := s.GetInterview(ctx, userID, *in.InterviewID); err != nil {
		return entity.InterviewLog{}, err
	}
	l := entity.InterviewLog{ID: s.IDs.NewID(), UserID: userID, InterviewID: *in.InterviewID}
	if err := applyLog(&l, in); err != nil {
		return entity.InterviewLog{}, err
	}
	return s.Interviews.CreateLog(ctx, l)
}

// UpdateLog patches a log.
func (s *Service) UpdateLog(ctx context.Context, userID, id string, in LogInput) (entity.InterviewLog, error) {
	l, err := s.GetLog(ctx, userID, id)
	if err != nil {
		return entity.InterviewLog{}, err
	}
	if err := applyLog(&l, in); err != nil {
		return entity.InterviewLog{}, err
	}
	out, err := s.Interviews.UpdateLog(ctx, l)
	return out, common.MapRepoErr(err, "面接ログ")
}

// DeleteLog deletes a log.
func (s *Service) DeleteLog(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Interviews.DeleteLog(ctx, userID, id), "面接ログ")
}
