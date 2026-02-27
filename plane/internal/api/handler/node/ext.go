package node

import (
	"encoding/json"

	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetAvailableNodes 获取用户可用的节点（基于套餐）
func (h *NodeHandler) GetAvailableNodes(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// 管理员可以看到所有节点
	if middleware.IsAdmin(c) {
		nodes, err := h.app.DAO.ListNodes("", "", 1000, 0)
		if err != nil {
			response.InternalError(c, "Failed to get nodes")
			return
		}
		response.GinSuccess(c, nodes)
		return
	}

	// 普通用户：检查订阅状态
	sub, err := h.app.DAO.GetActiveSubscription(userID)
	if err != nil {
		response.InternalError(c, "Failed to check subscription")
		return
	}

	// 如果没有订阅，返回空列表
	if sub == nil {
		response.GinSuccess(c, gin.H{
			"nodes":            []interface{}{},
			"has_subscription": false,
			"message":          "请先购买套餐以查看可用节点",
		})
		return
	}

	// 获取套餐信息
	plan, err := h.app.DAO.GetPlan(sub.PlanID)
	if err != nil || plan == nil {
		response.GinSuccess(c, gin.H{
			"nodes":            []interface{}{},
			"has_subscription": false,
			"message":          "请先购买套餐以查看可用节点",
		})
		return
	}

	// 获取所有节点
	allNodes, err := h.app.DAO.ListNodes("", "", 1000, 0)
	if err != nil {
		response.InternalError(c, "Failed to get nodes")
		return
	}

	// 如果套餐的allowed_node_ids为空或"[]"，则允许使用所有节点
	/* NodeGroupIDs 用于限制可用节点组，为空则允许所有 */
	allowedGroupIDsStr := plan.NodeGroupIDs

	if allowedGroupIDsStr == "" || allowedGroupIDsStr == "[]" {
		response.GinSuccess(c, gin.H{
			"nodes":            allNodes,
			"has_subscription": true,
			"plan_name":        plan.Name,
		})
		return
	}

	/* 解析允许的节点组 ID 列表 */
	var allowedGroupIDs []string
	if err := json.Unmarshal([]byte(allowedGroupIDsStr), &allowedGroupIDs); err != nil {
		logger.Error("解析套餐节点组ID失败", zap.Error(err))
		response.InternalError(c, "Failed to parse plan node group IDs")
		return
	}

	allowedGroupsMap := make(map[string]bool)
	for _, id := range allowedGroupIDs {
		allowedGroupsMap[id] = true
	}

	/* 过滤节点：属于允许节点组的节点 */
	var availableNodes []interface{}
	for _, nd := range allNodes {
		for _, g := range nd.Groups {
			if allowedGroupsMap[g.ID] {
				availableNodes = append(availableNodes, nd)
				break
			}
		}
	}

	response.GinSuccess(c, gin.H{
		"nodes":            availableNodes,
		"has_subscription": true,
		"plan_name":        plan.Name,
		"total_allowed":    len(allowedGroupIDs),
		"available_count":  len(availableNodes),
	})
}
