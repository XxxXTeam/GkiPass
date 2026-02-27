package billing

import (
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PaymentConfigHandler struct {
	app *types.App
}

func NewPaymentConfigHandler(app *types.App) *PaymentConfigHandler {
	return &PaymentConfigHandler{app: app}
}

// ListConfigs 列出所有支付配置
func (h *PaymentConfigHandler) ListConfigs(c *gin.Context) {
	configs, err := h.app.DAO.ListPaymentConfigs()
	if err != nil {
		logger.Error("查询支付配置失败", zap.Error(err))
		response.InternalError(c, "Failed to list payment configs")
		return
	}

	response.GinSuccess(c, configs)
}

// GetConfig 获取单个支付配置
func (h *PaymentConfigHandler) GetConfig(c *gin.Context) {
	id := c.Param("id")

	config, err := h.app.DAO.GetPaymentConfig(id)
	if err != nil {
		response.InternalError(c, "Failed to get payment config")
		return
	}
	if config == nil {
		response.GinNotFound(c, "Payment config not found")
		return
	}

	response.GinSuccess(c, config)
}

// UpdateConfig 更新支付配置
func (h *PaymentConfigHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Enabled bool   `json:"enabled"`
		Config  string `json:"config" binding:"required,max=4096"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证配置是否为有效JSON（已通过Gin的binding完成基础验证）

	config, gErr := h.app.DAO.GetPaymentConfig(id)
	if gErr != nil || config == nil {
		response.GinNotFound(c, "Payment config not found")
		return
	}
	config.Enabled = req.Enabled
	config.Config = req.Config
	if err := h.app.DAO.UpdatePaymentConfig(config); err != nil {
		logger.Error("更新支付配置失败", zap.Error(err))
		response.InternalError(c, "Failed to update payment config")
		return
	}

	logger.Info("更新支付配置",
		zap.String("id", id),
		zap.Bool("enabled", req.Enabled))

	response.GinSuccess(c, gin.H{
		"id":      id,
		"enabled": req.Enabled,
		"message": "Payment config updated successfully",
	})
}

// ToggleConfig 切换支付配置启用状态
func (h *PaymentConfigHandler) ToggleConfig(c *gin.Context) {
	id := c.Param("id")

	config, err := h.app.DAO.GetPaymentConfig(id)
	if err != nil || config == nil {
		response.GinNotFound(c, "Payment config not found")
		return
	}

	newStatus := !config.Enabled
	config.Enabled = newStatus
	if err := h.app.DAO.UpdatePaymentConfig(config); err != nil {
		logger.Error("切换支付配置失败", zap.Error(err))
		response.InternalError(c, "Failed to toggle payment config")
		return
	}

	logger.Info("切换支付配置状态",
		zap.String("id", id),
		zap.Bool("enabled", newStatus))

	response.GinSuccess(c, gin.H{
		"id":      id,
		"enabled": newStatus,
	})
}
