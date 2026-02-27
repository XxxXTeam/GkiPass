package user

import (
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StatisticsHandler 统计处理器
type StatisticsHandler struct {
	app *types.App
}

// NewStatisticsHandler 创建统计处理器
func NewStatisticsHandler(app *types.App) *StatisticsHandler {
	return &StatisticsHandler{app: app}
}

// GetNodeStats 获取节点统计
func (h *StatisticsHandler) GetNodeStats(c *gin.Context) {
	nodeID := c.Param("id")
	fromStr := c.DefaultQuery("from", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	toStr := c.DefaultQuery("to", time.Now().Format(time.RFC3339))

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		response.GinBadRequest(c, "Invalid from time format")
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		response.GinBadRequest(c, "Invalid to time format")
		return
	}

	metrics, err := h.app.DAO.ListNodeMetrics(nodeID, from, to, 1000)
	if err != nil {
		logger.Error("获取节点统计失败", zap.String("nodeID", nodeID), zap.Error(err))
		response.InternalError(c, "Failed to get statistics")
		return
	}

	/* 计算汇总数据 */
	var totalBytesIn, totalBytesOut int64
	var avgCPU, avgMem float64

	if len(metrics) > 0 {
		for _, m := range metrics {
			totalBytesIn += m.NetworkIn
			totalBytesOut += m.NetworkOut
			avgCPU += m.CPUUsage
			avgMem += m.MemoryUsage
		}
		avgCPU /= float64(len(metrics))
		avgMem /= float64(len(metrics))
	}

	response.GinSuccess(c, gin.H{
		"node_id": nodeID,
		"from":    from,
		"to":      to,
		"summary": gin.H{
			"total_bytes_in":  totalBytesIn,
			"total_bytes_out": totalBytesOut,
			"avg_cpu":         avgCPU,
			"avg_memory":      avgMem,
		},
		"data": metrics,
	})
}

// GetOverview 获取总览统计
func (h *StatisticsHandler) GetOverview(c *gin.Context) {
	nodes, err := h.app.DAO.ListNodes("", "", 1000, 0)
	if err != nil {
		logger.Error("获取节点列表失败", zap.Error(err))
		response.InternalError(c, "Failed to get nodes")
		return
	}

	var onlineCount, offlineCount, errorCount int
	for _, n := range nodes {
		switch n.Status {
		case models.NodeStatusOnline:
			onlineCount++
		case models.NodeStatusOffline:
			offlineCount++
		case models.NodeStatusError:
			errorCount++
		}
	}

	/* 获取实时流量（Redis 可选） */
	var totalTrafficIn, totalTrafficOut int64
	if h.app.DB.HasCache() {
		for _, n := range nodes {
			bytesIn, bytesOut, _ := h.app.DB.Cache.Redis.GetTraffic(n.ID)
			totalTrafficIn += bytesIn
			totalTrafficOut += bytesOut
		}
	}

	policies, _ := h.app.DAO.ListPolicies("", nil)
	enabledPolicies := 0
	for _, p := range policies {
		if p.Enabled {
			enabledPolicies++
		}
	}

	certs, _ := h.app.DAO.ListCertificates("", nil)
	activeCerts := 0
	for _, cert := range certs {
		if !cert.Revoked && time.Now().Before(cert.NotAfter) {
			activeCerts++
		}
	}

	response.GinSuccess(c, gin.H{
		"nodes": gin.H{
			"total":   len(nodes),
			"online":  onlineCount,
			"offline": offlineCount,
			"error":   errorCount,
		},
		"policies": gin.H{
			"total":   len(policies),
			"enabled": enabledPolicies,
		},
		"certificates": gin.H{
			"total":  len(certs),
			"active": activeCerts,
		},
		"traffic": gin.H{
			"total_in":  totalTrafficIn,
			"total_out": totalTrafficOut,
		},
	})
}

// GetAdminOverview 获取管理员总览统计
func (h *StatisticsHandler) GetAdminOverview(c *gin.Context) {
	totalUsers, err := h.app.DAO.GetUserCount()
	if err != nil {
		logger.Error("获取用户数失败", zap.Error(err))
		response.InternalError(c, "Failed to get users count")
		return
	}

	nodes, err := h.app.DAO.ListNodes("", "", 10000, 0)
	if err != nil {
		logger.Error("获取节点列表失败", zap.Error(err))
		response.InternalError(c, "Failed to get nodes")
		return
	}

	/* 隧道计数 */
	var totalTunnels int64
	h.app.DAO.DB.Model(&models.Tunnel{}).Count(&totalTunnels)

	activeSubscriptions, _ := h.app.DAO.CountActiveSubscriptions()

	response.GinSuccess(c, gin.H{
		"total_users":         totalUsers,
		"total_nodes":         len(nodes),
		"total_tunnels":       totalTunnels,
		"total_subscriptions": activeSubscriptions,
	})
}

// ReportStatsRequest 上报统计请求
type ReportStatsRequest struct {
	NodeID         string  `json:"node_id" binding:"required"`
	BytesIn        int64   `json:"bytes_in"`
	BytesOut       int64   `json:"bytes_out"`
	PacketsIn      int64   `json:"packets_in"`
	PacketsOut     int64   `json:"packets_out"`
	Connections    int     `json:"connections"`
	ActiveSessions int     `json:"active_sessions"`
	ErrorCount     int     `json:"error_count"`
	AvgLatency     float64 `json:"avg_latency"`
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    int64   `json:"memory_usage"`
}

// ReportStats 节点上报统计
func (h *StatisticsHandler) ReportStats(c *gin.Context) {
	var req ReportStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	metric := &models.NodeMetrics{}
	metric.ID = uuid.New().String()
	metric.NodeID = req.NodeID
	metric.CPUUsage = req.CPUUsage
	metric.MemoryUsage = float64(req.MemoryUsage)
	metric.NetworkIn = req.BytesIn
	metric.NetworkOut = req.BytesOut
	metric.Connections = req.Connections

	if err := h.app.DAO.CreateNodeMetrics(metric); err != nil {
		logger.Error("保存统计数据失败", zap.Error(err))
		response.InternalError(c, "Failed to save statistics")
		return
	}

	/* 更新 Redis 实时流量 */
	if h.app.DB.HasCache() {
		_ = h.app.DB.Cache.Redis.IncrementTraffic(req.NodeID, req.BytesIn, req.BytesOut)
	}

	response.SuccessWithMessage(c, "Statistics reported successfully", nil)
}
