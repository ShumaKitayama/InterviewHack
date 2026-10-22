// Package qa implements the Q&A bank.
// DesignDocument/1_Requirementdefinition.md 1.1.7 / 7_BackendDesign.md 7.5.3
package qa

import (
	"context"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/common"
)

// Service is the Q&A use case.
type Service struct {
	QA        repository.QARepository
	Companies repository.CompanyRepository
	Assets    repository.AssetRepository
	Tx        repository.TxManager
	IDs       external.IDGenerator
}

// Input is a create/patch payload.
type Input struct {
	CompanyID    **string
	QuestionText *string
	AnswerText   *string
	SourceType   *string
	IsFavorite   *bool
	ReviewStatus *string
	AssetIDs     *[]string
}

func (s *Service) apply(ctx context.Context, userID string, q *entity.QA, in Input) error {
	if in.CompanyID != nil {
		if *in.CompanyID != nil {
			if _, err := s.Companies.Get(ctx, userID, **in.CompanyID); err != nil {
				return common.MapRepoErr(err, "企業")
			}
		}
		q.CompanyID = *in.CompanyID
	}
	if in.QuestionText != nil {
		q.QuestionText = *in.QuestionText
	}
	if in.AnswerText != nil {
		q.AnswerText = *in.AnswerText
	}
	if in.SourceType != nil {
		switch vo.QASourceType(*in.SourceType) {
		case vo.QASourcePreGenerated, vo.QASourceReviewImproved:
			q.SourceType = vo.QASourceType(*in.SourceType)
		default:
			return domainerror.ErrUnprocessable("sourceType が不正です")
		}
	}
	if in.IsFavorite != nil {
		q.IsFavorite = *in.IsFavorite
	}
	if in.ReviewStatus != nil {
		if !vo.ReviewStatus(*in.ReviewStatus).Valid() {
			return domainerror.ErrUnprocessable("reviewStatus は new / reviewed / needs_review のいずれかです")
		}
		q.ReviewStatus = vo.ReviewStatus(*in.ReviewStatus)
	}
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
		q.AssetIDs = ids
	}
	if err := common.RequireText("questionText", q.QuestionText, common.MaxTextChars); err != nil {
		return err
	}
	return common.RequireText("answerText", q.AnswerText, common.MaxTextChars)
}

// List lists Q&A. Favorites come first (2.2.9).
func (s *Service) List(ctx context.Context, userID string, f repository.QAFilter, p repository.PageParams) (repository.Page[entity.QA], error) {
	if f.ReviewStatus != "" && !vo.ReviewStatus(f.ReviewStatus).Valid() {
		return repository.Page[entity.QA]{}, domainerror.ErrValidation("reviewStatus が不正です")
	}
	p.Limit = common.PageLimit(p.Limit)
	return s.QA.List(ctx, userID, f, p)
}

// Get returns one Q&A.
func (s *Service) Get(ctx context.Context, userID, id string) (entity.QA, error) {
	q, err := s.QA.Get(ctx, userID, id)
	return q, common.MapRepoErr(err, "Q&A")
}

// Create creates a Q&A.
func (s *Service) Create(ctx context.Context, userID string, in Input) (entity.QA, error) {
	q := entity.QA{ID: s.IDs.NewID(), UserID: userID, SourceType: vo.QASourcePreGenerated, ReviewStatus: vo.ReviewNew}
	if err := s.apply(ctx, userID, &q, in); err != nil {
		return entity.QA{}, err
	}
	var out entity.QA
	err := s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.QA.Create(ctx, q)
		return err
	})
	return out, err
}

// Update patches a Q&A.
func (s *Service) Update(ctx context.Context, userID, id string, in Input) (entity.QA, error) {
	q, err := s.Get(ctx, userID, id)
	if err != nil {
		return entity.QA{}, err
	}
	q.AssetIDs = nil // nil = keep relations
	if err := s.apply(ctx, userID, &q, in); err != nil {
		return entity.QA{}, err
	}
	var out entity.QA
	err = s.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.QA.Update(ctx, q)
		return common.MapRepoErr(err, "Q&A")
	})
	return out, err
}

// Delete deletes a Q&A.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return common.MapRepoErr(s.QA.Delete(ctx, userID, id), "Q&A")
}
