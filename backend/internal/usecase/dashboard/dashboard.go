// Package dashboard implements GET /dashboard.
// DesignDocument/7_BackendDesign.md 7.5.4 / 1_Requirementdefinition.md 1.1.8
package dashboard

import (
	"context"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/domain/service/session"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
)

// RecentLimit is how many recent items each list shows.
const RecentLimit = 5

// Service is the dashboard use case.
type Service struct {
	Interviews repository.InterviewRepository
	QA         repository.QARepository
	Sessions   repository.PromptSessionRepository
	Dashboard  repository.DashboardRepository
	Clock      external.Clock
}

// View is the dashboard content.
type View struct {
	UpcomingInterviews  []entity.Interview
	LatestSummary       *entity.DashboardSummary
	RecentQA            []entity.QA
	RecentInterviewLogs []entity.InterviewLog
	PendingCount        int
	PendingSessions     []entity.PromptSession
}

// Get builds the dashboard. Old unfinished sessions are expired first (7.9).
func (s *Service) Get(ctx context.Context, userID string) (View, error) {
	now := s.Clock.Now()
	if err := s.Sessions.ExpireUnfinished(ctx, userID, session.ExpiryCutoff(now), now); err != nil {
		return View{}, err
	}
	var v View
	var err error
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if v.UpcomingInterviews, err = s.Interviews.UpcomingInterviews(ctx, userID, today, RecentLimit); err != nil {
		return View{}, err
	}
	if v.LatestSummary, err = s.Dashboard.LatestSummary(ctx, userID); err != nil {
		return View{}, err
	}
	if v.RecentQA, err = s.QA.Recent(ctx, userID, RecentLimit); err != nil {
		return View{}, err
	}
	if v.RecentInterviewLogs, err = s.Interviews.RecentLogs(ctx, userID, RecentLimit); err != nil {
		return View{}, err
	}
	if v.PendingCount, err = s.Sessions.CountUnfinished(ctx, userID); err != nil {
		return View{}, err
	}
	page, err := s.Sessions.List(ctx, userID, []vo.SessionStatus{vo.SessionCreated, vo.SessionPasted}, repository.PageParams{Limit: RecentLimit})
	if err != nil {
		return View{}, err
	}
	v.PendingSessions = page.Items
	return v, nil
}
