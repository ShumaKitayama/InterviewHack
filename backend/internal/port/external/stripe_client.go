// DesignDocument/7_BackendDesign.md 7.5.2 Billing (Stripe)
package external

import "context"

type CheckoutSessionInput struct {
	UserID        string
	PackageID     string
	StripePriceID string
	Credits       int
	SuccessURL    string
	CancelURL     string
}

type CheckoutSessionOutput struct {
	CheckoutURL string
	SessionID   string
}

type WebhookEvent struct {
	ID          string
	Type        string
	UserID      string
	ExternalRef string
	PackageID   string
	Credits     int
}

type StripeClient interface {
	CreateCheckoutSession(ctx context.Context, input *CheckoutSessionInput) (*CheckoutSessionOutput, error)

	// VerifyWebhookSignature verifies the Stripe-Signature header and parses the event.
	VerifyWebhookSignature(payload []byte, signature string) (*WebhookEvent, error)
}
