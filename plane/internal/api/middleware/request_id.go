package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

/*
RequestID 请求 ID 中间件
功能：为每个请求生成唯一 ID，写入 X-Request-ID 响应头和 context，
用于日志链路追踪和问题排查。若客户端已携带 X-Request-ID 则复用。
*/
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

/*
GetRequestID 从 context 获取请求 ID
*/
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return ""
}
