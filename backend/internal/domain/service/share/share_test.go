// DesignDocument/11_TestDesign.md 11.2.4 共有リンク（トークン・期限）
package share

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

func sp(s string) *string { return &s }

// UT-SH-002: only a SHA-256 hex digest is stored.
func TestHashToken(t *testing.T) {
	h := HashToken("raw-token")
	if len(h) != 64 || strings.Contains(h, "raw-token") || h == HashToken("raw-token2") || h != HashToken("raw-token") {
		t.Fatalf("unexpected hash %q", h)
	}
}

// UT-SH-003
func TestValidateExpiresInDays(t *testing.T) {
	for _, d := range []int{1, 7, 30} {
		if err := ValidateExpiresInDays(d); err != nil {
			t.Fatalf("%d must be allowed: %v", d, err)
		}
	}
	for _, d := range []int{0, -1, 2, 31, 365} {
		if err := ValidateExpiresInDays(d); !domainerror.Is(err, vo.ErrorCodeUnprocessable) {
			t.Fatalf("%d must be rejected, got %v", d, err)
		}
	}
}

// UT-SH-004 / 005
func TestIsActive(t *testing.T) {
	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	l := entity.ShareLink{ExpiresAt: ExpiresAt(created, 7)}
	if !IsActive(l, l.ExpiresAt.Add(-time.Second)) {
		t.Fatal("just before expiry must be active")
	}
	if IsActive(l, l.ExpiresAt) {
		t.Fatal("exactly at expiry must be inactive")
	}
	if IsActive(l, l.ExpiresAt.Add(time.Second)) {
		t.Fatal("after expiry must be inactive")
	}
	revoked := created.Add(time.Hour)
	l.RevokedAt = &revoked
	if IsActive(l, created.Add(2*time.Hour)) {
		t.Fatal("revoked link must be inactive")
	}
}

// UT-SH-006 / 007 / 008
func TestBuildSnapshot(t *testing.T) {
	sections := []entity.ESSection{
		{ID: "s1", Title: "志望動機", ContentDraft: sp("下書き"), ContentFinal: sp("完成版")},
		{ID: "s2", Title: "自己PR", ContentDraft: sp("自己PRの下書き")},
		{ID: "s3", Title: "ガクチカ", ContentDraft: sp("非公開")},
	}
	snap, err := BuildSnapshot(sections, []string{"s2", "s1"}, "株式会社テスト", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Sections) != 2 || snap.Sections[0].Title != "自己PR" || snap.Sections[1].Body != "完成版" || snap.Sections[0].Body != "自己PRの下書き" {
		t.Fatalf("snapshot = %+v", snap)
	}
	if snap.CompanyName != nil {
		t.Fatal("company name must not be included when showCompanyName=false")
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "非公開") || strings.Contains(string(b), "companyName") {
		t.Fatalf("unselected sections / company leaked: %s", b)
	}
	snap, _ = BuildSnapshot(sections, []string{"s1"}, "株式会社テスト", true)
	if snap.CompanyName == nil || *snap.CompanyName != "株式会社テスト" {
		t.Fatal("company name must be included when showCompanyName=true")
	}
	if _, err := BuildSnapshot(sections, nil, "", false); !domainerror.Is(err, vo.ErrorCodeUnprocessable) {
		t.Fatalf("empty selection must be 422, got %v", err)
	}
	if _, err := BuildSnapshot(sections, []string{"other"}, "", false); !domainerror.Is(err, vo.ErrorCodeNotFound) {
		t.Fatalf("unknown section must be NOT_FOUND, got %v", err)
	}
}
