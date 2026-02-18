// DesignDocument/7_BackendDesign.md 7.2.3 cmd/api - DI初期化
package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/interviewhack/backend/internal/adapter/http/handler"
	"github.com/interviewhack/backend/internal/domain/service"
	"github.com/interviewhack/backend/internal/infra/config"
	"github.com/interviewhack/backend/internal/infra/llm"
	infrastripe "github.com/interviewhack/backend/internal/infra/stripe"
	"github.com/interviewhack/backend/internal/infra/supabase"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/usecase/ai"
	"github.com/interviewhack/backend/internal/usecase/billing"
	"github.com/interviewhack/backend/internal/usecase/credit"

	router "github.com/interviewhack/backend/internal/adapter/http"
)

type App struct {
	Router *router.RouterDeps
	DB     *supabase.Client
}

func initApp(ctx context.Context, cfg *config.Config) (*App, error) {
	db, err := supabase.NewClient(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	idGen := func() string { return uuid.New().String() }

	// Repositories
	creditRepo := supabase.NewCreditRepository(db)
	costRepo := supabase.NewCreditCostRepository(db)
	packageRepo := supabase.NewCreditPackageRepository(db)
	jobRepo := supabase.NewGenerationJobRepository(db)
	eventRepo := supabase.NewStripeEventRepository(db)

	// External clients
	stripeClient := infrastripe.NewStripeClient(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	llmClient := llm.NewOpenAIClient(cfg.OpenAIAPIKey)
	var authVerifier external.AuthVerifier // TODO: implement JWT verifier

	// Domain services
	creditService := service.NewCreditService()

	// Usecases
	consumeUC := credit.NewConsumeUsecase(creditRepo, costRepo, creditService, idGen)
	purchaseUC := credit.NewPurchaseUsecase(creditRepo, idGen)
	checkoutUC := billing.NewCheckoutUsecase(packageRepo, stripeClient, creditService, cfg.CheckoutSuccessURL, cfg.CheckoutCancelURL)
	webhookUC := billing.NewWebhookUsecase(stripeClient, eventRepo, packageRepo, purchaseUC)

	baseAI := ai.NewBaseAIUsecase(consumeUC, llmClient, jobRepo, idGen)
	qaGenerateUC := ai.NewQAGenerateUsecase(baseAI)
	answerReviewUC := ai.NewAnswerReviewUsecase(baseAI)
	summaryGenerateUC := ai.NewSummaryGenerateUsecase(baseAI)
	researchSummarizeUC := ai.NewResearchSummarizeUsecase(baseAI)
	esGenerateUC := ai.NewESGenerateUsecase(baseAI)
	esRefineUC := ai.NewESRefineUsecase(baseAI)

	// Handlers
	healthHandler := handler.NewHealthHandler()
	billingHandler := handler.NewBillingHandler(checkoutUC, webhookUC, packageRepo)
	creditHandler := handler.NewCreditHandler(creditRepo)
	aiHandler := handler.NewAIHandler(
		qaGenerateUC, answerReviewUC, summaryGenerateUC,
		researchSummarizeUC, esGenerateUC, esRefineUC,
	)

	return &App{
		Router: &router.RouterDeps{
			HealthHandler:  healthHandler,
			BillingHandler: billingHandler,
			CreditHandler:  creditHandler,
			AIHandler:      aiHandler,
			AuthVerifier:   authVerifier,
			AllowOrigins:   cfg.AllowOrigins,
		},
		DB: db,
	}, nil
}
