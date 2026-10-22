// Package share implements ES share links and the public share page lookup.
// DesignDocument/7_BackendDesign.md 7.5.6 / 7.6.2, DirectionChange_20261022.md D12
package share

import (
	"context"
	"errors"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	sharesvc "github.com/interviewhack/backend/internal/domain/service/share"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the share-link use case.
type Service struct {
	Links     repository.ShareLinkRepository
	ES        repository.ESRepository
	Companies repository.CompanyRepository
	Tokens    external.TokenGenerator
	IDs       external.IDGenerator
	Clock     external.Clock
	// CreateDisabled / ViewDisabled are feature flags (12.11).
	CreateDisabled bool
	ViewDisabled   bool
}

// CreateInput is POST /es-projects/{id}/share-links.
type CreateInput struct {
	SectionIDs      []string
	ExpiresInDays   int
	ShowCompanyName *bool
}

// Created holds the new link and its raw token (returned only once).
type Created struct {
	Link  entity.ShareLink
	Token string
}

// Create issues a link. Only the token hash is stored.
func (s *Service) Create(ctx context.Context, userID, projectID string, in CreateInput) (Created, error) {
	if s.CreateDisabled {
		return Created{}, domainerror.ErrFeatureDisabled("share_create")
	}
	if err := sharesvc.ValidateExpiresInDays(in.ExpiresInDays); err != nil {
		return Created{}, err
	}
	if err := common.IDCount("sectionIds", in.SectionIDs); err != nil {
		return Created{}, err
	}
	p, err := s.ES.GetProject(ctx, userID, projectID)
	if err != nil {
		return Created{}, common.MapRepoErr(err, "ESプロジェクト")
	}
	sections, err := s.ES.ListSections(ctx, userID, projectID)
	if err != nil {
		return Created{}, err
	}
	show := in.ShowCompanyName != nil && *in.ShowCompanyName
	companyName := ""
	if show {
		c, err := s.Companies.Get(ctx, userID, p.CompanyID)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return Created{}, err
		}
		companyName = c.Name
	}
	snap, err := sharesvc.BuildSnapshot(sections, in.SectionIDs, companyName, show)
	if err != nil {
		return Created{}, err
	}
	token, err := s.Tokens.Generate()
	if err != nil {
		return Created{}, err
	}
	now := s.Clock.Now()
	link, err := s.Links.Create(ctx, entity.ShareLink{
		ID: s.IDs.NewID(), UserID: userID, ESProjectID: projectID,
		TokenHash: sharesvc.HashToken(token), Snapshot: snap, ShowCompanyName: show,
		ExpiresAt: sharesvc.ExpiresAt(now, in.ExpiresInDays),
	})
	if err != nil {
		return Created{}, err
	}
	return Created{Link: link, Token: token}, nil
}

// List lists links of a project (never the URL/token).
func (s *Service) List(ctx context.Context, userID, projectID string) ([]entity.ShareLink, error) {
	if _, err := s.ES.GetProject(ctx, userID, projectID); err != nil {
		return nil, common.MapRepoErr(err, "ESプロジェクト")
	}
	return s.Links.ListByProject(ctx, userID, projectID)
}

// IsActive reports whether a link is viewable now.
func (s *Service) IsActive(l entity.ShareLink) bool {
	return sharesvc.IsActive(l, s.Clock.Now())
}

// Revoke revokes a link (idempotent).
func (s *Service) Revoke(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Links.Revoke(ctx, userID, id, s.Clock.Now()), "共有リンク")
}

// View resolves a raw token for the public share page. It returns
// FEATURE_DISABLED, SHARE_LINK_GONE (expired / revoked) or NOT_FOUND.
func (s *Service) View(ctx context.Context, token string) (entity.ShareLink, error) {
	if s.ViewDisabled {
		return entity.ShareLink{}, domainerror.ErrFeatureDisabled("share_view")
	}
	if token == "" || len(token) > 200 {
		return entity.ShareLink{}, domainerror.ErrNotFound("共有リンク")
	}
	l, err := s.Links.GetByTokenHash(ctx, sharesvc.HashToken(token))
	if err != nil {
		return entity.ShareLink{}, common.MapRepoErr(err, "共有リンク")
	}
	if !sharesvc.IsActive(l, s.Clock.Now()) {
		return l, domainerror.ErrShareLinkGone()
	}
	return l, nil
}
