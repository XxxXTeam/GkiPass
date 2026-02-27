package api

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	frontend "gkipass/plane/frontend"

	"github.com/gin-gonic/gin"
)

/*
SetupFrontend 配置前端静态文件服务和 SPA fallback
功能：从 go:embed 嵌入的 Next.js 静态导出中提供文件服务，
未匹配的前端路由回退到 index.html 实现客户端路由。
开发模式下（out/ 仅含 .gitkeep）自动跳过，不影响 API 服务。
*/
func SetupFrontend(router *gin.Engine) {
	staticFS, err := fs.Sub(frontend.StaticFiles, "out")
	if err != nil {
		return
	}

	/* 预读 SPA fallback HTML，若不存在说明前端未构建，跳过 */
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		return
	}

	router.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		/* API / WebSocket / 系统端点不由前端处理 */
		if strings.HasPrefix(reqPath, "/api/") ||
			strings.HasPrefix(reqPath, "/ws/") ||
			reqPath == "/health" ||
			reqPath == "/metrics" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		/* 规范化路径，移除前导斜杠 */
		cleanPath := strings.TrimPrefix(path.Clean(reqPath), "/")

		/* 1. 精确文件匹配（JS/CSS/图片/字体等静态资源） */
		if data, err := fs.ReadFile(staticFS, cleanPath); err == nil {
			contentType := mime.TypeByExtension(path.Ext(cleanPath))
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			/* _next/static/ 下的哈希资源设置长缓存 */
			if strings.HasPrefix(cleanPath, "_next/static/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
			c.Data(http.StatusOK, contentType, data)
			return
		}

		/* 2. 目录请求 → 尝试 path/index.html（Next.js trailingSlash 模式） */
		indexPath := path.Join(cleanPath, "index.html")
		if data, err := fs.ReadFile(staticFS, indexPath); err == nil {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			return
		}

		/* 3. SPA fallback：返回根 index.html，由客户端路由接管 */
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}
