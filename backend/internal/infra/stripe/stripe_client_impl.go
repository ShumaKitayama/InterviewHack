// DesignDocument/7_BackendDesign.md 7.5.2 / DesignDocument/9_SecurityDesign.md 9.5
package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/interviewhack/backend/internal/port/external"
	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

type stripeClientImpl struct {
	webhookSecret string
}

func NewStripeClient(secretKey, webhookSecret string) external.StripeClient {
	gostripe.Key = secretKey
	return &stripeClientImpl{webhookSecret: webhookSecret}
}

func (c *stripeClientImpl) CreateCheckoutSession(ctx context.Context, input *external.CheckoutSessionInput) (*external.CheckoutSessionOutput, error) {
	params := &gostripe.CheckoutSessionParams{
		Mode: gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		LineItems: []*gostripe.CheckoutSessionLineItemParams{
			{
				Price:    gostripe.String(input.StripePriceID),
				Quantity: gostripe.Int64(1),
			},
		},
		SuccessURL: gostripe.String(input.SuccessURL),
		CancelURL:  gostripe.String(input.CancelURL),
	}
	// DesignDocument/9_SecurityDesign.md 9.5.7: metadata に user_id と package_id を設定
	params.AddMetadata("user_id", input.UserID)
	params.AddMetadata("package_id", input.PackageID)
	params.AddMetadata("credits", fmt.Sprintf("%d", input.Credits))

	s, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &external.CheckoutSessionOutput{
		CheckoutURL: s.URL,
		SessionID:   s.ID,
	}, nil
}

func (c *stripeClientImpl) VerifyWebhookSignature(payload []byte, signature string) (*external.WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, c.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	we := &external.WebhookEvent{
		ID:   event.ID,
		Type: string(event.Type),
	}

	if event.Type == "checkout.session.completed" {
		var sess gostripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return nil, fmt.Errorf("unmarshal checkout session: %w", err)
		}
		we.UserID = sess.Metadata["user_id"]
		we.PackageID = sess.Metadata["package_id"]
		we.ExternalRef = sess.ID
	}

	return we, nil
}
