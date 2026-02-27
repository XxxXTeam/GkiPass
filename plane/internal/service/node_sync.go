package service

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/dao"
	dbmodels "gkipass/plane/internal/db/models"
	"gkipass/plane/internal/models"
	"gkipass/plane/internal/pkg/logger"

	"go.uber.org/zap"
)

// NodeSyncService 节点同步服务
type NodeSyncService struct {
	dao *dao.DAO
}

// NewNodeSyncService 创建节点同步服务
func NewNodeSyncService(d *dao.DAO) *NodeSyncService {
	return &NodeSyncService{dao: d}
}

// SyncTunnelsToNode 同步隧道到指定节点
func (s *NodeSyncService) SyncTunnelsToNode(nodeID string) error {
	ndNode, err := s.dao.GetNode(nodeID)
	if err != nil || ndNode == nil {
		return fmt.Errorf("节点 %s 不存在", nodeID)
	}

	groupID := ""
	if len(ndNode.Groups) > 0 {
		groupID = ndNode.Groups[0].ID
	}

	var tunnels []dbmodels.Tunnel
	if groupID != "" {
		s.dao.DB.Where("(ingress_group_id = ? OR egress_group_id = ?) AND enabled = ?",
			groupID, groupID, true).Find(&tunnels)
	}

	logger.Info("同步隧道到节点",
		zap.String("nodeID", nodeID),
		zap.String("groupID", groupID),
		zap.Int("tunnelCount", len(tunnels)))

	return nil
}

// SyncTunnelsToGroup 同步隧道到节点组的所有节点
func (s *NodeSyncService) SyncTunnelsToGroup(groupID string) error {
	nodes, err := s.dao.ListNodes(groupID, "", 1000, 0)
	if err != nil {
		return fmt.Errorf("获取节点列表失败: %w", err)
	}

	for _, nd := range nodes {
		if err := s.SyncTunnelsToNode(nd.ID); err != nil {
			logger.Error("同步隧道到节点失败",
				zap.String("nodeID", nd.ID),
				zap.Error(err))
		}
	}

	return nil
}

// BuildNodeConfig 构建节点配置
func (s *NodeSyncService) BuildNodeConfig(nodeID string) (*models.NodeConfig, error) {
	ndNode, err := s.dao.GetNode(nodeID)
	if err != nil || ndNode == nil {
		return nil, fmt.Errorf("节点不存在")
	}

	groupID := ""
	groupName := ""
	if len(ndNode.Groups) > 0 {
		groupID = ndNode.Groups[0].ID
		groupName = ndNode.Groups[0].Name
	}

	/* 获取该节点组的所有隧道 */
	var tunnels []dbmodels.Tunnel
	if groupID != "" {
		s.dao.DB.Where("(ingress_group_id = ? OR egress_group_id = ?) AND enabled = ?",
			groupID, groupID, true).Find(&tunnels)
	}

	tunnelConfigs := make([]models.TunnelConfig, 0, len(tunnels))
	for _, tunnel := range tunnels {
		if !tunnel.Enabled {
			continue
		}

		targets := []models.TargetConfig{{
			Host:       tunnel.TargetAddress,
			Port:       tunnel.TargetPort,
			Weight:     1,
			Protocol:   string(tunnel.Protocol),
			Timeout:    30,
			MaxRetries: 3,
		}}

		tunnelConfigs = append(tunnelConfigs, models.TunnelConfig{
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
		})
	}

	peerServers := s.getPeerServers(ndNode)
	configVersion := fmt.Sprintf("%d", time.Now().Unix())

	config := &models.NodeConfig{
		NodeInfo: models.NodeInfo{
			NodeID:    ndNode.ID,
			NodeName:  ndNode.Name,
			NodeType:  string(ndNode.Role),
			GroupID:   groupID,
			GroupName: groupName,
			Region:    "default",
			Tags:      make(map[string]string),
		},
		Tunnels:     tunnelConfigs,
		PeerServers: peerServers,
		Capabilities: models.NodeCapability{
			SupportedProtocols: []string{"tcp", "udp", "http", "https"},
			MaxTunnels:         1000,
			MaxBandwidth:       1000 * 1024 * 1024,
			MaxConnections:     10000,
			Features:           make(map[string]bool),
			ReportInterval:     60,
			HeartbeatInterval:  30,
		},
		Version:   configVersion,
		UpdatedAt: time.Now(),
	}

	return config, nil
}

// getPeerServers 获取对端服务器列表
func (s *NodeSyncService) getPeerServers(ndNode *dbmodels.Node) []models.PeerServer {
	var targetRole string
	if ndNode.Role == "ingress" || ndNode.Role == "entry" {
		targetRole = "egress"
	} else {
		targetRole = "ingress"
	}

	groups, err := s.dao.ListNodeGroups(targetRole)
	if err != nil {
		logger.Error("获取对端节点组失败", zap.Error(err))
		return []models.PeerServer{}
	}

	peerServers := make([]models.PeerServer, 0)
	for _, g := range groups {
		nodes, _ := s.dao.ListNodes(g.ID, "online", 100, 0)
		for _, nd := range nodes {
			peerServers = append(peerServers, models.PeerServer{
				ServerID:   nd.ID,
				ServerName: nd.Name,
				Host:       nd.PublicIP,
				Port:       nd.Port,
			})
		}
	}

	return peerServers
}
func (s *NodeSyncService) NotifyConfigUpdate(nodeID string, config *models.NodeConfig) error {
	logger.Info("通知节点配置更新",
		zap.String("nodeID", nodeID),
		zap.String("configVersion", config.Version),
		zap.Int("tunnelCount", len(config.Tunnels)))

	return nil
}

func (s *NodeSyncService) UpdateNodeConfig(tunnelID string) error {
	var tunnel dbmodels.Tunnel
	if err := s.dao.DB.First(&tunnel, "id = ?", tunnelID).Error; err != nil {
		return err
	}

	if tunnel.IngressGroupID != "" {
		if err := s.SyncTunnelsToGroup(tunnel.IngressGroupID); err != nil {
			logger.Error("同步入口组失败", zap.Error(err))
		}
	}

	if tunnel.EgressGroupID != "" {
		if err := s.SyncTunnelsToGroup(tunnel.EgressGroupID); err != nil {
			logger.Error("同步出口组失败", zap.Error(err))
		}
	}

	return nil
}
