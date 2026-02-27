package tunnel

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TrafficStatsHandler 流量统计处理器
type TrafficStatsHandler struct {
	app *types.App
}

// NewTrafficStatsHandler 创建流量统计处理器
func NewTrafficStatsHandler(app *types.App) *TrafficStatsHandler {
	return &TrafficStatsHandler{app: app}
}

// ListTrafficStatsResponse 流量统计列表响应
type ListTrafficStatsResponse struct {
	Data  []*TrafficStatWithDetails `json:"data"`
	Total int                       `json:"total"`
}

// TrafficStatWithDetails 带详情的流量统计
type TrafficStatWithDetails struct {
	models.TrafficStats
	TunnelName     string `json:"tunnel_name"`
	EntryGroupName string `json:"entry_group_name"`
	ExitGroupName  string `json:"exit_group_name"`
}

// ListTrafficStats 列出流量统计
func (h *TrafficStatsHandler) ListTrafficStats(c *gin.Context) {
	// 管理员可以查看所有用户的数据
	queryUserID := middleware.GetUserID(c)
	if middleware.IsAdmin(c) && c.Query("user_id") != "" {
		queryUserID = c.Query("user_id")
	}

	tunnelID := c.Query("tunnel_id")
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if val, err := intParse(p); err == nil {
			page = val
		}
	}
	if l := c.Query("limit"); l != "" {
		if val, err := intParse(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}

	offset := (page - 1) * limit

	stats, total64, err := h.app.DAO.ListTrafficStats(queryUserID, tunnelID, limit, offset)
	if err != nil {
		logger.Error("查询流量统计失败", zap.Error(err))
		response.InternalError(c, "Failed to list traffic stats")
		return
	}

	result := make([]*TrafficStatWithDetails, 0, len(stats))
	for _, stat := range stats {
		detail := &TrafficStatWithDetails{
			TrafficStats: stat,
		}

		if tun, _ := h.app.DAO.GetTunnel(stat.TunnelID); tun != nil {
			detail.TunnelName = tun.Name
		}

		result = append(result, detail)
	}

	response.GinSuccess(c, ListTrafficStatsResponse{
		Data:  result,
		Total: int(total64),
	})
}

// GetTrafficSummary 获取流量汇总
func (h *TrafficStatsHandler) GetTrafficSummary(c *gin.Context) {
	// 管理员可以查看所有用户的数据
	queryUserID := middleware.GetUserID(c)
	if middleware.IsAdmin(c) && c.Query("user_id") != "" {
		queryUserID = c.Query("user_id")
	}

	tunnelID := c.Query("tunnel_id")

	// 解析日期范围
	startDateStr := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		response.GinBadRequest(c, "Invalid start_date format")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		response.GinBadRequest(c, "Invalid end_date format")
		return
	}

	trafficIn, trafficOut, err := h.app.DAO.GetTrafficSummary(
		queryUserID, tunnelID, startDate, endDate)
	if err != nil {
		logger.Error("获取流量汇总失败", zap.Error(err))
		response.InternalError(c, "Failed to get traffic summary")
		return
	}

	response.GinSuccess(c, gin.H{
		"traffic_in":    trafficIn,
		"traffic_out":   trafficOut,
		"total_traffic": trafficIn + trafficOut,
		"start_date":    startDate,
		"end_date":      endDate,
	})
}

// ReportTrafficRequest 上报流量请求
type ReportTrafficRequest struct {
	TunnelID   string `json:"tunnel_id" binding:"required"`
	TrafficIn  int64  `json:"traffic_in"`
	TrafficOut int64  `json:"traffic_out"`
}

// ReportTraffic 上报流量（节点使用）
func (h *TrafficStatsHandler) ReportTraffic(c *gin.Context) {
	var req ReportTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tunnel, err := h.app.DAO.GetTunnel(req.TunnelID)
	if err != nil || tunnel == nil {
		response.GinNotFound(c, "Tunnel not found")
		return
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	stat := &models.TrafficStats{
		NodeID:    "",
		TunnelID:  req.TunnelID,
		UserID:    tunnel.CreatedBy,
		BytesIn:   req.TrafficIn,
		BytesOut:  req.TrafficOut,
		Period:    "daily",
		PeriodKey: today.Format("2006-01-02"),
		StartAt:   today,
		EndAt:     today.Add(24 * time.Hour),
	}
	stat.ID = uuid.New().String()

	if err := h.app.DAO.CreateTrafficStats(stat); err != nil {
		logger.Error("创建流量统计失败", zap.Error(err))
		response.InternalError(c, "Failed to create traffic stat")
		return
	}

	response.SuccessWithMessage(c, "Traffic reported successfully", nil)
}

// intParse 解析整数
func intParse(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
