package tunnel

import (
	"encoding/json"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

/*
PolicyConfig 策略配置详情（用于解析 JSON）
*/
type PolicyConfig struct {
	Protocols  []string `json:"protocols,omitempty"`
	AllowIPs   []string `json:"allow_ips,omitempty"`
	DenyIPs    []string `json:"deny_ips,omitempty"`
	AllowPorts []int    `json:"allow_ports,omitempty"`
	DenyPorts  []int    `json:"deny_ports,omitempty"`
	TargetHost string   `json:"target_host,omitempty"`
	TargetPort int      `json:"target_port,omitempty"`
}

// PolicyHandler 策略处理器
type PolicyHandler struct {
	app *types.App
}

// NewPolicyHandler 创建策略处理器
func NewPolicyHandler(app *types.App) *PolicyHandler {
	return &PolicyHandler{app: app}
}

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	Name        string       `json:"name" binding:"required,min=1,max=64"`
	Type        string       `json:"type" binding:"required,oneof=protocol acl routing"`
	Priority    int          `json:"priority" binding:"gte=0,lte=9999"`
	Enabled     bool         `json:"enabled"`
	Config      PolicyConfig `json:"config" binding:"required"`
	NodeIDs     []string     `json:"node_ids" binding:"omitempty,max=100"`
	Description string       `json:"description" binding:"omitempty,max=512"`
}

// Create 创建策略
func (h *PolicyHandler) Create(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 序列化配置和节点ID列表
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		response.GinBadRequest(c, "Invalid config format")
		return
	}

	nodeIDsJSON, err := json.Marshal(req.NodeIDs)
	if err != nil {
		response.GinBadRequest(c, "Invalid node_ids format")
		return
	}

	policy := &models.Policy{
		Name:        req.Name,
		Type:        req.Type,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		Config:      string(configJSON),
		NodeIDs:     string(nodeIDsJSON),
		Description: req.Description,
	}
	policy.ID = uuid.New().String()

	if err := h.app.DAO.CreatePolicy(policy); err != nil {
		logger.Error("创建策略失败", zap.Error(err))
		response.InternalError(c, "Failed to create policy")
		return
	}

	// 使相关节点的策略缓存失效
	if h.app.DB.HasCache() {
		for _, nodeID := range req.NodeIDs {
			_ = h.app.DB.Cache.Redis.InvalidatePolicyCache(nodeID)
		}
	}

	response.SuccessWithMessage(c, "Policy created successfully", policy)
}

// List 列出策略
func (h *PolicyHandler) List(c *gin.Context) {
	policyType := c.Query("type")
	enabledStr := c.Query("enabled")

	var enabled *bool
	if enabledStr != "" {
		val := enabledStr == "true"
		enabled = &val
	}

	policies, err := h.app.DAO.ListPolicies(policyType, enabled)
	if err != nil {
		logger.Error("获取策略列表失败", zap.Error(err))
		response.InternalError(c, "Failed to list policies")
		return
	}

	response.GinSuccess(c, gin.H{
		"policies": policies,
		"total":    len(policies),
	})
}

// Get 获取策略详情
func (h *PolicyHandler) Get(c *gin.Context) {
	id := c.Param("id")

	policy, err := h.app.DAO.GetPolicy(id)
	if err != nil {
		logger.Error("获取策略失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to get policy")
		return
	}

	if policy == nil {
		response.GinNotFound(c, "Policy not found")
		return
	}

	response.GinSuccess(c, policy)
}

// UpdatePolicyRequest 更新策略请求
type UpdatePolicyRequest struct {
	Name        string        `json:"name"`
	Priority    int           `json:"priority"`
	Enabled     *bool         `json:"enabled"`
	Config      *PolicyConfig `json:"config"`
	NodeIDs     []string      `json:"node_ids"`
	Description string        `json:"description"`
}

// Update 更新策略
func (h *PolicyHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	policy, err := h.app.DAO.GetPolicy(id)
	if err != nil {
		logger.Error("获取策略失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to get policy")
		return
	}

	if policy == nil {
		response.GinNotFound(c, "Policy not found")
		return
	}

	// 更新字段
	if req.Name != "" {
		policy.Name = req.Name
	}
	if req.Priority > 0 {
		policy.Priority = req.Priority
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		policy.Config = string(configJSON)
	}
	if req.NodeIDs != nil {
		nodeIDsJSON, _ := json.Marshal(req.NodeIDs)
		policy.NodeIDs = string(nodeIDsJSON)
	}
	if req.Description != "" {
		policy.Description = req.Description
	}

	if err := h.app.DAO.UpdatePolicy(policy); err != nil {
		logger.Error("更新策略失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to update policy")
		return
	}

	// 使相关节点的策略缓存失效
	if h.app.DB.HasCache() {
		var nodeIDs []string
		if err := json.Unmarshal([]byte(policy.NodeIDs), &nodeIDs); err == nil {
			for _, nodeID := range nodeIDs {
				_ = h.app.DB.Cache.Redis.InvalidatePolicyCache(nodeID)
			}
		}
	}

	response.SuccessWithMessage(c, "Policy updated successfully", policy)
}

// Delete 删除策略
func (h *PolicyHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	policy, _ := h.app.DAO.GetPolicy(id)
	if err := h.app.DAO.DeletePolicy(id); err != nil {
		logger.Error("删除策略失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to delete policy")
		return
	}

	// 使相关节点的策略缓存失效
	if h.app.DB.HasCache() && policy != nil {
		var nodeIDs []string
		if err := json.Unmarshal([]byte(policy.NodeIDs), &nodeIDs); err == nil {
			for _, nodeID := range nodeIDs {
				_ = h.app.DB.Cache.Redis.InvalidatePolicyCache(nodeID)
			}
		}
	}

	response.SuccessWithMessage(c, "Policy deleted successfully", nil)
}

// Deploy 部署策略到节点
func (h *PolicyHandler) Deploy(c *gin.Context) {
	id := c.Param("id")

	policy, err := h.app.DAO.GetPolicy(id)
	if err != nil {
		logger.Error("获取策略失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to get policy")
		return
	}

	if policy == nil {
		response.GinNotFound(c, "Policy not found")
		return
	}

	// 这里实现策略下发逻辑
	// 在实际应用中，可以通过WebSocket或HTTP推送给节点
	// 目前只是更新缓存

	if h.app.DB.HasCache() {
		var nodeIDs []string
		if err := json.Unmarshal([]byte(policy.NodeIDs), &nodeIDs); err == nil {
			for _, nodeID := range nodeIDs {
				_ = h.app.DB.Cache.Redis.SetPolicyForNode(nodeID, []string{policy.ID})
			}
		}
	}

	response.SuccessWithMessage(c, "Policy deployed successfully", gin.H{
		"policy_id": id,
		"status":    "deployed",
	})
}
