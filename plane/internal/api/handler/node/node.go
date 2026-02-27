package node

import (
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/auth"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	app *types.App
}

// NewNodeHandler 创建节点处理器
func NewNodeHandler(app *types.App) *NodeHandler {
	return &NodeHandler{app: app}
}

// CreateNodeRequest 创建节点请求
type CreateNodeRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=64"`
	Role        string `json:"role" binding:"omitempty,oneof=ingress egress both"` /* ingress/egress/both */
	PublicIP    string `json:"public_ip" binding:"omitempty,max=256"`
	Port        int    `json:"port" binding:"omitempty,min=1,max=65535"`
	GroupID     string `json:"group_id" binding:"omitempty,max=36"` /* 可选：节点组ID */
	Description string `json:"description" binding:"omitempty,max=512"`
}

// Create 创建节点
func (h *NodeHandler) Create(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	/* 如果指定了组，验证组是否存在 */
	if req.GroupID != "" {
		group, err := h.app.DAO.GetNodeGroup(req.GroupID)
		if err != nil || group == nil {
			response.GinBadRequest(c, "Node group not found")
			return
		}
	}

	role := models.NodeRole(req.Role)
	if role == "" {
		role = models.NodeRoleBoth
	}

	node := &models.Node{
		Name:        req.Name,
		Description: req.Description,
		Status:      models.NodeStatusOffline,
		Role:        role,
		PublicIP:    req.PublicIP,
		Port:        req.Port,
	}

	if err := h.app.DAO.CreateNode(node); err != nil {
		logger.Error("创建节点失败", zap.Error(err))
		response.GinInternalError(c, "创建节点失败", err)
		return
	}

	/* 如果指定了组，将节点加入组 */
	if req.GroupID != "" {
		_ = h.app.DAO.AddNodeToGroup(node.ID, req.GroupID)
	}

	/* 自动为节点生成 Connection Key（30天有效） */
	ck, ckErr := auth.CreateNodeCK(node.ID, 30*24*time.Hour)
	if ckErr != nil {
		logger.Error("生成CK失败", zap.Error(ckErr))
	} else if err := h.app.DAO.CreateConnectionKey(ck); err != nil {
		logger.Error("保存CK失败", zap.Error(err))
	}

	logger.Info("节点已创建",
		zap.String("nodeID", node.ID),
		zap.String("name", node.Name))

	respData := gin.H{
		"node": node,
	}
	if ck != nil {
		respData["connection_key"] = ck.Key
		respData["usage"] = "节点启动命令: ./client --token " + ck.Key
		respData["expires_at"] = ck.ExpiresAt
	}
	response.SuccessWithMessage(c, "Node created successfully", respData)
}

// ListNodesRequest 列出节点请求
type ListNodesRequest struct {
	Type   string `form:"type"`
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// List 列出节点
func (h *NodeHandler) List(c *gin.Context) {
	var req ListNodesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 设置默认值
	if req.Limit == 0 {
		req.Limit = 50
	}

	/* 预留：后续可按用户角色筛选节点 */
	nodes, err := h.app.DAO.ListNodes("", req.Status, req.Limit, req.Offset)
	if err != nil {
		response.GinInternalError(c, "获取节点列表失败", err)
		return
	}

	response.GinSuccess(c, gin.H{
		"nodes": nodes,
		"total": len(nodes),
	})
}

// Get 获取节点详情
func (h *NodeHandler) Get(c *gin.Context) {
	id := c.Param("id")

	node, err := h.app.DAO.GetNode(id)
	if err != nil {
		response.GinInternalError(c, "获取节点失败", err)
		return
	}

	if node == nil {
		response.GinNotFound(c, "节点不存在")
		return
	}

	response.GinSuccess(c, node)
}

// UpdateNodeRequest 更新节点请求
type UpdateNodeRequest struct {
	Name        string `json:"name" binding:"omitempty,max=64"`
	IP          string `json:"ip" binding:"omitempty,max=256"`
	Port        int    `json:"port" binding:"omitempty,min=1,max=65535"`
	GroupID     string `json:"group_id" binding:"omitempty,max=36"`
	Status      string `json:"status" binding:"omitempty,oneof=online offline error"`
	Description string `json:"description" binding:"omitempty,max=512"`
}

// Update 更新节点
func (h *NodeHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	node, err := h.app.DAO.GetNode(id)
	if err != nil || node == nil {
		response.GinNotFound(c, "节点不存在")
		return
	}

	/* 更新字段 */
	if req.Name != "" {
		node.Name = req.Name
	}
	if req.Description != "" {
		node.Description = req.Description
	}
	if req.Port > 0 {
		node.Port = req.Port
	}
	if req.Status != "" {
		node.Status = models.NodeStatus(req.Status)
	}

	if err := h.app.DAO.UpdateNode(node); err != nil {
		response.GinInternalError(c, "更新节点失败", err)
		return
	}

	response.GinSuccessWithMessage(c, "节点已更新", node)
}

// Delete 删除节点
func (h *NodeHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.app.DAO.DeleteNode(id); err != nil {
		logger.Error("删除节点失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to delete node")
		return
	}

	response.GinSuccessWithMessage(c, "节点已删除", nil)
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	Load        float64 `json:"load"`
	Connections int     `json:"connections"`
}

// Heartbeat 节点心跳
func (h *NodeHandler) Heartbeat(c *gin.Context) {
	id := c.Param("id")

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	/* 更新节点状态 */
	_ = h.app.DAO.UpdateNodeStatus(id, models.NodeStatusOnline)

	response.GinSuccess(c, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}
