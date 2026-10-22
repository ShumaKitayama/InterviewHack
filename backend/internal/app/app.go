// Package app wires the layers together (DI). It is shared by cmd/api and the
// integration tests so both run exactly the same graph.
// DesignDocument/7_BackendDesign.md 7.2.3 cmd/api（DI）
package app

import (
	"github.com/gin-gonic/gin"

	router "github.com/interviewhack/backend/internal/adapter/http"
	"github.com/interviewhack/backend/internal/adapter/http/handler"
	"github.com/interviewhack/backend/internal/adapter/http/middleware"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/infra/postgres"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/usecase/account"
	"github.com/interviewhack/backend/internal/usecase/asset"
	"github.com/interviewhack/backend/internal/usecase/company"
	"github.com/interviewhack/backend/internal/usecase/dashboard"
	"github.com/interviewhack/backend/internal/usecase/es"
	"github.com/interviewhack/backend/internal/usecase/interview"
	"github.com/interviewhack/backend/internal/usecase/prompt"
	"github.com/interviewhack/backend/internal/usecase/qa"
	"github.com/interviewhack/backend/internal/usecase/research"
	"github.com/interviewhack/backend/internal/usecase/share"
)

// Options configures the graph.
type Options struct {
	DB           *postgres.DB
	Verifier     external.AuthVerifier
	Tokens       external.TokenGenerator
	Clock        external.Clock
	IDs          external.IDGenerator
	ShareBaseURL string
	// Feature flags (12.11).
	ShareViewEnabled       bool
	ShareCreateEnabled     bool
	CommitDisabledUseCases []string
	UserLimiter            *middleware.Limiter
	ShareLimiter           *middleware.Limiter
}

// NewRouter builds the HTTP router with all dependencies.
func NewRouter(o Options) *gin.Engine {
	db := o.DB
	users := &postgres.UserRepo{DB: db}
	assets := &postgres.AssetRepo{DB: db}
	tags := &postgres.TagRepo{DB: db}
	companies := &postgres.CompanyRepo{DB: db}
	materials := &postgres.ResearchRepo{DB: db}
	esRepo := &postgres.ESRepo{DB: db}
	interviews := &postgres.InterviewRepo{DB: db}
	qaRepo := &postgres.QARepo{DB: db}
	sessions := &postgres.PromptSessionRepo{DB: db}
	links := &postgres.ShareLinkRepo{DB: db}
	dash := &postgres.DashboardRepo{DB: db}

	disabled := map[vo.UseCase]bool{}
	for _, uc := range o.CommitDisabledUseCases {
		disabled[vo.UseCase(uc)] = true
	}

	accountSvc := &account.Service{Users: users, Tx: db, Clock: o.Clock}
	esSvc := &es.Service{ES: esRepo, Companies: companies, Assets: assets, Materials: materials, Tx: db, IDs: o.IDs}
	h := &handler.Handler{
		Account:    accountSvc,
		Assets:     &asset.Service{Assets: assets, Tags: tags, Tx: db, IDs: o.IDs},
		Companies:  &company.Service{Companies: companies, IDs: o.IDs},
		Research:   &research.Service{Materials: materials, Companies: companies, IDs: o.IDs},
		ES:         esSvc,
		Interviews: &interview.Service{Interviews: interviews, Companies: companies, IDs: o.IDs},
		QA:         &qa.Service{QA: qaRepo, Companies: companies, Assets: assets, Tx: db, IDs: o.IDs},
		Dashboard:  &dashboard.Service{Interviews: interviews, QA: qaRepo, Sessions: sessions, Dashboard: dash, Clock: o.Clock},
		Prompts: &prompt.Service{
			Sessions: sessions, Companies: companies, Assets: assets, Materials: materials, ES: esRepo, Interviews: interviews,
			QA: qaRepo, Dashboard: dash, ESService: esSvc, Tx: db, IDs: o.IDs, Clock: o.Clock, DisabledCommit: disabled,
		},
		Share: &share.Service{
			Links: links, ES: esRepo, Companies: companies, Tokens: o.Tokens, IDs: o.IDs, Clock: o.Clock,
			CreateDisabled: !o.ShareCreateEnabled, ViewDisabled: !o.ShareViewEnabled,
		},
		ShareBaseURL: o.ShareBaseURL,
	}
	return router.NewRouter(&router.RouterDeps{
		Handler: h, AuthVerifier: o.Verifier, Users: accountSvc,
		UserLimiter: o.UserLimiter, ShareLimiter: o.ShareLimiter,
	})
}
