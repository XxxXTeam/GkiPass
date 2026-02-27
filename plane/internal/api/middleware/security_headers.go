package middleware

import (
	"github.com/gin-gonic/gin"
)

/*
SecurityHeaders 安全响应头中间件
功能：为所有 HTTP 响应添加安全防护头，防止常见 Web 攻击：
  - X-Content-Type-Options: 阻止浏览器 MIME 嗅探
  - X-Frame-Options: 阻止页面被嵌入 iframe（防点击劫持）
  - X-XSS-Protection: 启用浏览器 XSS 过滤器
  - Referrer-Policy: 限制 Referer 头泄漏完整 URL
  - Permissions-Policy: 禁用不必要的浏览器功能（摄像头/麦克风/地理位置）
  - Cache-Control: API 响应不缓存敏感数据
*/
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		/* API 路由禁止缓存敏感数据；静态资源由前端服务自行控制 */
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
		}

		c.Next()
	}
}
