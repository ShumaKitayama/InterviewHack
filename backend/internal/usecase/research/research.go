// Package research implements research materials. Text is extracted on device (D11);
// the BFF never fetches URLs.
// DesignDocument/1_Requirementdefinition.md 1.1.3 / 7_BackendDesign.md 7.5.3
package research

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the research material use case.
type Service struct {
	Materials repository.ResearchRepository
	Companies repository.CompanyRepository
	IDs       external.IDGenerator
}

// Input is a create/patch payload. CompanyID is only used on create.
type Input struct {
	CompanyID  *string
	Title      *string
	Type       *string
	SourceURL  *string
	ContentRaw *string
	Summary    *string
}

func apply(m *entity.ResearchMaterial, in Input) {
	if in.Title != nil {
		m.Title = *in.Title
	}
	if in.Type != nil {
		m.Type = *in.Type
	}
	if in.SourceURL != nil {
		m.SourceURL = in.SourceURL
	}
	if in.ContentRaw != nil {
		m.ContentRaw = in.ContentRaw
	}
	if in.Summary != nil {
		m.Summary = in.Summary
	}
}

func validate(m entity.ResearchMaterial) error {
	if err := common.RequireText("title", m.Title, common.MaxTitleChars); err != nil {
		return err
	}
	if !vo.ValidResearchMaterialType(m.Type) {
		return domainerror.ErrUnprocessable("type は url / pdf / note / image のいずれかです")
	}
	if err := common.MaxLenPtr("sourceUrl", m.SourceURL, 2000); err != nil {
		return err
	}
	if err := common.MaxLenPtr("contentRaw", m.ContentRaw, common.MaxContentRawChars); err != nil {
		return err
	}
	return common.MaxLenPtr("summary", m.Summary, common.MaxTextChars)
}

// List lists materials of a company.
func (s *Service) List(ctx context.Context, userID, companyID string, p repository.PageParams) (repository.Page[entity.ResearchMaterial], error) {
	if _, err := s.Companies.Get(ctx, userID, companyID); err != nil {
		return repository.Page[entity.ResearchMaterial]{}, common.MapRepoErr(err, "企業")
	}
	p.Limit = common.PageLimit(p.Limit)
	return s.Materials.List(ctx, userID, companyID, p)
}

// Get returns one material.
func (s *Service) Get(ctx context.Context, userID, id string) (entity.ResearchMaterial, error) {
	m, err := s.Materials.Get(ctx, userID, id)
	return m, common.MapRepoErr(err, "企業研究資料")
}

// Create creates a material.
func (s *Service) Create(ctx context.Context, userID string, in Input) (entity.ResearchMaterial, error) {
	if in.CompanyID == nil {
		return entity.ResearchMaterial{}, domainerror.ErrValidation("companyId は必須です")
	}
	if _, err := s.Companies.Get(ctx, userID, *in.CompanyID); err != nil {
		return entity.ResearchMaterial{}, common.MapRepoErr(err, "企業")
	}
	m := entity.ResearchMaterial{ID: s.IDs.NewID(), UserID: userID, CompanyID: *in.CompanyID}
	apply(&m, in)
	if err := validate(m); err != nil {
		return entity.ResearchMaterial{}, err
	}
	return s.Materials.Create(ctx, m)
}

// Update patches a material.
func (s *Service) Update(ctx context.Context, userID, id string, in Input) (entity.ResearchMaterial, error) {
	m, err := s.Get(ctx, userID, id)
	if err != nil {
		return entity.ResearchMaterial{}, err
	}
	apply(&m, in)
	if err := validate(m); err != nil {
		return entity.ResearchMaterial{}, err
	}
	out, err := s.Materials.Update(ctx, m)
	return out, common.MapRepoErr(err, "企業研究資料")
}

// Delete deletes a material.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Materials.Delete(ctx, userID, id), "企業研究資料")
}
