package middleware

import (
	"strings"

	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/auth"

	"github.com/gin-gonic/gin"
)

/*
extractCK 从请求中提取 Connection Key
优先级：X-Connection-Key 头 > Authorization CK 头 > query 参数
*/
func extractCK(c *gin.Context) string {
	if ck := c.GetHeader("X-Connection-Key"); ck != "" {
		return ck
	}
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "CK ") {
		return strings.TrimPrefix(h, "CK ")
	}
	return c.Query("ck")
}

/*
CKAuth CK 认证中间件（用户）
功能：校验 CK 格式 → 查库验证 → 检查过期/吊销 → 校验类型 → 注入用户上下文
*/
func CKAuth(d *dao.DAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		ck := extractCK(c)
		if ck == "" {
			response.GinUnauthorized(c, "Connection Key required")
			c.Abort()
			return
		}

		/* 格式校验 */
		if err := auth.ValidateCK(ck); err != nil {
			response.GinUnauthorized(c, "Invalid CK format: "+err.Error())
			c.Abort()
			return
		}

		/* 数据库查询（仅返回未吊销且未过期的记录） */
		ckObj, err := d.GetCKByKey(ck)
		if err != nil || ckObj == nil {
			response.GinUnauthorized(c, "Invalid or expired Connection Key")
			c.Abort()
			return
		}

		/* 过期/吊销双重校验（防御性编程，GetCKByKey 已过滤） */
		if ckObj.Revoked {
			response.GinUnauthorized(c, "Connection Key has been revoked")
			c.Abort()
			return
		}
		if auth.IsExpired(ckObj) {
			response.GinUnauthorized(c, "Connection Key has expired")
			c.Abort()
			return
		}

		/* 仅允许用户类型的 CK */
		if ckObj.Type != "user" {
			response.GinForbidden(c, "Invalid CK type for user access")
			c.Abort()
			return
		}

		/* 获取并校验用户 */
		user, err := d.GetUser(ckObj.NodeID) /* NodeID 字段存储 UserID */
		if err != nil || user == nil {
			response.GinUnauthorized(c, "User not found")
			c.Abort()
			return
		}
		if !user.Enabled {
			response.GinForbidden(c, "User account is disabled")
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Set("ck", ck)
		c.Next()
	}
}

/*
NodeCKAuth 节点 CK 认证中间件
功能：校验 CK 格式 → 查库验证 → 检查过期/吊销 → 校验类型 → 注入节点上下文
*/
func NodeCKAuth(d *dao.DAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		ck := extractCK(c)
		if ck == "" {
			response.GinUnauthorized(c, "Connection Key required")
			c.Abort()
			return
		}

		if err := auth.ValidateCK(ck); err != nil {
			response.GinUnauthorized(c, "Invalid CK format: "+err.Error())
			c.Abort()
			return
		}

		ckObj, err := d.GetCKByKey(ck)
		if err != nil || ckObj == nil {
			response.GinUnauthorized(c, "Invalid or expired Connection Key")
			c.Abort()
			return
		}

		if ckObj.Revoked {
			response.GinUnauthorized(c, "Connection Key has been revoked")
			c.Abort()
			return
		}
		if auth.IsExpired(ckObj) {
			response.GinUnauthorized(c, "Connection Key has expired")
			c.Abort()
			return
		}

		if ckObj.Type != "node" {
			response.GinForbidden(c, "Invalid CK type for node access")
			c.Abort()
			return
		}

		node, err := d.GetNode(ckObj.NodeID)
		if err != nil || node == nil {
			response.GinUnauthorized(c, "Node not found")
			c.Abort()
			return
		}

		c.Set("node_id", node.ID)
		c.Set("node_name", node.Name)
		c.Set("ck", ck)
		c.Next()
	}
}
