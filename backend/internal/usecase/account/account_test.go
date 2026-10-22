// Usecase unit tests with port fakes. DesignDocument/7_BackendDesign.md 7.4
package account

import (
	"context"
	"testing"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/external"
	"github.com/interviewhack/backend/internal/port/repository"
)

type fakeUsers struct {
	repository.UserRepository
	users   map[string]entity.User
	deleted []string
}

func (f *fakeUsers) GetOrCreateByFirebaseUID(_ context.Context, uid string, email *string) (entity.User, error) {
	u, ok := f.users[uid]
	if !ok {
		u = entity.User{ID: "id-" + uid, FirebaseUID: uid, Email: email}
		f.users[uid] = u
	}
	return u, nil
}

func (f *fakeUsers) DeleteAccount(_ context.Context, userID string, _ time.Time) error {
	f.deleted = append(f.deleted, userID)
	return nil
}

type passTx struct{ calls int }

func (p *passTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	p.calls++
	return fn(ctx)
}

type clock struct{}

func (clock) Now() time.Time { return time.Unix(0, 0) }

func TestResolve(t *testing.T) {
	deletedAt := time.Now()
	users := &fakeUsers{users: map[string]entity.User{"gone": {ID: "id-gone", FirebaseUID: "gone", DeletedAt: &deletedAt}}}
	svc := &Service{Users: users, Tx: &passTx{}, Clock: clock{}}
	u, err := svc.Resolve(context.Background(), external.Claims{UID: "new"})
	if err != nil || u.ID != "id-new" {
		t.Fatalf("new user: %+v %v", u, err)
	}
	if _, err := svc.Resolve(context.Background(), external.Claims{UID: "gone"}); !domainerror.Is(err, vo.ErrorCodeUnauthorized) {
		t.Fatalf("deleted user must be 401, got %v", err)
	}
	if _, err := svc.Resolve(context.Background(), external.Claims{}); !domainerror.Is(err, vo.ErrorCodeUnauthorized) {
		t.Fatalf("empty uid must be 401, got %v", err)
	}
}

func TestDeleteRunsInTransaction(t *testing.T) {
	users := &fakeUsers{users: map[string]entity.User{}}
	tx := &passTx{}
	svc := &Service{Users: users, Tx: tx, Clock: clock{}}
	if err := svc.Delete(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	if tx.calls != 1 || len(users.deleted) != 1 || users.deleted[0] != "u1" {
		t.Fatalf("tx=%d deleted=%v", tx.calls, users.deleted)
	}
}
