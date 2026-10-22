// Package company implements companies (セット2).
// DesignDocument/1_Requirementdefinition.md 1.1.3 / 7_BackendDesign.md 7.5.3
package company

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the company use case.
type Service struct {
	Companies repository.CompanyRepository
	IDs       external.IDGenerator
}

// Input is a create/patch payload.
type Input struct {
	Name           *string
	JobTitle       *string
	SelectionPhase *string
	MotivationMemo *string
	ReferenceURL   *string
}

func apply(c *entity.Company, in Input) {
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.JobTitle != nil {
		c.JobTitle = in.JobTitle
	}
	if in.SelectionPhase != nil {
		c.SelectionPhase = in.SelectionPhase
	}
	if in.MotivationMemo != nil {
		c.MotivationMemo = in.MotivationMemo
	}
	if in.ReferenceURL != nil {
		c.ReferenceURL = in.ReferenceURL
	}
}

func validate(c entity.Company) error {
	if err := common.RequireText("name", c.Name, common.MaxTitleChars); err != nil {
		return err
	}
	for _, f := range []struct {
		name string
		v    *string
		max  int
	}{
		{"jobTitle", c.JobTitle, common.MaxTitleChars},
		{"selectionPhase", c.SelectionPhase, common.MaxTitleChars},
		{"motivationMemo", c.MotivationMemo, common.MaxTextChars},
		{"referenceUrl", c.ReferenceURL, 2000},
	} {
		if err := common.MaxLenPtr(f.name, f.v, f.max); err != nil {
			return err
		}
	}
	return nil
}

// List lists companies.
func (s *Service) List(ctx context.Context, userID string, f repository.CompanyFilter, p repository.PageParams) (repository.Page[entity.Company], error) {
	p.Limit = common.PageLimit(p.Limit)
	return s.Companies.List(ctx, userID, f, p)
}

// Get returns one company.
func (s *Service) Get(ctx context.Context, userID, id string) (entity.Company, error) {
	c, err := s.Companies.Get(ctx, userID, id)
	return c, common.MapRepoErr(err, "企業")
}

// Create creates a company.
func (s *Service) Create(ctx context.Context, userID string, in Input) (entity.Company, error) {
	c := entity.Company{ID: s.IDs.NewID(), UserID: userID}
	apply(&c, in)
	if err := validate(c); err != nil {
		return entity.Company{}, err
	}
	return s.Companies.Create(ctx, c)
}

// Update patches a company.
func (s *Service) Update(ctx context.Context, userID, id string, in Input) (entity.Company, error) {
	c, err := s.Get(ctx, userID, id)
	if err != nil {
		return entity.Company{}, err
	}
	apply(&c, in)
	if err := validate(c); err != nil {
		return entity.Company{}, err
	}
	out, err := s.Companies.Update(ctx, c)
	return out, common.MapRepoErr(err, "企業")
}

// Delete soft-deletes a company.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Companies.SoftDelete(ctx, userID, id), "企業")
}
