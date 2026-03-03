package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

/*
Prometheus HTTP 请求指标
功能：自动采集每个 API 请求的耗时、状态码和方法分布，
通过 /metrics 端点暴露给 Prometheus 抓取。
*/
var (
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求耗时（秒）",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "path", "status"},
	)
)

/*
Metrics Prometheus 指标中间件
功能：记录每个请求的耗时和计数，按 method/path/status 维度分组。
路径规范化：将 UUID 参数替换为 :id 防止基数爆炸。
*/
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		httpRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(duration)
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
	}
}
