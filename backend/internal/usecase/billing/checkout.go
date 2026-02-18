// DesignDocument/7_BackendDesign.md 7.5.2 POST /billing/checkout-session
package billing

import (
	"context"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/service"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
)

type CheckoutInput struct {
	UserID    string
	PackageID string
}

type CheckoutOutput struct {
	CheckoutURL string
}

type CheckoutUsecase struct {
	packageRepo   repository.CreditPackageRepository
	stripeClient  external.StripeClient
	creditService *service.CreditService
	successURL    string
	cancelURL     string
}

func NewCheckoutUsecase(
	packageRepo repository.CreditPackageRepository,
	stripeClient external.StripeClient,
	creditService *service.CreditService,
	successURL, cancelURL string,
) *CheckoutUsecase {
	return &CheckoutUsecase{
		packageRepo:   packageRepo,
		stripeClient:  stripeClient,
		creditService: creditService,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

func (u *CheckoutUsecase) Execute(ctx context.Context, input *CheckoutInput) (*CheckoutOutput, error) {
	pkg, err := u.packageRepo.GetByID(ctx, input.PackageID)
	if err != nil {
		return nil, fmt.Errorf("get package: %w", err)
	}
	if err := u.creditService.ValidatePackage(pkg); err != nil {
		return nil, err
	}

	session, err := u.stripeClient.CreateCheckoutSession(ctx, &external.CheckoutSessionInput{
		UserID:        input.UserID,
		PackageID:     input.PackageID,
		StripePriceID: pkg.StripePriceID,
		Credits:       pkg.Credits,
		SuccessURL:    u.successURL,
		CancelURL:     u.cancelURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &CheckoutOutput{CheckoutURL: session.CheckoutURL}, nil
}
