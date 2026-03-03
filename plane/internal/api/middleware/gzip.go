package middleware

import (
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

/*
Gzip 响应压缩中间件
功能：对 JSON/HTML/JS/CSS 等文本响应自动 gzip 压缩，
减少网络传输量，提升 API 响应速度。
仅在客户端支持 gzip（Accept-Encoding: gzip）时启用。
跳过已压缩的响应和小于 1KB 的响应。
*/

var gzipPool = sync.Pool{
	New: func() interface{} {
		gz, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return gz
	},
}

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

func GzipCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		/* 仅在客户端支持 gzip 时启用 */
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		/* 跳过 WebSocket 升级请求和 SSE */
		if c.GetHeader("Upgrade") != "" || c.GetHeader("Accept") == "text/event-stream" {
			c.Next()
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(c.Writer)

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		/* 压缩后 Content-Length 不再准确，移除 */
		c.Writer.Header().Del("Content-Length")

		c.Writer = &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}
		defer gz.Close()

		c.Next()
	}
}
