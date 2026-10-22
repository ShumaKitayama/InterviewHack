// Package dto defines the JSON shapes of the API (camelCase, 7.5).
// DesignDocument/7_BackendDesign.md 7.2.2 / 7.5
package dto

import (
	"encoding/json"
	"regexp"
	"time"
)

// Opt is a PATCH field that distinguishes "absent" from "null".
type Opt[T any] struct {
	Set bool
	Val *T
}

// UnmarshalJSON marks the field as present; null leaves Val nil.
func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Val = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Val = &v
	return nil
}

// Ptr returns a pointer to the value, or nil when absent. A JSON null on a
// text field is treated as the empty string.
func StrPtr(o Opt[string]) *string {
	if !o.Set {
		return nil
	}
	if o.Val == nil {
		empty := ""
		return &empty
	}
	return o.Val
}

// Nullable returns nil when absent, else a pointer to the (possibly nil) value.
func Nullable[T any](o Opt[T]) **T {
	if !o.Set {
		return nil
	}
	v := o.Val
	return &v
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a uuid.
func IsUUID(s string) bool { return uuidRe.MatchString(s) }

// AllUUID reports whether every id is a uuid.
func AllUUID(ids []string) bool {
	for _, id := range ids {
		if !IsUUID(id) {
			return false
		}
	}
	return true
}

// DateLayout is the date format of interviewDate / deadline.
const DateLayout = "2006-01-02"

// ParseDate parses YYYY-MM-DD.
func ParseDate(s string) (time.Time, bool) {
	t, err := time.Parse(DateLayout, s)
	return t, err == nil
}

// FormatDate formats a date.
func FormatDate(t time.Time) string { return t.Format(DateLayout) }

// FormatDatePtr formats an optional date.
func FormatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatDate(*t)
	return &s
}
