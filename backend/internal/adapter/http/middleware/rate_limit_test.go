// DesignDocument/7_BackendDesign.md 7.8
package middleware

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	l := NewLimiter(3, time.Minute)
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d must be allowed", i+1)
		}
	}
	ok, retry := l.Allow("a")
	if ok || retry <= 0 || retry > time.Minute {
		t.Fatalf("4th request must be limited with a retry delay, got %v %v", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}
	now = now.Add(time.Minute + time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("a new window must allow again")
	}
}
