package node

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NodeDeployHandler 节点部署处理器
type NodeDeployHandler struct {
	app *types.App
}

// NewNodeDeployHandler 创建节点部署处理器
func NewNodeDeployHandler(app *types.App) *NodeDeployHandler {
	return &NodeDeployHandler{app: app}
}

// DeployNodeRequest 部署节点请求
type DeployNodeRequest struct {
	Name         string `json:"name"`          // 服务器名（可选）
	ConnectionIP string `json:"connection_ip"` // 连接IP或域名（可选）
	ExitNetwork  string `json:"exit_network"`  // 出口网络（可选）
	DebugMode    bool   `json:"debug_mode"`    // 调试模式
}

// CreateNodeResponse 创建节点响应
type CreateNodeResponse struct {
	ID              string    `json:"id"`
	GroupID         string    `json:"group_id"`
	Name            string    `json:"name,omitempty"`
	DeploymentToken string    `json:"deployment_token"`
	ConnectionIP    string    `json:"connection_ip,omitempty"`
	ExitNetwork     string    `json:"exit_network,omitempty"`
	DebugMode       bool      `json:"debug_mode"`
	Status          string    `json:"status"`
	InstallCommand  string    `json:"install_command"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateNode 创建节点并生成部署Token
func (h *NodeDeployHandler) CreateNode(c *gin.Context) {
	groupID := c.Param("id")

	var req DeployNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	/* 验证节点组是否存在 */
	group, err := h.app.DAO.GetNodeGroup(groupID)
	if err != nil || group == nil {
		response.GinNotFound(c, "节点组不存在")
		return
	}

	deploymentToken := generateDeploymentToken()

	node := &models.Node{
		Name:        req.Name,
		Description: fmt.Sprintf("Deployment pending - Token: %s", deploymentToken[:8]+"..."),
		Status:      models.NodeStatusConnecting,
		Role:        group.Role,
		PublicIP:    req.ConnectionIP,
		Port:        443,
		Token:       deploymentToken,
	}

	if err := h.app.DAO.CreateNode(node); err != nil {
		logger.Error("创建节点失败", zap.Error(err))
		response.InternalError(c, "创建节点失败")
		return
	}

	/* 将节点加入组 */
	_ = h.app.DAO.AddNodeToGroup(node.ID, groupID)

	/* 生成安装命令 */
	serverURL := fmt.Sprintf("https://%s", c.Request.Host)
	installCmd := generateInstallCommand(serverURL, deploymentToken, node.ID, req)

	resp := CreateNodeResponse{
		ID:              node.ID,
		GroupID:         groupID,
		Name:            req.Name,
		DeploymentToken: deploymentToken,
		ConnectionIP:    req.ConnectionIP,
		ExitNetwork:     req.ExitNetwork,
		DebugMode:       req.DebugMode,
		Status:          "pending",
		InstallCommand:  installCmd,
		CreatedAt:       node.CreatedAt,
	}

	response.SuccessWithMessage(c, "Node created successfully", resp)
}

// RegisterNodeRequest 节点注册请求（服务器端调用）
type RegisterNodeRequest struct {
	NodeID          string                 `json:"node_id" binding:"required"`
	DeploymentToken string                 `json:"deployment_token" binding:"required"`
	ServerInfo      map[string]interface{} `json:"server_info"`
}

// RegisterNode 节点注册
func (h *NodeDeployHandler) RegisterNode(c *gin.Context) {
	var req RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	/* 获取节点 */
	node, err := h.app.DAO.GetNode(req.NodeID)
	if err != nil || node == nil {
		response.GinNotFound(c, "节点不存在")
		return
	}

	/* 验证部署Token（常量时间比较，防止时序攻击） */
	if subtle.ConstantTimeCompare([]byte(node.Token), []byte(req.DeploymentToken)) != 1 {
		response.GinUnauthorized(c, "部署令牌无效")
		return
	}

	if node.Status != models.NodeStatusConnecting {
		response.GinBadRequest(c, "节点已注册")
		return
	}

	/* 更新节点状态 */
	node.Status = models.NodeStatusOnline
	node.LastOnline = time.Now()
	node.Token = "" /* 清除部署令牌 */

	if publicIP, ok := req.ServerInfo["public_ip"].(string); ok && publicIP != "" {
		node.PublicIP = publicIP
	}

	if err := h.app.DAO.UpdateNode(node); err != nil {
		logger.Error("注册节点失败", zap.Error(err))
		response.InternalError(c, "注册节点失败")
		return
	}

	response.GinSuccessWithMessage(c, "节点注册成功", gin.H{
		"node_id": node.ID,
	})
}

// NodeHeartbeatRequest 节点心跳请求
type NodeHeartbeatRequest struct {
	DeploymentToken string                 `json:"deployment_token"`
	Status          map[string]interface{} `json:"status"`
}

// NodeHeartbeat 节点心跳
func (h *NodeDeployHandler) NodeHeartbeat(c *gin.Context) {
	nodeID := c.Param("id")

	var req NodeHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	_ = h.app.DAO.UpdateNodeStatus(nodeID, models.NodeStatusOnline)

	response.GinSuccess(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
	})
}

// ListNodesInGroup 列出节点组内的节点
func (h *NodeDeployHandler) ListNodesInGroup(c *gin.Context) {
	groupID := c.Param("id")

	nodes, err := h.app.DAO.GetNodesInGroup(groupID)
	if err != nil {
		logger.Error("获取节点列表失败", zap.Error(err))
		response.InternalError(c, "获取节点列表失败")
		return
	}

	var connecting, online, offline int
	for _, node := range nodes {
		switch node.Status {
		case models.NodeStatusConnecting:
			connecting++
		case models.NodeStatusOnline:
			if time.Since(node.LastOnline) > 5*time.Minute {
				offline++
			} else {
				online++
			}
		default:
			offline++
		}
	}

	response.GinSuccess(c, gin.H{
		"data":       nodes,
		"total":      len(nodes),
		"connecting": connecting,
		"online":     online,
		"offline":    offline,
	})
}

// generateDeploymentToken 生成部署Token
func generateDeploymentToken() string {
	// 格式：gkipass_deploy_{timestamp}_{random}
	timestamp := time.Now().Unix()

	// 生成32字节随机数
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("gkipass_deploy_%d_%s", timestamp, randomStr)
}

/* shellSafePattern 仅允许安全字符通过，防止命令注入 */
var shellSafePattern = regexp.MustCompile(`[^a-zA-Z0-9._\-:/]`)

/* sanitizeShellArg 过滤 shell 参数中的危险字符 */
func sanitizeShellArg(s string) string {
	s = strings.TrimSpace(s)
	return shellSafePattern.ReplaceAllString(s, "")
}

// generateInstallCommand 生成安装命令
func generateInstallCommand(serverURL, token, nodeID string, req DeployNodeRequest) string {
	baseCmd := fmt.Sprintf("curl -sSL https://dl.relayx.cc/install.sh | bash -s -- -s %s -t %s -n %s",
		sanitizeShellArg(serverURL), sanitizeShellArg(token), sanitizeShellArg(nodeID))

	// 添加可选参数（已过滤危险字符）
	if req.Name != "" {
		baseCmd += fmt.Sprintf(" --name %s", sanitizeShellArg(req.Name))
	}
	if req.ConnectionIP != "" {
		baseCmd += fmt.Sprintf(" --ip %s", sanitizeShellArg(req.ConnectionIP))
	}
	if req.ExitNetwork != "" {
		baseCmd += fmt.Sprintf(" --interface %s", sanitizeShellArg(req.ExitNetwork))
	}
	if req.DebugMode {
		baseCmd += " --debug"
	}

	return baseCmd
}
