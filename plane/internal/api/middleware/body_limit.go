package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
BodyLimit 请求体大小限制中间件
功能：限制请求体最大字节数，防止恶意超大请求导致内存耗尽（OOM）。
Gin 默认不限制请求体大小，必须手动设置。
maxBytes 建议值：API 请求 1-2MB，文件上传按需调整。
*/
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"success": false,
				"code":    http.StatusRequestEntityTooLarge,
				"message": "请求体过大",
			})
			c.Abort()
			return
		}

		/* 即使 Content-Length 未设置或被伪造，也通过 MaxBytesReader 强制限制 */
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
