package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

/*
LoginRateLimiter 登录端点专用限流器
功能：基于 IP 地址的滑动窗口限流，防止暴力破解密码。
每个 IP 在窗口期内最多允许 maxAttempts 次请求，超出后返回 429。
内置自动清理机制，每 5 分钟清除过期记录，防止内存泄漏。
*/
type LoginRateLimiter struct {
	attempts    map[string][]time.Time /* IP → 请求时间戳列表 */
	mu          sync.Mutex
	maxAttempts int           /* 窗口期内最大尝试次数 */
	window      time.Duration /* 滑动窗口时长 */
	stopChan    chan struct{} /* 用于停止 cleanup goroutine，防止泄漏 */
}

/*
NewLoginRateLimiter 创建登录限流器
maxAttempts: 窗口期内最大尝试次数（建议 5-10）
window: 滑动窗口时长（建议 5-15 分钟）
*/
func NewLoginRateLimiter(maxAttempts int, window time.Duration) *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
		stopChan:    make(chan struct{}),
	}

	/* 后台定时清理过期记录 */
	go rl.cleanup()

	return rl
}

/* Stop 停止限流器的后台清理 goroutine，防止泄漏 */
func (rl *LoginRateLimiter) Stop() {
	close(rl.stopChan)
}

/*
Middleware 返回 Gin 中间件
功能：检查当前 IP 在窗口期内的请求次数，超限返回 429
*/
func (rl *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		/* 清除窗口外的过期记录 */
		attempts := rl.attempts[ip]
		valid := attempts[:0]
		for _, t := range attempts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}

		if len(valid) >= rl.maxAttempts {
			rl.attempts[ip] = valid
			rl.mu.Unlock()

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"code":    http.StatusTooManyRequests,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		/* 记录本次请求 */
		rl.attempts[ip] = append(valid, now)
		rl.mu.Unlock()

		c.Next()
	}
}

/* cleanup 定期清理过期的限流记录，防止内存泄漏 */
func (rl *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-rl.window)
			for ip, attempts := range rl.attempts {
				valid := attempts[:0]
				for _, t := range attempts {
					if t.After(cutoff) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(rl.attempts, ip)
				} else {
					rl.attempts[ip] = valid
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}
