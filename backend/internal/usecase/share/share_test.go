// Usecase unit tests with port fakes (11.1 / AGENTS.md). DesignDocument/7_BackendDesign.md 7.6.2
package share

import (
	"context"
	"testing"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	sharesvc "github.com/interviewhack/backend/internal/domain/service/share"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
)

type fakeLinks struct {
	repository.ShareLinkRepository
	byHash  map[string]entity.ShareLink
	created []entity.ShareLink
}

func (f *fakeLinks) GetByTokenHash(_ context.Context, h string) (entity.ShareLink, error) {
	l, ok := f.byHash[h]
	if !ok {
		return entity.ShareLink{}, repository.ErrNotFound
	}
	return l, nil
}

func (f *fakeLinks) Create(_ context.Context, l entity.ShareLink) (entity.ShareLink, error) {
	f.created = append(f.created, l)
	return l, nil
}

type fakeES struct {
	repository.ESRepository
	sections []entity.ESSection
}

func (f *fakeES) GetProject(_ context.Context, userID, id string) (entity.ESProject, error) {
	if userID != "u1" || id != "p1" {
		return entity.ESProject{}, repository.ErrNotFound
	}
	return entity.ESProject{ID: "p1", UserID: "u1", CompanyID: "c1"}, nil
}

func (f *fakeES) ListSections(context.Context, string, string) ([]entity.ESSection, error) {
	return f.sections, nil
}

type fakeCompanies struct{ repository.CompanyRepository }

func (fakeCompanies) Get(context.Context, string, string) (entity.Company, error) {
	return entity.Company{Name: "株式会社テスト"}, nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type fixedToken struct{}

func (fixedToken) Generate() (string, error) { return "raw-token", nil }

type seqIDs struct{}

func (seqIDs) NewID() string { return "l1" }

func newService(now time.Time) (*Service, *fakeLinks) {
	draft := "本文"
	links := &fakeLinks{byHash: map[string]entity.ShareLink{}}
	return &Service{
		Links: links, ES: &fakeES{sections: []entity.ESSection{{ID: "s1", Title: "志望動機", ContentDraft: &draft}}},
		Companies: fakeCompanies{}, Tokens: fixedToken{}, IDs: seqIDs{}, Clock: fixedClock{now},
	}, links
}

func TestCreateStoresOnlyHash(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	svc, links := newService(now)
	res, err := svc.Create(context.Background(), "u1", "p1", CreateInput{SectionIDs: []string{"s1"}, ExpiresInDays: 7})
	if err != nil {
		t.Fatal(err)
	}
	stored := links.created[0]
	if res.Token != "raw-token" || stored.TokenHash != sharesvc.HashToken("raw-token") || stored.TokenHash == "raw-token" {
		t.Fatalf("stored = %+v", stored)
	}
	if !stored.ExpiresAt.Equal(now.Add(7*24*time.Hour)) || stored.ShowCompanyName || stored.Snapshot.CompanyName != nil {
		t.Fatalf("defaults: %+v (UT-SH-009)", stored)
	}
	if _, err := svc.Create(context.Background(), "u2", "p1", CreateInput{SectionIDs: []string{"s1"}, ExpiresInDays: 7}); !domainerror.Is(err, vo.ErrorCodeNotFound) {
		t.Fatalf("other user's project must be NOT_FOUND, got %v", err)
	}
	svc.CreateDisabled = true
	if _, err := svc.Create(context.Background(), "u1", "p1", CreateInput{SectionIDs: []string{"s1"}, ExpiresInDays: 7}); !domainerror.Is(err, vo.ErrorCodeFeatureDisabled) {
		t.Fatalf("disabled create, got %v", err)
	}
}

func TestView(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	svc, links := newService(now)
	links.byHash[sharesvc.HashToken("ok")] = entity.ShareLink{ID: "l1", ExpiresAt: now.Add(time.Hour)}
	links.byHash[sharesvc.HashToken("old")] = entity.ShareLink{ID: "l2", ExpiresAt: now}
	if _, err := svc.View(context.Background(), "ok"); err != nil {
		t.Fatalf("active link: %v", err)
	}
	if _, err := svc.View(context.Background(), "old"); !domainerror.Is(err, vo.ErrorCodeShareLinkGone) {
		t.Fatalf("expired link, got %v", err)
	}
	if _, err := svc.View(context.Background(), "none"); !domainerror.Is(err, vo.ErrorCodeNotFound) {
		t.Fatalf("unknown link, got %v", err)
	}
	svc.ViewDisabled = true
	if _, err := svc.View(context.Background(), "ok"); !domainerror.Is(err, vo.ErrorCodeFeatureDisabled) {
		t.Fatalf("disabled view, got %v", err)
	}
}
