// DesignDocument/11_TestDesign.md 11.2.3 prompt_sessions の状態遷移
package session

import (
	"testing"
	"time"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

func TestTransitions(t *testing.T) {
	cases := []struct {
		id   string
		from vo.SessionStatus
		act  Action
		ok   bool
	}{
		{"UT-PS-001 created→pasted", vo.SessionCreated, ActionParse, true},
		{"UT-PS-001 pasted→saved", vo.SessionPasted, ActionCommit, true},
		{"UT-PS-002 created→saved", vo.SessionCreated, ActionCommit, false},
		{"UT-PS-003 pasted→pasted", vo.SessionPasted, ActionParse, true},
		{"UT-PS-004 saved→parse", vo.SessionSaved, ActionParse, false},
		{"UT-PS-004 saved→commit", vo.SessionSaved, ActionCommit, false},
		{"UT-PS-004 saved→abandon", vo.SessionSaved, ActionAbandon, false},
		{"UT-PS-005 abandoned→parse", vo.SessionAbandoned, ActionParse, false},
		{"UT-PS-005 abandoned→commit", vo.SessionAbandoned, ActionCommit, false},
		{"UT-PS-006 created→abandoned", vo.SessionCreated, ActionAbandon, true},
		{"UT-PS-006 pasted→abandoned", vo.SessionPasted, ActionAbandon, true},
		{"prompt on created", vo.SessionCreated, ActionPrompt, true},
		{"prompt on saved", vo.SessionSaved, ActionPrompt, false},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			if got := CanTransition(tc.from, tc.act); got != tc.ok {
				t.Fatalf("CanTransition = %v, want %v", got, tc.ok)
			}
			err := Check(tc.from, tc.act)
			if tc.ok && err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !tc.ok && !domainerror.Is(err, vo.ErrorCodeSessionStateConflict) {
				t.Fatalf("want SESSION_STATE_CONFLICT, got %v", err)
			}
		})
	}
}

// UT-PS-007: boundary of the 7-day expiry with a fixed clock.
func TestIsExpired(t *testing.T) {
	created := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		status vo.SessionStatus
		now    time.Time
		want   bool
	}{
		{"just before 7 days", vo.SessionCreated, created.Add(AbandonAfter - time.Second), false},
		{"exactly 7 days", vo.SessionCreated, created.Add(AbandonAfter), true},
		{"after 7 days pasted", vo.SessionPasted, created.Add(AbandonAfter + time.Hour), true},
		{"saved never expires", vo.SessionSaved, created.Add(30 * 24 * time.Hour), false},
		{"abandoned stays", vo.SessionAbandoned, created.Add(30 * 24 * time.Hour), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsExpired(tc.status, created, tc.now); got != tc.want {
				t.Fatalf("IsExpired = %v, want %v", got, tc.want)
			}
			// ExpiryCutoff must agree with IsExpired for unfinished sessions.
			if tc.status == vo.SessionCreated || tc.status == vo.SessionPasted {
				if byCutoff := !created.After(ExpiryCutoff(tc.now)); byCutoff != tc.want {
					t.Fatalf("ExpiryCutoff disagrees: %v", byCutoff)
				}
			}
		})
	}
}
