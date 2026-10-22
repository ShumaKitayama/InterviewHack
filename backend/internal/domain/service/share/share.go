// Package share holds the ES share-link rules.
// DesignDocument/7_BackendDesign.md 7.6.2 / 2_DomainDesign.md 2.2.11
package share

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
)

// TokenBytes is the size of a raw share token (256 bits, D12: at least 128 bits).
const TokenBytes = 32

// ValidateExpiresInDays allows only 1, 7 or 30 days (no unlimited links).
func ValidateExpiresInDays(days int) error {
	switch days {
	case 1, 7, 30:
		return nil
	}
	return domainerror.ErrUnprocessable("expiresInDays は 1 / 7 / 30 のいずれかを指定してください")
}

// ExpiresAt returns the expiry time of a link created at now.
func ExpiresAt(now time.Time, days int) time.Time {
	return now.Add(time.Duration(days) * 24 * time.Hour)
}

// HashToken returns the SHA-256 hex digest stored in share_links.token_hash.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsActive reports whether a link can be viewed: not revoked and now < expires_at.
func IsActive(l entity.ShareLink, now time.Time) bool {
	return l.RevokedAt == nil && now.Before(l.ExpiresAt)
}

// SectionBody returns content_final, falling back to content_draft.
func SectionBody(s entity.ESSection) string {
	if s.ContentFinal != nil && strings.TrimSpace(*s.ContentFinal) != "" {
		return *s.ContentFinal
	}
	if s.ContentDraft != nil {
		return *s.ContentDraft
	}
	return ""
}

// BuildSnapshot freezes the selected sections, in the order given by sectionIDs.
// The company name is included only when showCompanyName is true. The user name
// is never included.
func BuildSnapshot(sections []entity.ESSection, sectionIDs []string, companyName string, showCompanyName bool) (entity.ShareSnapshot, error) {
	if len(sectionIDs) == 0 {
		return entity.ShareSnapshot{}, domainerror.ErrUnprocessable("共有するセクションを 1 つ以上選んでください")
	}
	byID := make(map[string]entity.ESSection, len(sections))
	for _, s := range sections {
		byID[s.ID] = s
	}
	snap := entity.ShareSnapshot{}
	seen := map[string]bool{}
	for _, id := range sectionIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		s, ok := byID[id]
		if !ok {
			return entity.ShareSnapshot{}, domainerror.ErrNotFound("ESセクション")
		}
		snap.Sections = append(snap.Sections, entity.ShareSnapshotSection{Title: s.Title, Body: SectionBody(s)})
	}
	if showCompanyName && companyName != "" {
		name := companyName
		snap.CompanyName = &name
	}
	return snap, nil
}
