package system

import (
	"strconv"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MonitoringHandler 监控处理器
type MonitoringHandler struct {
	app               *types.App
	monitoringService *service.NodeMonitoringService
}

// NewMonitoringHandler 创建监控处理器
func NewMonitoringHandler(app *types.App) *MonitoringHandler {
	return &MonitoringHandler{
		app:               app,
		monitoringService: service.NewNodeMonitoringService(app.DAO),
	}
}

// ReportNodeMonitoringData 接收节点监控数据上报
func (h *MonitoringHandler) ReportNodeMonitoringData(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		nodeID = c.Query("node_id")
	}

	var data service.NodeMonitoringReportData
	if err := c.ShouldBindJSON(&data); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证节点权限（通过CK或API Key）
	if !h.validateNodeAccess(c, nodeID) {
		response.GinUnauthorized(c, "Invalid node credentials")
		return
	}

	// 处理监控数据
	if err := h.monitoringService.ReportMonitoringData(nodeID, &data); err != nil {
		logger.Error("处理监控数据失败",
			zap.String("nodeID", nodeID),
			zap.Error(err))
		response.InternalError(c, "Failed to process monitoring data")
		return
	}

	response.GinSuccess(c, gin.H{
		"status":    "success",
		"timestamp": time.Now(),
		"message":   "Monitoring data received",
	})
}

// GetNodeMonitoringStatus 获取节点监控状态
func (h *MonitoringHandler) GetNodeMonitoringStatus(c *gin.Context) {
	nodeID := c.Param("id")
	userID := middleware.GetUserID(c)

	// 权限检查
	if !h.monitoringService.CheckMonitoringPermission(userID, nodeID, "view_basic") {
		response.GinForbidden(c, "No permission to view monitoring data")
		return
	}

	status, err := h.monitoringService.GetNodeMonitoringStatus(nodeID)
	if err != nil {
		response.InternalError(c, "Failed to get monitoring status")
		return
	}

	response.GinSuccess(c, status)
}

// GetNodeMonitoringData 获取节点详细监控数据
func (h *MonitoringHandler) GetNodeMonitoringData(c *gin.Context) {
	nodeID := c.Param("id")
	userID := middleware.GetUserID(c)

	// 权限检查
	if !h.monitoringService.CheckMonitoringPermission(userID, nodeID, "view_detailed") {
		response.GinForbidden(c, "No permission to view detailed monitoring data")
		return
	}

	// 解析时间范围
	fromStr := c.DefaultQuery("from", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	toStr := c.DefaultQuery("to", time.Now().Format(time.RFC3339))
	limitStr := c.DefaultQuery("limit", "100")

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

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	// 获取监控数据
	data, err := h.app.DAO.ListNodeMonitoringData(nodeID, from, to, limit)
	if err != nil {
		response.InternalError(c, "Failed to get monitoring data")
		return
	}

	response.GinSuccess(c, gin.H{
		"node_id":    nodeID,
		"from":       from,
		"to":         to,
		"data_count": len(data),
		"data":       data,
	})
}

// GetNodePerformanceHistory 获取节点性能历史
func (h *MonitoringHandler) GetNodePerformanceHistory(c *gin.Context) {
	nodeID := c.Param("id")
	userID := middleware.GetUserID(c)

	// 权限检查
	if !h.monitoringService.CheckMonitoringPermission(userID, nodeID, "view_detailed") {
		response.GinForbidden(c, "No permission to view performance history")
		return
	}

	// 解析参数
	aggregationType := c.DefaultQuery("type", "hourly")
	fromStr := c.DefaultQuery("from", time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	toStr := c.DefaultQuery("to", time.Now().Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		response.GinBadRequest(c, "Invalid from date format")
		return
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		response.GinBadRequest(c, "Invalid to date format")
		return
	}

	// 获取性能历史数据
	history, err := h.app.DAO.GetNodePerformanceHistory(nodeID, aggregationType, from, to)
	if err != nil {
		response.InternalError(c, "Failed to get performance history")
		return
	}

	response.GinSuccess(c, gin.H{
		"node_id":          nodeID,
		"aggregation_type": aggregationType,
		"from":             from,
		"to":               to,
		"data_count":       len(history),
		"data":             history,
	})
}

// GetNodeMonitoringConfig 获取节点监控配置
func (h *MonitoringHandler) GetNodeMonitoringConfig(c *gin.Context) {
	nodeID := c.Param("id")

	/* 权限由 JWT 中间件控制 */
	config, err := h.app.DAO.GetNodeMonitoringConfig(nodeID)
	if err != nil {
		response.InternalError(c, "Failed to get monitoring config")
		return
	}

	// 如果没有配置，返回默认值
	if config == nil {
		config = models.DefaultNodeMonitoringConfig(nodeID)
	}

	response.GinSuccess(c, config)
}

// UpdateNodeMonitoringConfig 更新节点监控配置
func (h *MonitoringHandler) UpdateNodeMonitoringConfig(c *gin.Context) {
	nodeID := c.Param("id")
	userID := middleware.GetUserID(c)

	// 权限检查 - 只有管理员可以更新配置
	if !middleware.IsAdmin(c) {
		response.GinForbidden(c, "Only admin can update monitoring config")
		return
	}

	var req models.NodeMonitoringConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	req.NodeID = nodeID
	if err := h.app.DAO.UpsertNodeMonitoringConfig(&req); err != nil {
		logger.Error("更新监控配置失败",
			zap.String("nodeID", nodeID),
			zap.Error(err))
		response.InternalError(c, "Failed to update monitoring config")
		return
	}

	logger.Info("节点监控配置已更新",
		zap.String("nodeID", nodeID),
		zap.String("updatedBy", userID))

	response.SuccessWithMessage(c, "Monitoring config updated", &req)
}

// ListNodeMonitoringOverview 获取监控概览（管理员）
func (h *MonitoringHandler) ListNodeMonitoringOverview(c *gin.Context) {
	/* 预留：后续可按用户角色筛选节点 */
	nodes, err := h.app.DAO.ListNodes("", "", 1000, 0)
	if err != nil {
		response.InternalError(c, "Failed to get nodes")
		return
	}

	// 获取每个节点的监控状态
	var overview []gin.H
	onlineCount := 0
	totalNodes := len(nodes)
	totalAlerts := 0

	for _, node := range nodes {
		status, err := h.monitoringService.GetNodeMonitoringStatus(node.ID)
		if err != nil {
			continue
		}

		if status.IsOnline {
			onlineCount++
		}
		totalAlerts += status.ActiveAlerts

		nodeInfo := gin.H{
			"node_id":            node.ID,
			"node_name":          node.Name,
			"node_role":          node.Role,
			"is_online":          status.IsOnline,
			"last_seen":          status.LastSeen,
			"has_monitoring":     status.HasData,
			"cpu_usage":          status.CPUUsage,
			"memory_usage":       status.MemoryUsage,
			"disk_usage":         status.DiskUsage,
			"active_connections": status.ActiveConnections,
			"active_tunnels":     status.ActiveTunnels,
			"response_time":      status.ResponseTime,
			"active_alerts":      status.ActiveAlerts,
			"uptime":             status.Uptime,
		}

		overview = append(overview, nodeInfo)
	}

	response.GinSuccess(c, gin.H{
		"summary": gin.H{
			"total_nodes":   totalNodes,
			"online_nodes":  onlineCount,
			"offline_nodes": totalNodes - onlineCount,
			"total_alerts":  totalAlerts,
		},
		"nodes": overview,
	})
}

// GetNodeAlerts 获取节点告警列表
func (h *MonitoringHandler) GetNodeAlerts(c *gin.Context) {
	nodeID := c.Param("id")
	userID := middleware.GetUserID(c)

	// 权限检查
	if !h.monitoringService.CheckMonitoringPermission(userID, nodeID, "view_basic") {
		response.GinForbidden(c, "No permission to view alerts")
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	alerts, err := h.app.DAO.ListNodeAlertHistory(nodeID, limit)
	if err != nil {
		response.InternalError(c, "Failed to get alerts")
		return
	}

	response.GinSuccess(c, gin.H{
		"node_id": nodeID,
		"alerts":  alerts,
		"total":   len(alerts),
	})
}

// CreateAlertRule 创建告警规则（管理员）
func (h *MonitoringHandler) CreateAlertRule(c *gin.Context) {
	nodeID := c.Param("id")

	var req struct {
		RuleName             string  `json:"rule_name" binding:"required"`
		MetricType           string  `json:"metric_type" binding:"required"`
		Operator             string  `json:"operator" binding:"required"`
		ThresholdValue       float64 `json:"threshold_value" binding:"required"`
		DurationSeconds      int     `json:"duration_seconds"`
		Severity             string  `json:"severity" binding:"required"`
		Enabled              bool    `json:"enabled"`
		NotificationChannels string  `json:"notification_channels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	/* 校验指标类型 */
	validMetrics := map[string]bool{"cpu": true, "memory": true, "disk": true, "response_time": true, "connections": true}
	if !validMetrics[req.MetricType] {
		response.GinBadRequest(c, "Invalid metric_type, must be one of: cpu, memory, disk, response_time, connections")
		return
	}

	/* 校验操作符 */
	validOps := map[string]bool{">": true, "<": true, ">=": true, "<=": true, "=": true, "!=": true}
	if !validOps[req.Operator] {
		response.GinBadRequest(c, "Invalid operator")
		return
	}

	/* 校验严重级别 */
	validSeverity := map[string]bool{"info": true, "warning": true, "critical": true}
	if !validSeverity[req.Severity] {
		response.GinBadRequest(c, "Invalid severity, must be one of: info, warning, critical")
		return
	}

	/* 校验节点存在 */
	node, err := h.app.DAO.GetNode(nodeID)
	if err != nil || node == nil {
		response.GinNotFound(c, "Node not found")
		return
	}

	rule := &models.NodeAlertRule{
		NodeID:               nodeID,
		RuleName:             req.RuleName,
		MetricType:           req.MetricType,
		Operator:             req.Operator,
		ThresholdValue:       req.ThresholdValue,
		DurationSeconds:      req.DurationSeconds,
		Severity:             req.Severity,
		Enabled:              req.Enabled,
		NotificationChannels: req.NotificationChannels,
	}

	if err := h.app.DAO.CreateNodeAlertRule(rule); err != nil {
		logger.Error("创建告警规则失败", zap.Error(err))
		response.InternalError(c, "Failed to create alert rule")
		return
	}

	response.SuccessWithMessage(c, "Alert rule created", rule)
}

// ListAlertRules 列出节点告警规则
func (h *MonitoringHandler) ListAlertRules(c *gin.Context) {
	nodeID := c.Param("id")

	rules, err := h.app.DAO.ListNodeAlertRules(nodeID)
	if err != nil {
		response.InternalError(c, "Failed to list alert rules")
		return
	}

	response.GinSuccess(c, gin.H{
		"node_id": nodeID,
		"rules":   rules,
		"total":   len(rules),
	})
}

// UpdateAlertRule 更新告警规则（管理员）
func (h *MonitoringHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("rule_id")

	rule, err := h.app.DAO.GetNodeAlertRule(ruleID)
	if err != nil || rule == nil {
		response.GinNotFound(c, "Alert rule not found")
		return
	}

	var req struct {
		RuleName             *string  `json:"rule_name"`
		MetricType           *string  `json:"metric_type"`
		Operator             *string  `json:"operator"`
		ThresholdValue       *float64 `json:"threshold_value"`
		DurationSeconds      *int     `json:"duration_seconds"`
		Severity             *string  `json:"severity"`
		Enabled              *bool    `json:"enabled"`
		NotificationChannels *string  `json:"notification_channels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.RuleName != nil {
		rule.RuleName = *req.RuleName
	}
	if req.MetricType != nil {
		rule.MetricType = *req.MetricType
	}
	if req.Operator != nil {
		rule.Operator = *req.Operator
	}
	if req.ThresholdValue != nil {
		rule.ThresholdValue = *req.ThresholdValue
	}
	if req.DurationSeconds != nil {
		rule.DurationSeconds = *req.DurationSeconds
	}
	if req.Severity != nil {
		rule.Severity = *req.Severity
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.NotificationChannels != nil {
		rule.NotificationChannels = *req.NotificationChannels
	}

	if err := h.app.DAO.UpdateNodeAlertRule(rule); err != nil {
		logger.Error("更新告警规则失败", zap.Error(err))
		response.InternalError(c, "Failed to update alert rule")
		return
	}

	response.SuccessWithMessage(c, "Alert rule updated", rule)
}

// DeleteAlertRule 删除告警规则（管理员）
func (h *MonitoringHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("rule_id")

	rule, _ := h.app.DAO.GetNodeAlertRule(ruleID)
	if rule == nil {
		response.GinNotFound(c, "Alert rule not found")
		return
	}

	if err := h.app.DAO.DeleteNodeAlertRule(ruleID); err != nil {
		logger.Error("删除告警规则失败", zap.Error(err))
		response.InternalError(c, "Failed to delete alert rule")
		return
	}

	response.SuccessWithMessage(c, "Alert rule deleted", nil)
}

// AcknowledgeAlert 确认告警（管理员）
func (h *MonitoringHandler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("alert_id")
	userID := middleware.GetUserID(c)

	if err := h.app.DAO.UpdateNodeAlertHistoryStatus(alertID, "acknowledged", userID); err != nil {
		logger.Error("确认告警失败", zap.Error(err))
		response.InternalError(c, "Failed to acknowledge alert")
		return
	}

	response.SuccessWithMessage(c, "Alert acknowledged", nil)
}

// ResolveAlert 解决告警（管理员）
func (h *MonitoringHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("alert_id")

	if err := h.app.DAO.UpdateNodeAlertHistoryStatus(alertID, "resolved", ""); err != nil {
		logger.Error("解决告警失败", zap.Error(err))
		response.InternalError(c, "Failed to resolve alert")
		return
	}

	response.SuccessWithMessage(c, "Alert resolved", nil)
}

// CreateMonitoringPermission 创建监控权限（管理员）
func (h *MonitoringHandler) CreateMonitoringPermission(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		TargetUserID   string `json:"user_id" binding:"required"`
		NodeID         string `json:"node_id"` // 可为空表示全局配置
		PermissionType string `json:"permission_type" binding:"required"`
		Enabled        bool   `json:"enabled"`
		Description    string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证权限类型
	validTypes := map[string]bool{
		"view_basic": true, "view_detailed": true,
		"view_system": true, "view_network": true, "disabled": true,
	}
	if !validTypes[req.PermissionType] {
		response.GinBadRequest(c, "Invalid permission type")
		return
	}

	// 如果指定了节点，验证节点是否存在
	if req.NodeID != "" {
		node, err := h.app.DAO.GetNode(req.NodeID)
		if err != nil || node == nil {
			response.GinNotFound(c, "节点不存在")
			return
		}
	}

	targetUser, err := h.app.DAO.GetUser(req.TargetUserID)
	if err != nil || targetUser == nil {
		response.GinNotFound(c, "目标用户不存在")
		return
	}

	permission := &models.MonitoringPermission{
		UserID:         req.TargetUserID,
		NodeID:         req.NodeID,
		PermissionType: req.PermissionType,
		Enabled:        req.Enabled,
		CreatedBy:      userID,
		Description:    req.Description,
	}

	if err := h.app.DAO.CreateMonitoringPermission(permission); err != nil {
		logger.Error("创建监控权限失败", zap.Error(err))
		response.InternalError(c, "Failed to create monitoring permission")
		return
	}

	logger.Info("监控权限已创建",
		zap.String("targetUserID", req.TargetUserID),
		zap.String("nodeID", req.NodeID),
		zap.String("permissionType", req.PermissionType),
		zap.String("createdBy", userID))

	response.SuccessWithMessage(c, "Monitoring permission created", permission)
}

// ListMonitoringPermissions 列出监控权限（管理员）
func (h *MonitoringHandler) ListMonitoringPermissions(c *gin.Context) {
	targetUserID := c.Query("user_id")

	permissions, err := h.app.DAO.ListMonitoringPermissions(targetUserID)
	if err != nil {
		response.GinInternalError(c, "获取监控权限列表失败", err)
		return
	}

	response.GinSuccess(c, gin.H{
		"permissions": permissions,
		"total":       len(permissions),
	})
}

// GetMyMonitoringPermissions 获取我的监控权限
func (h *MonitoringHandler) GetMyMonitoringPermissions(c *gin.Context) {
	userID := middleware.GetUserID(c)

	permissions, err := h.app.DAO.ListMonitoringPermissions(userID)
	if err != nil {
		response.InternalError(c, "Failed to get monitoring permissions")
		return
	}

	response.GinSuccess(c, gin.H{
		"permissions": permissions,
		"total":       len(permissions),
	})
}

// validateNodeAccess 验证节点访问权限（用于数据上报）
func (h *MonitoringHandler) validateNodeAccess(c *gin.Context, nodeID string) bool {
	// 从Header获取认证信息
	apiKey := c.GetHeader("X-API-Key")
	connectionKey := c.GetHeader("X-Connection-Key")

	if apiKey != "" {
		// 通过API Key验证
		node, err := h.app.DAO.GetNodeByAPIKey(apiKey)
		return err == nil && node != nil && node.ID == nodeID
	}

	if connectionKey != "" {
		// 通过Connection Key验证
		ck, err := h.app.DAO.GetCKByKey(connectionKey)
		return err == nil && ck != nil && ck.NodeID == nodeID && ck.Type == "node"
	}

	return false
}

// NodeMonitoringSummary 节点监控汇总（用于Dashboard）
func (h *MonitoringHandler) NodeMonitoringSummary(c *gin.Context) {
	/* 预留：后续可按用户角色筛选节点 */
	nodes, err := h.app.DAO.ListNodes("", "", 1000, 0)
	if err != nil {
		response.InternalError(c, "Failed to get nodes")
		return
	}

	// 统计数据
	var summary struct {
		TotalNodes       int     `json:"total_nodes"`
		OnlineNodes      int     `json:"online_nodes"`
		MonitoredNodes   int     `json:"monitored_nodes"`
		TotalAlerts      int     `json:"total_alerts"`
		AvgCPUUsage      float64 `json:"avg_cpu_usage"`
		AvgMemoryUsage   float64 `json:"avg_memory_usage"`
		TotalConnections int     `json:"total_connections"`
		TotalTraffic     int64   `json:"total_traffic"`
	}

	summary.TotalNodes = len(nodes)
	var totalCPU, totalMemory float64
	var monitoredCount int

	for _, node := range nodes {
		status, err := h.monitoringService.GetNodeMonitoringStatus(node.ID)
		if err != nil {
			continue
		}

		if status.IsOnline {
			summary.OnlineNodes++
		}

		if status.HasData {
			summary.MonitoredNodes++
			monitoredCount++
			totalCPU += status.CPUUsage
			totalMemory += status.MemoryUsage
			summary.TotalConnections += status.ActiveConnections
			summary.TotalTraffic += status.TrafficIn + status.TrafficOut
		}

		summary.TotalAlerts += status.ActiveAlerts
	}

	if monitoredCount > 0 {
		summary.AvgCPUUsage = totalCPU / float64(monitoredCount)
		summary.AvgMemoryUsage = totalMemory / float64(monitoredCount)
	}

	response.GinSuccess(c, summary)
}
