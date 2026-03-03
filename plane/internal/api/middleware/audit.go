package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
AuditLog 审计日志中间件
功能：对写操作（POST/PUT/DELETE）自动记录审计日志，
包含用户 ID、请求方法、路径、客户端 IP、请求 ID 和响应状态码。
跳过公开端点（/auth/login、/auth/register、/captcha 等）避免噪音。
*/
func AuditLog() gin.HandlerFunc {
	/* 不需要审计的公开路径前缀 */
	skipPrefixes := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/captcha",
		"/api/v1/setup",
		"/health",
		"/metrics",
	}

	return func(c *gin.Context) {
		method := c.Request.Method

		/* 只审计写操作 */
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		/* 跳过公开端点 */
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		c.Next()

		/* 请求完成后记录审计日志 */
		userID := GetUserID(c)
		username := GetUsername(c)
		status := c.Writer.Status()

		logFunc := zap.L().Info
		if status >= 400 {
			logFunc = zap.L().Warn
		}

		logFunc("审计日志",
			zap.String("request_id", GetRequestID(c)),
			zap.String("user_id", userID),
			zap.String("username", username),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("status", status),
		)
	}
}
