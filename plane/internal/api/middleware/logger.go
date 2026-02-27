package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

/* sensitiveQueryKeys 需要在日志中脱敏的 query 参数名 */
var sensitiveQueryKeys = map[string]bool{
	"token":    true,
	"ck":       true,
	"key":      true,
	"secret":   true,
	"password": true,
	"code":     true,
}

/*
sanitizeQuery 对 query string 中的敏感参数值进行脱敏
功能：将 token/ck/key/secret/password/code 等参数值替换为 "***"，
防止认证凭据通过日志泄漏。
*/
func sanitizeQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "***parse_error***"
	}
	for key := range values {
		if sensitiveQueryKeys[strings.ToLower(key)] {
			values.Set(key, "***")
		}
	}
	return values.Encode()
}

/*
Logger 返回 Gin 日志中间件
功能：为每个请求生成/传播 Request-ID，记录结构化访问日志，
跳过 _next/static 等嵌入前端静态资源的日志避免噪音，
对 query string 中的敏感参数自动脱敏。
*/
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		/* 跳过前端静态资源的日志记录（哈希文件名，无需审计） */
		if strings.HasPrefix(path, "/_next/static/") {
			c.Next()
			return
		}

		query := sanitizeQuery(c.Request.URL.RawQuery)

		/* 生成或复用 Request-ID */
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		logger := zap.L()

		/* 根据状态码选择日志级别 */
		logFunc := logger.Info
		if status >= 500 {
			logFunc = logger.Error
		} else if status >= 400 {
			logFunc = logger.Warn
		}

		logFunc("HTTP请求",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("status", status),
			zap.Int("size", c.Writer.Size()),
			zap.Duration("duration", duration),
		)
	}
}
