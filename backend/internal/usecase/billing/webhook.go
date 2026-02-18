// DesignDocument/7_BackendDesign.md 7.5.2 POST /billing/webhook
// DesignDocument/9_SecurityDesign.md 9.5.3 冪等性と二重付与防止
package billing

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/credit"
)

type WebhookUsecase struct {
	stripeClient    external.StripeClient
	eventRepo       repository.StripeEventRepository
	packageRepo     repository.CreditPackageRepository
	purchaseUsecase *credit.PurchaseUsecase
}

func NewWebhookUsecase(
	stripeClient external.StripeClient,
	eventRepo repository.StripeEventRepository,
	packageRepo repository.CreditPackageRepository,
	purchaseUsecase *credit.PurchaseUsecase,
) *WebhookUsecase {
	return &WebhookUsecase{
		stripeClient:    stripeClient,
		eventRepo:       eventRepo,
		packageRepo:     packageRepo,
		purchaseUsecase: purchaseUsecase,
	}
}

// HandleWebhook processes a Stripe webhook event with idempotency.
func (u *WebhookUsecase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := u.stripeClient.VerifyWebhookSignature(payload, signature)
	if err != nil {
		return fmt.Errorf("verify webhook signature: %w", err)
	}

	exists, err := u.eventRepo.Exists(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("check event existence: %w", err)
	}
	if exists {
		slog.Info("duplicate webhook event, skipping", "stripe_event_id", event.ID)
		return nil
	}

	if err := u.eventRepo.Save(ctx, event.ID); err != nil {
		return fmt.Errorf("save event: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed":
		return u.handleCheckoutCompleted(ctx, event)
	default:
		slog.Info("unhandled webhook event type", "type", event.Type)
		return nil
	}
}

func (u *WebhookUsecase) handleCheckoutCompleted(ctx context.Context, event *external.WebhookEvent) error {
	pkg, err := u.packageRepo.GetByID(ctx, event.PackageID)
	if err != nil {
		return fmt.Errorf("get package: %w", err)
	}

	return u.purchaseUsecase.Execute(ctx, &credit.PurchaseInput{
		UserID:      event.UserID,
		Credits:     pkg.Credits,
		ExternalRef: event.ExternalRef,
	})
}
