package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

/*
以下辅助函数从 Gin Context 中安全提取 JWT 中间件注入的用户信息。
使用安全类型断言，不存在或类型不匹配时返回零值，避免 panic。
所有 handler 应使用这些函数替代直接的 c.Get() + .(string) 模式。
*/

/* GetUserID 从上下文安全提取用户 ID */
func GetUserID(c *gin.Context) string {
	v, _ := c.Get("user_id")
	s, _ := v.(string)
	return s
}

/* GetUsername 从上下文安全提取用户名 */
func GetUsername(c *gin.Context) string {
	v, _ := c.Get("username")
	s, _ := v.(string)
	return s
}

/*
GetRole 从上下文安全提取用户角色
兼容 string 和自定义 string 类型（如 models.UserRole）
*/
func GetRole(c *gin.Context) string {
	v, _ := c.Get("role")
	if s, ok := v.(string); ok {
		return s
	}
	/* 兼容自定义 string 类型（如 models.UserRole） */
	if v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

/* IsAdmin 检查当前用户是否为管理员 */
func IsAdmin(c *gin.Context) bool {
	return GetRole(c) == "admin"
}
