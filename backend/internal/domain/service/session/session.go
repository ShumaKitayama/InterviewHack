// Package session holds the prompt_sessions state-transition rules.
// DesignDocument/7_BackendDesign.md 7.6.1 / 2_DomainDesign.md 2.4.5
package session

import (
	"time"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

// AbandonAfter is how long an unfinished session lives before it is treated as abandoned.
const AbandonAfter = 7 * 24 * time.Hour

// Action is an operation on a session.
type Action string

const (
	ActionParse   Action = "parse"
	ActionCommit  Action = "commit"
	ActionAbandon Action = "abandon"
	ActionPrompt  Action = "prompt"
)

// AllowedFrom returns the statuses an action may start from.
func AllowedFrom(a Action) []vo.SessionStatus {
	switch a {
	case ActionParse, ActionAbandon, ActionPrompt:
		return []vo.SessionStatus{vo.SessionCreated, vo.SessionPasted}
	case ActionCommit:
		return []vo.SessionStatus{vo.SessionPasted}
	}
	return nil
}

// CanTransition reports whether action a is allowed from status s.
// created→saved (commit without parse) is rejected; saved/abandoned are terminal.
func CanTransition(s vo.SessionStatus, a Action) bool {
	for _, from := range AllowedFrom(a) {
		if s == from {
			return true
		}
	}
	return false
}

// Check returns SESSION_STATE_CONFLICT when the action is not allowed.
func Check(s vo.SessionStatus, a Action) error {
	if !CanTransition(s, a) {
		return domainerror.ErrSessionStateConflict(string(s))
	}
	return nil
}

// IsExpired reports whether an unfinished session created at createdAt should be abandoned at now.
func IsExpired(s vo.SessionStatus, createdAt, now time.Time) bool {
	if s != vo.SessionCreated && s != vo.SessionPasted {
		return false
	}
	return !now.Before(createdAt.Add(AbandonAfter))
}

// ExpiryCutoff returns the created_at threshold: sessions created at or before it are expired.
func ExpiryCutoff(now time.Time) time.Time {
	return now.Add(-AbandonAfter)
}
