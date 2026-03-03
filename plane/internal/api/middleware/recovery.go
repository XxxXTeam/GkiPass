package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
Recovery 错误恢复中间件
功能：捕获 handler 中的 panic，记录结构化日志含堆栈追踪，
返回 500 JSON 响应，防止单个请求崩溃导致进程退出
*/
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				requestID := GetRequestID(c)
				zap.L().Error("请求处理 panic",
					zap.String("request_id", requestID),
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("client_ip", c.ClientIP()),
					zap.ByteString("stack", stack),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success":    false,
					"code":       500,
					"message":    "服务器内部错误",
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}
