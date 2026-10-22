// Package asset implements self assets (セット1) and tags.
// DesignDocument/1_Requirementdefinition.md 1.1.2 / 7_BackendDesign.md 7.5.3
package asset

import (
	"context"
	"strings"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the asset use case.
type Service struct {
	Assets repository.AssetRepository
	Tags   repository.TagRepository
	Tx     repository.TxManager
	IDs    external.IDGenerator
}

// Input is a create/patch payload. Nil fields are left unchanged on patch.
type Input struct {
	Type    *string
	Title   *string
	Content *string
	TagIDs  *[]string
}

func (s *Service) checkTags(ctx context.Context, userID string, ids []string) ([]string, error) {
	ids = common.Unique(ids)
	if err := common.IDCount("tagIds", ids); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return ids, nil
	}
	n, err := s.Tags.CountOwned(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	return ids, common.EnsureOwned(n, ids, "タグ")
}

func validate(a entity.Asset) error {
	if !vo.ValidAssetType(a.Type) {
		return domainerror.ErrUnprocessable("type が不正です")
	}
	if err := common.RequireText("title", a.Title, common.MaxTitleChars); err != nil {
		return err
	}
	return common.RequireText("content", a.Content, common.MaxTextChars)
}

// List lists assets.
func (s *Service) List(ctx context.Context, userID string, f repository.AssetFilter, p repository.PageParams) (repository.Page[entity.Asset], error) {
	p.Limit = common.PageLimit(p.Limit)
	return s.Assets.List(ctx, userID, f, p)
}

// Get returns one asset.
func (s *Service) Get(ctx context.Context, userID, id string) (entity.Asset, error) {
	a, err := s.Assets.Get(ctx, userID, id)
	return a, common.MapRepoErr(err, "自己資産")
}

// Create creates an asset.
func (s *Service) Create(ctx context.Context, userID string, in Input) (entity.Asset, error) {
	a := entity.Asset{ID: s.IDs.NewID(), UserID: userID}
	if in.Type != nil {
		a.Type = *in.Type
	}
	if in.Title != nil {
		a.Title = *in.Title
	}
	if in.Content != nil {
		a.Content = *in.Content
	}
	if err := validate(a); err != nil {
		return entity.Asset{}, err
	}
	if in.TagIDs != nil {
		ids, err := s.checkTags(ctx, userID, *in.TagIDs)
		if err != nil {
			return entity.Asset{}, err
		}
		a.TagIDs = ids
	}
	var out entity.Asset
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.Assets.Create(ctx, a)
		return err
	})
	return out, err
}

// Update patches an asset.
func (s *Service) Update(ctx context.Context, userID, id string, in Input) (entity.Asset, error) {
	a, err := s.Get(ctx, userID, id)
	if err != nil {
		return entity.Asset{}, err
	}
	if in.Type != nil {
		a.Type = *in.Type
	}
	if in.Title != nil {
		a.Title = *in.Title
	}
	if in.Content != nil {
		a.Content = *in.Content
	}
	if err := validate(a); err != nil {
		return entity.Asset{}, err
	}
	if in.TagIDs != nil {
		ids, err := s.checkTags(ctx, userID, *in.TagIDs)
		if err != nil {
			return entity.Asset{}, err
		}
		a.TagIDs = ids
	} else {
		a.TagIDs = nil // nil means "keep" for the repository
	}
	var out entity.Asset
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.Assets.Update(ctx, a)
		return common.MapRepoErr(err, "自己資産")
	})
	return out, err
}

// Delete soft-deletes an asset.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Assets.SoftDelete(ctx, userID, id), "自己資産")
}

// ListTags lists tags.
func (s *Service) ListTags(ctx context.Context, userID string) ([]entity.AssetTag, error) {
	return s.Tags.List(ctx, userID)
}

// CreateTag creates a tag. Duplicate names are 409 CONFLICT.
func (s *Service) CreateTag(ctx context.Context, userID, name string) (entity.AssetTag, error) {
	name = strings.TrimSpace(name)
	if err := common.RequireText("name", name, common.MaxTitleChars); err != nil {
		return entity.AssetTag{}, err
	}
	t, err := s.Tags.Create(ctx, entity.AssetTag{ID: s.IDs.NewID(), UserID: userID, Name: name})
	return t, common.MapRepoErr(err, "タグ")
}

// UpdateTag renames a tag.
func (s *Service) UpdateTag(ctx context.Context, userID, id, name string) (entity.AssetTag, error) {
	name = strings.TrimSpace(name)
	if err := common.RequireText("name", name, common.MaxTitleChars); err != nil {
		return entity.AssetTag{}, err
	}
	t, err := s.Tags.Update(ctx, entity.AssetTag{ID: id, UserID: userID, Name: name})
	return t, common.MapRepoErr(err, "タグ")
}

// DeleteTag deletes a tag.
func (s *Service) DeleteTag(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.Tags.Delete(ctx, userID, id), "タグ")
}
