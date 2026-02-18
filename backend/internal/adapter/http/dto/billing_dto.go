// DesignDocument/7_BackendDesign.md 7.5.2 Billing（Stripe）
package dto

type CheckoutSessionRequest struct {
	PackageID string `json:"packageId" binding:"required"`
}

type CheckoutSessionResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
}

type PackageResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Credits       int    `json:"credits"`
	StripePriceID string `json:"stripePriceId"`
}
