// DesignDocument/9_SecurityDesign.md 9.5.3 冪等性
// DesignDocument/8_DBDesign.dbml stripe_events
package repository

import "context"

type StripeEventRepository interface {
	// Exists checks if a Stripe event has already been processed.
	Exists(ctx context.Context, eventID string) (bool, error)

	// Save records a Stripe event ID for idempotency.
	Save(ctx context.Context, eventID string) error
}
