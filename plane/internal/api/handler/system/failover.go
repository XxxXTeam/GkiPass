package system

import (
	"strconv"

	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
)

/*
FailoverHandler 容灾事件 API 处理器
功能：提供容灾事件的查询接口，供前端仪表盘展示：
- 查询隧道容灾历史
- 查询当前活跃容灾状态
- 查询出口组容灾摘要
*/
type FailoverHandler struct {
	app             *types.App
	failoverService *service.FailoverService
}

/*
NewFailoverHandler 创建容灾事件处理器
*/
func NewFailoverHandler(app *types.App) *FailoverHandler {
	return &FailoverHandler{
		app:             app,
		failoverService: service.NewFailoverService(app.DB.GormDB),
	}
}

/*
GetTunnelFailoverHistory 获取隧道容灾历史
功能：按隧道 ID 查询容灾切换/回切事件列表
GET /api/v1/failover/tunnels/:tunnel_id/history?limit=20
*/
func (h *FailoverHandler) GetTunnelFailoverHistory(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")
	if tunnelID == "" {
		response.GinBadRequest(c, "缺少隧道 ID")
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	events, err := h.failoverService.GetTunnelFailoverHistory(tunnelID, limit)
	if err != nil {
		response.GinInternalError(c, "查询容灾历史失败", err)
		return
	}

	response.GinSuccess(c, gin.H{
		"tunnel_id": tunnelID,
		"events":    events,
		"count":     len(events),
	})
}

/*
GetActiveFailovers 获取当前所有活跃容灾状态
功能：返回所有正在容灾中（尚未回切）的隧道列表
GET /api/v1/failover/active
*/
func (h *FailoverHandler) GetActiveFailovers(c *gin.Context) {
	actives := h.failoverService.GetActiveFailovers()
	response.GinSuccess(c, gin.H{
		"active_failovers": actives,
		"count":            len(actives),
	})
}

/*
GetGroupFailoverSummary 获取出口组容灾摘要
功能：返回指定出口组的容灾统计信息
GET /api/v1/failover/groups/:group_id/summary
*/
func (h *FailoverHandler) GetGroupFailoverSummary(c *gin.Context) {
	groupID := c.Param("group_id")
	if groupID == "" {
		response.GinBadRequest(c, "缺少出口组 ID")
		return
	}

	summary := h.failoverService.GetGroupFailoverSummary(groupID)
	response.GinSuccess(c, summary)
}
