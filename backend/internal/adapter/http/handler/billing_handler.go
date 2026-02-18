// DesignDocument/7_BackendDesign.md 7.5.2 Billing（Stripe）
package handler

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/billing"
)

type BillingHandler struct {
	checkoutUC  *billing.CheckoutUsecase
	webhookUC   *billing.WebhookUsecase
	packageRepo repository.CreditPackageRepository
}

func NewBillingHandler(
	checkoutUC *billing.CheckoutUsecase,
	webhookUC *billing.WebhookUsecase,
	packageRepo repository.CreditPackageRepository,
) *BillingHandler {
	return &BillingHandler{
		checkoutUC:  checkoutUC,
		webhookUC:   webhookUC,
		packageRepo: packageRepo,
	}
}

// GetPackages handles GET /billing/packages
func (h *BillingHandler) GetPackages(c *gin.Context) {
	packages, err := h.packageRepo.ListActive(c.Request.Context())
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	resp := make([]dto.PackageResponse, 0, len(packages))
	for _, p := range packages {
		resp = append(resp, dto.PackageResponse{
			ID:            p.ID,
			Name:          p.Name,
			Credits:       p.Credits,
			StripePriceID: p.StripePriceID,
		})
	}
	presenter.RespondOK(c, resp)
}

// CreateCheckoutSession handles POST /billing/checkout-session
func (h *BillingHandler) CreateCheckoutSession(c *gin.Context) {
	var req dto.CheckoutSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.checkoutUC.Execute(c.Request.Context(), &billing.CheckoutInput{
		UserID:    userID,
		PackageID: req.PackageID,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.CheckoutSessionResponse{
		CheckoutURL: output.CheckoutURL,
	})
}

// HandleWebhook handles POST /billing/webhook
func (h *BillingHandler) HandleWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if err := h.webhookUC.HandleWebhook(c.Request.Context(), payload, signature); err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, gin.H{"received": true})
}
