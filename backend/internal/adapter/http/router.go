// DesignDocument/7_BackendDesign.md 7.5 API一覧
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/handler"
	"github.com/interviewhack/backend/internal/adapter/http/middleware"
	"github.com/interviewhack/backend/internal/port/external"
)

type RouterDeps struct {
	HealthHandler  *handler.HealthHandler
	BillingHandler *handler.BillingHandler
	CreditHandler  *handler.CreditHandler
	AIHandler      *handler.AIHandler
	AuthVerifier   external.AuthVerifier
	AllowOrigins   []string
}

func NewRouter(deps *RouterDeps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(deps.AllowOrigins))
	r.Use(middleware.Logger())

	r.GET("/health", deps.HealthHandler.Health)

	billing := r.Group("/billing")
	{
		billing.GET("/packages", deps.BillingHandler.GetPackages)
		billing.POST("/webhook", deps.BillingHandler.HandleWebhook)

		billingAuth := billing.Group("")
		billingAuth.Use(middleware.Auth(deps.AuthVerifier))
		billingAuth.Use(middleware.RateLimitCheckout())
		billingAuth.POST("/checkout-session", deps.BillingHandler.CreateCheckoutSession)
	}

	credits := r.Group("/credits")
	credits.Use(middleware.Auth(deps.AuthVerifier))
	{
		credits.GET("/balance", deps.CreditHandler.GetBalance)
		credits.GET("/transactions", deps.CreditHandler.GetTransactions)
		credits.POST("/onboarding-bonus", deps.CreditHandler.ClaimOnboardingBonus)
	}

	// DesignDocument/7_BackendDesign.md 7.8: /ai/ は user_id 単位でレート制限
	aiGroup := r.Group("/ai")
	aiGroup.Use(middleware.Auth(deps.AuthVerifier))
	aiGroup.Use(middleware.RateLimitAI())
	{
		aiGroup.POST("/qa-generate", deps.AIHandler.QAGenerate)
		aiGroup.POST("/answer-review", deps.AIHandler.AnswerReview)
		aiGroup.POST("/summary-generate", deps.AIHandler.SummaryGenerate)
		aiGroup.POST("/research-summarize", deps.AIHandler.ResearchSummarize)
		aiGroup.POST("/es-generate", deps.AIHandler.ESGenerate)
		aiGroup.POST("/es-refine", deps.AIHandler.ESRefine)
	}

	return r
}
