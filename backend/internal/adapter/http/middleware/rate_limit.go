// DesignDocument/7_BackendDesign.md 7.8 レート制限（コスト防御）
// user_id単位でAIエンドポイントとCheckout作成にレート制限を適用
package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/domain/vo"
)

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateLimitEntry
	maxCount int
	window   time.Duration
}

func newRateLimiter(maxCount int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		entries:  make(map[string]*rateLimitEntry),
		maxCount: maxCount,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[key]

	if !exists || now.After(entry.windowEnd) {
		rl.entries[key] = &rateLimitEntry{count: 1, windowEnd: now.Add(rl.window)}
		return true, 0
	}

	if entry.count >= rl.maxCount {
		return false, time.Until(entry.windowEnd)
	}

	entry.count++
	return true, 0
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.entries {
			if now.After(entry.windowEnd) {
				delete(rl.entries, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitAI applies per-user rate limiting to AI endpoints.
// DesignDocument/7_BackendDesign.md 7.8: /ai/ は 1分あたりN回
func RateLimitAI() gin.HandlerFunc {
	limiter := newRateLimiter(10, 1*time.Minute)
	return rateLimitHandler(limiter)
}

// RateLimitCheckout applies per-user rate limiting to checkout creation.
// DesignDocument/7_BackendDesign.md 7.8: Checkout作成も乱発防止
func RateLimitCheckout() gin.HandlerFunc {
	limiter := newRateLimiter(5, 1*time.Minute)
	return rateLimitHandler(limiter)
}

func rateLimitHandler(limiter *rateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.Next()
			return
		}

		allowed, retryAfter := limiter.allow(userID)
		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
			requestID, _ := c.Get("request_id")
			reqIDStr, _ := requestID.(string)
			c.JSON(http.StatusTooManyRequests, presenter.ErrorBody{
				Status:    "error",
				Code:      vo.ErrorCodeRateLimited.String(),
				Message:   "リクエスト回数の制限を超えました。しばらく待ってから再試行してください。",
				RequestID: reqIDStr,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
