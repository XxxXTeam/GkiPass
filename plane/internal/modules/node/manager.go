package node

import (
	"encoding/json"
	"fmt"

	"gkipass/plane/internal/db/dao"
	dbmodels "gkipass/plane/internal/db/models"
	"gkipass/plane/internal/models"
	"gkipass/plane/internal/pkg/logger"

	"go.uber.org/zap"
)

// Manager 节点管理器
type Manager struct {
	dao *dao.DAO
}

// NewManager 创建节点管理器
func NewManager(d *dao.DAO) *Manager {
	return &Manager{dao: d}
}

// GetNodeConfig 获取节点完整配置
func (m *Manager) GetNodeConfig(nodeID string) (*models.NodeConfig, error) {
	if m.dao == nil {
		return nil, fmt.Errorf("DAO 不可用")
	}

	// 1. 获取节点信息
	ndNode, err := m.dao.GetNode(nodeID)
	if err != nil || ndNode == nil {
		return nil, fmt.Errorf("节点不存在: %s", nodeID)
	}

	// 2. 获取第一个节点组
	groupID := ""
	groupName := ""
	if len(ndNode.Groups) > 0 {
		groupID = ndNode.Groups[0].ID
		groupName = ndNode.Groups[0].Name
	}

	// 3. 获取该节点组的所有隧道
	var tunnels []dbmodels.Tunnel
	if groupID != "" {
		var tErr error
		tunnels, tErr = m.getTunnelsByGroupID(groupID)
		if tErr != nil {
			logger.Error("获取隧道列表失败", zap.Error(tErr))
		}
	}

	// 4. 构建节点信息
	nodeInfo := models.NodeInfo{
		NodeID:    ndNode.ID,
		NodeName:  ndNode.Name,
		NodeType:  string(ndNode.Role),
		GroupID:   groupID,
		GroupName: groupName,
		Region:    "",
		Tags:      make(map[string]string),
	}

	// 5. 构建隧道配置列表
	tunnelConfigs := make([]models.TunnelConfig, 0, len(tunnels))
	for _, tunnel := range tunnels {
		if !tunnel.Enabled {
			continue
		}

		/* 构建目标列表 */
		targets := []models.TargetConfig{{
			Host:       tunnel.TargetAddress,
			Port:       tunnel.TargetPort,
			Weight:     1,
			Protocol:   "tcp",
			Timeout:    30,
			MaxRetries: 3,
		}}

		tunnelConfig := models.TunnelConfig{
			TunnelID:          tunnel.ID,
			Name:              tunnel.Name,
			Protocol:          string(tunnel.Protocol),
			LocalPort:         tunnel.ListenPort,
			Targets:           targets,
			Enabled:           tunnel.Enabled,
			DisabledProtocols: []string{},
			MaxBandwidth:      tunnel.RateLimitBPS,
			MaxConnections:    tunnel.MaxConnections,
			Options:           make(map[string]interface{}),
		}

		tunnelConfigs = append(tunnelConfigs, tunnelConfig)
	}

	// 6. 获取对端服务器列表
	peerServers := m.getPeerServers(string(ndNode.Role), groupID)

	// 7. 构建节点能力配置
	capability := models.NodeCapability{
		SupportedProtocols: []string{"tcp", "udp", "http", "https"},
		MaxTunnels:         100,
		MaxBandwidth:       1000000000, // 1Gbps
		MaxConnections:     10000,
		Features: map[string]bool{
			"load_balance":  true,
			"health_check":  true,
			"traffic_stats": true,
		},
		ReportInterval:    60, // 60秒上报一次流量
		HeartbeatInterval: 30, // 30秒心跳一次
	}

	// 8. 组装完整配置
	config := &models.NodeConfig{
		NodeInfo:     nodeInfo,
		Tunnels:      tunnelConfigs,
		PeerServers:  peerServers,
		Capabilities: capability,
		Version:      "1.0.0",
		UpdatedAt:    ndNode.UpdatedAt,
	}

	logger.Info("生成节点配置",
		zap.String("nodeID", nodeID),
		zap.Int("tunnelCount", len(tunnelConfigs)))

	return config, nil
}

/*
getTunnelsByGroupID 获取节点组关联的隧道
*/
func (m *Manager) getTunnelsByGroupID(groupID string) ([]dbmodels.Tunnel, error) {
	var tunnels []dbmodels.Tunnel
	err := m.dao.DB.Where(
		"(ingress_group_id = ? OR egress_group_id = ?) AND enabled = ?",
		groupID, groupID, true,
	).Find(&tunnels).Error
	return tunnels, err
}

// parseTargetsJSON 解析目标 JSON
func (m *Manager) parseTargetsJSON(targetsJSON string) ([]models.TargetConfig, error) {
	if targetsJSON == "" {
		return []models.TargetConfig{}, nil
	}

	var raw []struct {
		Host   string `json:"host"`
		Port   int    `json:"port"`
		Weight int    `json:"weight"`
	}
	if err := json.Unmarshal([]byte(targetsJSON), &raw); err != nil {
		return nil, fmt.Errorf("解析目标列表失败: %w", err)
	}

	targets := make([]models.TargetConfig, 0, len(raw))
	for _, t := range raw {
		targets = append(targets, models.TargetConfig{
			Host:       t.Host,
			Port:       t.Port,
			Weight:     t.Weight,
			Protocol:   "tcp",
			Timeout:    30,
			MaxRetries: 3,
		})
	}
	return targets, nil
}

// getPeerServers 获取对端服务器列表
func (m *Manager) getPeerServers(nodeRole, groupID string) []models.PeerServer {
	if m.dao == nil {
		return []models.PeerServer{}
	}

	/* 根据节点角色确定对端角色 */
	var targetRole string
	if nodeRole == "ingress" || nodeRole == "entry" {
		targetRole = "egress"
	} else {
		targetRole = "ingress"
	}

	groups, err := m.dao.ListNodeGroups(targetRole)
	if err != nil {
		logger.Error("获取节点组失败", zap.Error(err))
		return []models.PeerServer{}
	}

	peerServers := make([]models.PeerServer, 0)
	for _, group := range groups {
		nodes, _ := m.dao.ListNodes(group.ID, "online", 100, 0)
		for _, nd := range nodes {
			peer := models.PeerServer{
				ServerID:   nd.ID,
				ServerName: nd.Name,
				Host:       nd.PublicIP,
				Port:       nd.Port,
				Type:       string(nd.Role),
				Region:     "",
				Priority:   1,
				Protocols:  []string{"tcp", "udp", "http", "https"},
			}
			peerServers = append(peerServers, peer)
		}
	}

	return peerServers
}
