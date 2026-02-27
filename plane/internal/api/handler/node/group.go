package node

import (
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
)

// NodeGroupHandler 节点组处理器
type NodeGroupHandler struct {
	app *types.App
}

// NewNodeGroupHandler 创建节点组处理器
func NewNodeGroupHandler(app *types.App) *NodeGroupHandler {
	return &NodeGroupHandler{app: app}
}

// CreateNodeGroupRequest 创建节点组请求
type CreateNodeGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=64"`
	Role        string `json:"role" binding:"omitempty,oneof=ingress egress both"` /* ingress/egress/both */
	Description string `json:"description" binding:"omitempty,max=512"`
}

// Create 创建节点组
func (h *NodeGroupHandler) Create(c *gin.Context) {
	var req CreateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	role := models.NodeRole(req.Role)
	if role == "" {
		role = models.NodeRoleBoth
	}

	group := &models.NodeGroup{
		Name:        req.Name,
		Role:        role,
		Description: req.Description,
	}

	if err := h.app.DAO.CreateNodeGroup(group); err != nil {
		response.GinInternalError(c, "创建节点组失败", err)
		return
	}

	response.GinSuccessWithMessage(c, "节点组已创建", group)
}

// List 列出节点组
func (h *NodeGroupHandler) List(c *gin.Context) {
	role := c.Query("role")

	groups, err := h.app.DAO.ListNodeGroups(role)
	if err != nil {
		response.GinInternalError(c, "获取节点组列表失败", err)
		return
	}

	response.GinSuccess(c, gin.H{
		"groups": groups,
		"total":  len(groups),
	})
}

// Get 获取节点组详情
func (h *NodeGroupHandler) Get(c *gin.Context) {
	id := c.Param("id")

	group, err := h.app.DAO.GetNodeGroupWithNodes(id)
	if err != nil || group == nil {
		response.GinNotFound(c, "节点组不存在")
		return
	}

	response.GinSuccess(c, gin.H{
		"group": group,
		"nodes": group.Nodes,
	})
}

// UpdateNodeGroupRequest 更新节点组请求
type UpdateNodeGroupRequest struct {
	Name        string `json:"name" binding:"omitempty,max=64"`
	Role        string `json:"role" binding:"omitempty,oneof=ingress egress both"`
	Description string `json:"description" binding:"omitempty,max=512"`
}

// Update 更新节点组
func (h *NodeGroupHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	group, err := h.app.DAO.GetNodeGroup(id)
	if err != nil || group == nil {
		response.GinNotFound(c, "节点组不存在")
		return
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Role != "" {
		group.Role = models.NodeRole(req.Role)
	}
	if req.Description != "" {
		group.Description = req.Description
	}

	if err := h.app.DAO.UpdateNodeGroup(group); err != nil {
		response.GinInternalError(c, "更新节点组失败", err)
		return
	}

	response.GinSuccessWithMessage(c, "节点组已更新", group)
}

// Delete 删除节点组
func (h *NodeGroupHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.app.DAO.DeleteNodeGroup(id); err != nil {
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "节点组已删除", nil)
}
