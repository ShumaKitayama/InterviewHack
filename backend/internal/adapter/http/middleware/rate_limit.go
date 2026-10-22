// DesignDocument/7_BackendDesign.md 7.8 レート制限（乱用防止）
// Per-instance in-memory fixed-window counters (sufficient for phase 1).
package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

// Limits from 7.8.
const (
	UserRequestsPerMinute = 120
	IPSharePerMinute      = 30
)

type window struct {
	count int
	end   time.Time
}

// Limiter is a fixed-window counter keyed by user id or IP.
type Limiter struct {
	mu     sync.Mutex
	max    int
	period time.Duration
	now    func() time.Time
	byKey  map[string]*window
	calls  int
}

// NewLimiter returns a limiter allowing max requests per period.
func NewLimiter(max int, period time.Duration) *Limiter {
	return &Limiter{max: max, period: period, now: time.Now, byKey: map[string]*window{}}
}

// Allow records a request and reports whether it is allowed, with the retry delay.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.calls++
	if l.calls%1000 == 0 {
		for k, w := range l.byKey {
			if now.After(w.end) {
				delete(l.byKey, k)
			}
		}
	}
	w, ok := l.byKey[key]
	if !ok || now.After(w.end) {
		l.byKey[key] = &window{count: 1, end: now.Add(l.period)}
		return true, 0
	}
	if w.count >= l.max {
		return false, w.end.Sub(now)
	}
	w.count++
	return true, 0
}

func reject(c *gin.Context, retry time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
	presenter.RespondError(c, domainerror.New(vo.ErrorCodeRateLimited, "リクエスト回数の制限を超えました。しばらく待ってから再試行してください"))
}

// RateLimitUser limits authenticated requests per user id. Use after Auth.
func RateLimitUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, retry := l.Allow("u:" + c.GetString(UserIDKey)); !ok {
			reject(c, retry)
			return
		}
		c.Next()
	}
}

// RateLimitIP limits requests per client IP (share page brute force, IT-SH-009).
// onReject renders the rejection; when nil the JSON error is used.
func RateLimitIP(l *Limiter, onReject func(c *gin.Context, retry time.Duration)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, retry := l.Allow("ip:" + c.ClientIP()); !ok {
			if onReject != nil {
				c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
				onReject(c, retry)
				c.Abort()
				return
			}
			reject(c, retry)
			return
		}
		c.Next()
	}
}
