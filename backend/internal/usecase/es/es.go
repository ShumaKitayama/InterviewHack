// Package es implements ES projects, sections and section versions.
// DesignDocument/2_DomainDesign.md 2.2.5 / 2.2.6 / 2.2.6.1, 7_BackendDesign.md 7.5.3
package es

import (
	"context"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	esrule "github.com/interviewhack/backend/internal/domain/service/es"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the ES use case.
type Service struct {
	ES        repository.ESRepository
	Companies repository.CompanyRepository
	Assets    repository.AssetRepository
	Materials repository.ResearchRepository
	Tx        repository.TxManager
	IDs       external.IDGenerator
}

// ProjectInput is a create/patch payload for a project.
type ProjectInput struct {
	CompanyID           *string
	Status              *string
	Deadline            **time.Time
	AssetIDs            *[]string
	ResearchMaterialIDs *[]string
}

// SectionInput is a create/patch payload for a section.
type SectionInput struct {
	Title          *string
	ContentDraft   *string
	ContentFinal   *string
	CharacterLimit **int
}

func (s *Service) checkRelations(ctx context.Context, userID string, in ProjectInput, p *entity.ESProject) error {
	if in.AssetIDs != nil {
		ids := common.Unique(*in.AssetIDs)
		if err := common.IDCount("assetIds", ids); err != nil {
			return err
		}
		if len(ids) > 0 {
			n, err := s.Assets.CountOwned(ctx, userID, ids)
			if err != nil {
				return err
			}
			if err := common.EnsureOwned(n, ids, "自己資産"); err != nil {
				return err
			}
		}
		p.AssetIDs = ids
	}
	if in.ResearchMaterialIDs != nil {
		ids := common.Unique(*in.ResearchMaterialIDs)
		if err := common.IDCount("researchMaterialIds", ids); err != nil {
			return err
		}
		if len(ids) > 0 {
			n, err := s.Materials.CountOwned(ctx, userID, ids)
			if err != nil {
				return err
			}
			if err := common.EnsureOwned(n, ids, "企業研究資料"); err != nil {
				return err
			}
		}
		p.ResearchMaterialIDs = ids
	}
	return nil
}

// ListProjects lists ES projects.
func (s *Service) ListProjects(ctx context.Context, userID string, f repository.ESFilter, p repository.PageParams) (repository.Page[entity.ESProject], error) {
	if f.Status != "" && !vo.ESStatus(f.Status).Valid() {
		return repository.Page[entity.ESProject]{}, domainerror.ErrValidation("status が不正です")
	}
	p.Limit = common.PageLimit(p.Limit)
	return s.ES.ListProjects(ctx, userID, f, p)
}

// GetProject returns one project.
func (s *Service) GetProject(ctx context.Context, userID, id string) (entity.ESProject, error) {
	e, err := s.ES.GetProject(ctx, userID, id)
	return e, common.MapRepoErr(err, "ESプロジェクト")
}

// CreateProject creates a project. The initial status is not_started (2.2.5).
func (s *Service) CreateProject(ctx context.Context, userID string, in ProjectInput) (entity.ESProject, error) {
	if in.CompanyID == nil {
		return entity.ESProject{}, domainerror.ErrValidation("companyId は必須です")
	}
	if _, err := s.Companies.Get(ctx, userID, *in.CompanyID); err != nil {
		return entity.ESProject{}, common.MapRepoErr(err, "企業")
	}
	p := entity.ESProject{ID: s.IDs.NewID(), UserID: userID, CompanyID: *in.CompanyID, Status: vo.ESNotStarted}
	if in.Status != nil {
		if !vo.ESStatus(*in.Status).Valid() {
			return entity.ESProject{}, domainerror.ErrUnprocessable("status が不正です")
		}
		p.Status = vo.ESStatus(*in.Status)
	}
	if in.Deadline != nil {
		p.Deadline = *in.Deadline
	}
	if err := s.checkRelations(ctx, userID, in, &p); err != nil {
		return entity.ESProject{}, err
	}
	var out entity.ESProject
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.ES.CreateProject(ctx, p)
		return err
	})
	return out, err
}

// UpdateProject patches a project. Any status may be set; there are no transition constraints.
func (s *Service) UpdateProject(ctx context.Context, userID, id string, in ProjectInput) (entity.ESProject, error) {
	p, err := s.GetProject(ctx, userID, id)
	if err != nil {
		return entity.ESProject{}, err
	}
	if in.Status != nil {
		if !vo.ESStatus(*in.Status).Valid() {
			return entity.ESProject{}, domainerror.ErrUnprocessable("status が不正です")
		}
		p.Status = vo.ESStatus(*in.Status)
	}
	if in.Deadline != nil {
		p.Deadline = *in.Deadline
	}
	p.AssetIDs, p.ResearchMaterialIDs = nil, nil // nil = keep relations
	if err := s.checkRelations(ctx, userID, in, &p); err != nil {
		return entity.ESProject{}, err
	}
	var out entity.ESProject
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.ES.UpdateProject(ctx, p)
		return common.MapRepoErr(err, "ESプロジェクト")
	})
	return out, err
}

// DeleteProject deletes a project; its share links are deleted with it (7.5.3).
func (s *Service) DeleteProject(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.ES.DeleteProject(ctx, userID, id), "ESプロジェクト")
}

// ListSections lists sections of a project.
func (s *Service) ListSections(ctx context.Context, userID, projectID string) ([]entity.ESSection, error) {
	if _, err := s.GetProject(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.ES.ListSections(ctx, userID, projectID)
}

// GetSection returns one section.
func (s *Service) GetSection(ctx context.Context, userID, id string) (entity.ESSection, error) {
	sec, err := s.ES.GetSection(ctx, userID, id)
	return sec, common.MapRepoErr(err, "ESセクション")
}

func validateSection(sec entity.ESSection) error {
	if err := common.RequireText("title", sec.Title, common.MaxTitleChars); err != nil {
		return err
	}
	if sec.CharacterLimit != nil && (*sec.CharacterLimit < 1 || *sec.CharacterLimit > 10000) {
		return domainerror.ErrUnprocessable("characterLimit は 1〜10000 で指定してください")
	}
	if err := common.MaxLenPtr("contentDraft", sec.ContentDraft, common.MaxTextChars); err != nil {
		return err
	}
	return common.MaxLenPtr("contentFinal", sec.ContentFinal, common.MaxTextChars)
}

// touchProjectStatus applies the not_started → draft rule after a body is saved.
func (s *Service) touchProjectStatus(ctx context.Context, userID, projectID, body string) error {
	p, err := s.ES.GetProject(ctx, userID, projectID)
	if err != nil {
		return common.MapRepoErr(err, "ESプロジェクト")
	}
	next := esrule.StatusAfterContentSaved(p.Status, body)
	if next == p.Status {
		return nil
	}
	p.Status = next
	p.AssetIDs, p.ResearchMaterialIDs = nil, nil
	_, err = s.ES.UpdateProject(ctx, p)
	return err
}

func bodyOf(sec entity.ESSection) string {
	if sec.ContentDraft != nil && *sec.ContentDraft != "" {
		return *sec.ContentDraft
	}
	if sec.ContentFinal != nil {
		return *sec.ContentFinal
	}
	return ""
}

// CreateSection creates a section in a project.
func (s *Service) CreateSection(ctx context.Context, userID, projectID string, in SectionInput) (entity.ESSection, error) {
	if _, err := s.GetProject(ctx, userID, projectID); err != nil {
		return entity.ESSection{}, err
	}
	sec := entity.ESSection{ID: s.IDs.NewID(), ESProjectID: projectID, UserID: userID, Version: 1}
	applySection(&sec, in)
	if err := validateSection(sec); err != nil {
		return entity.ESSection{}, err
	}
	var out entity.ESSection
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.ES.CreateSection(ctx, sec); err != nil {
			return err
		}
		return s.touchProjectStatus(ctx, userID, projectID, bodyOf(out))
	})
	return out, err
}

func applySection(sec *entity.ESSection, in SectionInput) bool {
	changed := false
	if in.Title != nil {
		sec.Title = *in.Title
	}
	if in.ContentDraft != nil {
		sec.ContentDraft = in.ContentDraft
		changed = true
	}
	if in.ContentFinal != nil {
		sec.ContentFinal = in.ContentFinal
		changed = true
	}
	if in.CharacterLimit != nil {
		sec.CharacterLimit = *in.CharacterLimit
	}
	return changed
}

// UpdateSection patches a section. A body change increments version but does
// not create a version row (IT-VER-002).
func (s *Service) UpdateSection(ctx context.Context, userID, id string, in SectionInput) (entity.ESSection, error) {
	sec, err := s.GetSection(ctx, userID, id)
	if err != nil {
		return entity.ESSection{}, err
	}
	if applySection(&sec, in) {
		sec.Version++
	}
	if err := validateSection(sec); err != nil {
		return entity.ESSection{}, err
	}
	var out entity.ESSection
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.ES.UpdateSection(ctx, sec); err != nil {
			return common.MapRepoErr(err, "ESセクション")
		}
		return s.touchProjectStatus(ctx, userID, sec.ESProjectID, bodyOf(out))
	})
	return out, err
}

// DeleteSection deletes a section and its versions.
func (s *Service) DeleteSection(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.ES.DeleteSection(ctx, userID, id), "ESセクション")
}

// ListVersions lists versions of a section, newest first.
func (s *Service) ListVersions(ctx context.Context, userID, sectionID string) ([]entity.ESSectionVersion, error) {
	if _, err := s.GetSection(ctx, userID, sectionID); err != nil {
		return nil, err
	}
	return s.ES.ListVersions(ctx, userID, sectionID)
}

// KeepVersion stores the section's current content_draft as a version and prunes
// to MaxVersionsPerSection. It must run inside a transaction.
func (s *Service) KeepVersion(ctx context.Context, sec entity.ESSection, source vo.VersionSource) (entity.ESSectionVersion, error) {
	v, err := s.ES.CreateVersion(ctx, entity.ESSectionVersion{
		ID: s.IDs.NewID(), ESSectionID: sec.ID, UserID: sec.UserID,
		Version: sec.Version, Content: *sec.ContentDraft, Source: source,
	})
	if err != nil {
		return entity.ESSectionVersion{}, err
	}
	return v, s.ES.PruneVersions(ctx, sec.UserID, sec.ID, esrule.MaxVersionsPerSection)
}

// SaveVersion is 「この版を保存」.
func (s *Service) SaveVersion(ctx context.Context, userID, sectionID string) (entity.ESSectionVersion, error) {
	sec, err := s.GetSection(ctx, userID, sectionID)
	if err != nil {
		return entity.ESSectionVersion{}, err
	}
	if !esrule.ShouldKeepVersion(sec.ContentDraft) {
		return entity.ESSectionVersion{}, domainerror.ErrUnprocessable("本文が空のため、版を保存できません")
	}
	var out entity.ESSectionVersion
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.KeepVersion(ctx, sec, vo.VersionSourceManual)
		return err
	})
	return out, err
}

// RestoreVersion is 「この版に戻す」: keep the current body as a manual version,
// replace content_draft and increment version, all in one transaction.
func (s *Service) RestoreVersion(ctx context.Context, userID, sectionID, versionID string) (entity.ESSection, error) {
	var out entity.ESSection
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		sec, err := s.ES.GetSection(ctx, userID, sectionID)
		if err != nil {
			return common.MapRepoErr(err, "ESセクション")
		}
		v, err := s.ES.GetVersion(ctx, userID, sectionID, versionID)
		if err != nil {
			return common.MapRepoErr(err, "版")
		}
		if esrule.ShouldKeepVersion(sec.ContentDraft) {
			if _, err := s.KeepVersion(ctx, sec, vo.VersionSourceManual); err != nil {
				return err
			}
		}
		content := v.Content
		sec.ContentDraft = &content
		sec.Version++
		if out, err = s.ES.UpdateSection(ctx, sec); err != nil {
			return err
		}
		return s.touchProjectStatus(ctx, userID, sec.ESProjectID, content)
	})
	return out, err
}

// ReplaceDraftFromAI is used by the AI連携 commit (es_generate / es_refine). It keeps
// the previous body as a version, updates content_draft, increments version and
// moves the project from not_started to draft. Must run inside a transaction.
func (s *Service) ReplaceDraftFromAI(ctx context.Context, userID, sectionID string, uc vo.UseCase, text string) (entity.ESSection, error) {
	sec, err := s.ES.GetSection(ctx, userID, sectionID)
	if err != nil {
		return entity.ESSection{}, common.MapRepoErr(err, "ESセクション")
	}
	if esrule.ShouldKeepVersion(sec.ContentDraft) {
		if _, err := s.KeepVersion(ctx, sec, esrule.SourceFor(uc)); err != nil {
			return entity.ESSection{}, err
		}
	}
	sec.ContentDraft = &text
	sec.Version++
	out, err := s.ES.UpdateSection(ctx, sec)
	if err != nil {
		return entity.ESSection{}, err
	}
	return out, s.touchProjectStatus(ctx, userID, sec.ESProjectID, text)
}
