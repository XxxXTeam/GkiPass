package middleware

import (
	"gkipass/plane/internal/api/response"

	"github.com/gin-gonic/gin"
)

/*
AdminAuth 管理员权限中间件
功能：检查 JWT 中间件设置的 role 字段是否为 admin，
使用安全类型断言避免非字符串类型导致的误判。
*/
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			response.GinForbidden(c, "Admin access required")
			c.Abort()
			return
		}
		role, ok := roleVal.(string)
		if !ok || role != "admin" {
			response.GinForbidden(c, "Admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}
