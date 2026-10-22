// Package account implements user resolution, profile and account deletion.
// DesignDocument/6_AuthAccountCahrgingDesign.md 6.3 / 7_BackendDesign.md 7.4 / 7.5.2
package account

import (
	"context"
	"strings"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the account use case.
type Service struct {
	Users repository.UserRepository
	Tx    repository.TxManager
	Clock external.Clock
}

// Resolve returns the user for verified claims, creating it on first access.
// Deleted users are rejected with 401 (7.4 step 4).
func (s *Service) Resolve(ctx context.Context, claims external.Claims) (entity.User, error) {
	if claims.UID == "" {
		return entity.User{}, domainerror.ErrUnauthorized()
	}
	u, err := s.Users.GetOrCreateByFirebaseUID(ctx, claims.UID, claims.Email)
	if err != nil {
		return entity.User{}, err
	}
	if u.DeletedAt != nil {
		return entity.User{}, domainerror.ErrUnauthorized()
	}
	return u, nil
}

// Get returns the user.
func (s *Service) Get(ctx context.Context, userID string) (entity.User, error) {
	u, err := s.Users.Get(ctx, userID)
	return u, common.MapRepoErr(err, "ユーザー")
}

// UpdateDisplayName updates display_name. An empty string clears it.
func (s *Service) UpdateDisplayName(ctx context.Context, userID string, displayName *string) (entity.User, error) {
	if displayName == nil {
		return s.Get(ctx, userID)
	}
	if err := common.MaxLen("displayName", *displayName, common.MaxTitleChars); err != nil {
		return entity.User{}, err
	}
	v := strings.TrimSpace(*displayName)
	var p *string
	if v != "" {
		p = &v
	}
	u, err := s.Users.UpdateDisplayName(ctx, userID, p)
	return u, common.MapRepoErr(err, "ユーザー")
}

// Delete deletes the account in one transaction (6.3.3).
func (s *Service) Delete(ctx context.Context, userID string) error {
	return s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		return s.Users.DeleteAccount(ctx, userID, s.Clock.Now())
	})
}
