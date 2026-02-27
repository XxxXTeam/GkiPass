package tunnel

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"
)

/*
GinTunnelHandler 基于 Gin 框架的隧道管理 API 处理器
功能：提供隧道的 CRUD 操作、启用/禁用切换的 HTTP API
使用 GormTunnelService 作为数据访问层
*/
type GinTunnelHandler struct {
	app       *types.App
	tunnelSvc *service.GormTunnelService
	logger    *zap.Logger
}

/*
NewGinTunnelHandler 创建 Gin 隧道处理器
*/
func NewGinTunnelHandler(app *types.App) *GinTunnelHandler {
	return &GinTunnelHandler{
		app:       app,
		tunnelSvc: service.NewGormTunnelService(app.DB.GormDB),
		logger:    zap.L().Named("gin-tunnel-handler"),
	}
}

/*
List 列出所有隧道
功能：获取当前用户的隧道列表
路由：GET /api/v1/tunnels
*/
func (h *GinTunnelHandler) List(c *gin.Context) {
	/* 管理员可查看所有隧道，普通用户只看自己的 */
	filterUserID := ""
	if !middleware.IsAdmin(c) {
		filterUserID = middleware.GetUserID(c)
	}

	tunnels, err := h.tunnelSvc.ListTunnels(filterUserID, false)
	if err != nil {
		h.logger.Error("列出隧道失败", zap.Error(err))
		response.GinInternalError(c, "获取隧道列表失败", err)
		return
	}

	response.GinSuccess(c, tunnels)
}

/*
Get 获取单个隧道
功能：根据 ID 获取隧道详细信息
路由：GET /api/v1/tunnels/:id
*/
func (h *GinTunnelHandler) Get(c *gin.Context) {
	id := c.Param("id")

	tunnel, err := h.tunnelSvc.GetTunnel(id)
	if err != nil {
		h.logger.Error("获取隧道失败", zap.String("id", id), zap.Error(err))
		response.GinNotFound(c, "隧道不存在")
		return
	}

	response.GinSuccess(c, tunnel)
}

/*
Create 创建隧道
功能：创建新的转发隧道及其关联规则
路由：POST /api/v1/tunnels
*/
func (h *GinTunnelHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.GinUnauthorized(c, "未认证")
		return
	}

	var req service.CreateTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "无效的请求数据: "+err.Error())
		return
	}

	tunnel, err := h.tunnelSvc.CreateTunnel(&req, userID)
	if err != nil {
		h.logger.Error("创建隧道失败", zap.String("name", req.Name), zap.Error(err))
		response.GinInternalError(c, "创建隧道失败", err)
		return
	}

	response.GinSuccess(c, tunnel)
}

/*
Update 更新隧道
功能：更新已有隧道的配置信息
路由：POST /api/v1/tunnels/:id/update
*/
func (h *GinTunnelHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.CreateTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "无效的请求数据: "+err.Error())
		return
	}

	tunnel, err := h.tunnelSvc.UpdateTunnel(id, &req)
	if err != nil {
		h.logger.Error("更新隧道失败", zap.String("id", id), zap.Error(err))
		response.GinInternalError(c, "更新隧道失败", err)
		return
	}

	response.GinSuccess(c, tunnel)
}

/*
Delete 删除隧道
功能：删除隧道及其关联的规则
路由：POST /api/v1/tunnels/:id/delete
*/
func (h *GinTunnelHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.tunnelSvc.DeleteTunnel(id); err != nil {
		h.logger.Error("删除隧道失败", zap.String("id", id), zap.Error(err))
		response.GinInternalError(c, "删除隧道失败", err)
		return
	}

	response.GinSuccessWithMessage(c, "隧道已删除", nil)
}

/*
Toggle 切换隧道启用/禁用状态
功能：快速切换隧道的启用状态，同步更新关联规则
路由：POST /api/v1/tunnels/:id/toggle
*/
func (h *GinTunnelHandler) Toggle(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "无效的请求数据")
		return
	}

	if _, err := h.tunnelSvc.ToggleTunnel(id, req.Enabled); err != nil {
		h.logger.Error("切换隧道状态失败", zap.String("id", id), zap.Error(err))
		response.GinInternalError(c, "切换隧道状态失败", err)
		return
	}

	/* 重新获取更新后的隧道数据 */
	tunnel, _ := h.tunnelSvc.GetTunnel(id)

	response.GinSuccess(c, tunnel)
}
