// Package common holds input checks shared by use cases.
// DesignDocument/11_TestDesign.md 11.2.5 入力バリデーション / 9_SecurityDesign.md
package common

import (
	"fmt"
	"strings"
	"unicode/utf8"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
)

// Text length limits (characters).
const (
	MaxTitleChars      = 200
	MaxTextChars       = 20000
	MaxContentRawChars = 200000
	MaxIDs             = 100
	DefaultPageLimit   = 50
	MaxPageLimit       = 100
)

// RequireText returns 422 when s is blank or longer than max.
func RequireText(field, s string, max int) error {
	if strings.TrimSpace(s) == "" {
		return domainerror.ErrUnprocessable(fmt.Sprintf("%s を入力してください", field))
	}
	return MaxLen(field, s, max)
}

// MaxLen returns 422 when s is longer than max characters.
func MaxLen(field, s string, max int) error {
	if utf8.RuneCountInString(s) > max {
		return domainerror.ErrUnprocessable(fmt.Sprintf("%s は %d 文字以内で入力してください", field, max))
	}
	return nil
}

// MaxLenPtr is MaxLen for optional fields.
func MaxLenPtr(field string, s *string, max int) error {
	if s == nil {
		return nil
	}
	return MaxLen(field, *s, max)
}

// IDCount returns 422 when more than MaxIDs ids are given.
func IDCount(field string, ids []string) error {
	if len(ids) > MaxIDs {
		return domainerror.ErrUnprocessable(fmt.Sprintf("%s は %d 件以内で指定してください", field, MaxIDs))
	}
	return nil
}

// Unique returns ids without duplicates, keeping order.
func Unique(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// PageLimit clamps a requested page size.
func PageLimit(n int) int {
	if n <= 0 {
		return DefaultPageLimit
	}
	if n > MaxPageLimit {
		return MaxPageLimit
	}
	return n
}
