package middleware

import (
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
CORS 跨域中间件
功能：仅对携带 Origin 头的跨域请求返回 CORS 响应头。
前端嵌入后同源请求不携带 Origin，无额外开销。
allowedOrigins 为空或包含 "*" 时放行所有来源（仅建议开发环境使用）；
生产环境应配置具体域名白名单。
*/
func CORS(allowedOrigins []string) gin.HandlerFunc {
	/* 预计算白名单 set，O(1) 查找 */
	allowAll := len(allowedOrigins) == 0
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = struct{}{}
	}

	if allowAll {
		logger.Warn("CORS 允许所有来源（Access-Control-Allow-Origin: *），生产环境请配置具体域名白名单")
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		/* 无 Origin 头 = 同源请求（嵌入前端），无需 CORS 处理 */
		if origin == "" {
			c.Next()
			return
		}

		/* 校验 Origin 是否在白名单中 */
		if !allowAll {
			if _, ok := originSet[origin]; !ok {
				logger.Debug("CORS 拒绝非白名单 Origin", zap.String("origin", origin))
				c.AbortWithStatus(403)
				return
			}
		}

		/* 跨域请求：回显已验证的 Origin，支持 credentials */
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
