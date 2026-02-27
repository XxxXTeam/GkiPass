package service

import (
	"fmt"
	"sync"
	"time"

	"gkipass/plane/internal/db/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

/*
GormNodeSyncService 基于 GORM 的节点同步服务
功能：管理隧道规则到节点的同步推送，支持：
- 隧道创建/更新/删除时自动推送规则变更到相关节点
- 按节点组批量同步规则
- 版本化增量同步（仅推送变更）
- 端口冲突全局检测
*/
type GormNodeSyncService struct {
	db        *gorm.DB
	logger    *zap.Logger
	encKeySvc *EncryptionKeyService
	wsSender  WebSocketSender /* WebSocket 消息发送接口 */
	mu        sync.RWMutex
}

/*
WebSocketSender WebSocket 消息发送接口
功能：解耦同步服务与 WebSocket 实现，支持测试和替换
*/
type WebSocketSender interface {
	SendToNode(nodeID string, msgType string, data interface{}) error
	SendToGroup(nodeIDs []string, msgType string, data interface{}) error
	GetOnlineNodeIDs() []string
}

/*
NewGormNodeSyncService 创建节点同步服务
*/
func NewGormNodeSyncService(db *gorm.DB, wsSender WebSocketSender) *GormNodeSyncService {
	return &GormNodeSyncService{
		db:        db,
		logger:    zap.L().Named("gorm-node-sync"),
		encKeySvc: NewEncryptionKeyService(db),
		wsSender:  wsSender,
	}
}

/*
SyncRulePayload 同步规则消息体
功能：定义推送到节点的规则数据结构
*/
type SyncRulePayload struct {
	TunnelID         string              `json:"tunnel_id"`
	TunnelName       string              `json:"tunnel_name"`
	Protocol         string              `json:"protocol"`
	ListenPort       int                 `json:"listen_port"`
	TargetAddress    string              `json:"target_address"`
	TargetPort       int                 `json:"target_port"`
	Targets          []SyncTargetPayload `json:"targets"`
	Enabled          bool                `json:"enabled"`
	EnableEncryption bool                `json:"enable_encryption"`
	EncryptionKey    string              `json:"encryption_key,omitempty"`
	EncryptionAlgo   string              `json:"encryption_algo,omitempty"`
	RateLimitBPS     int64               `json:"rate_limit_bps"`
	MaxConnections   int                 `json:"max_connections"`
	IdleTimeout      int                 `json:"idle_timeout"`
	Version          int64               `json:"version"`
	UserID           string              `json:"user_id"`

	/*
		出口容灾策略（由面板下发，节点自主执行）
		节点检测到出口组不可达超过 FailoverTimeout 秒后，
		自动切换到 FailoverTargets 中的目标节点
	*/
	FailoverTargets     []SyncTargetPayload `json:"failover_targets,omitempty"`      /* 容灾出口组的目标节点列表 */
	FailoverTimeout     int                 `json:"failover_timeout,omitempty"`      /* 容灾触发超时（秒） */
	FailoverAutoRecover bool                `json:"failover_auto_recover,omitempty"` /* 原出口恢复后是否自动回切 */
	FailoverGroupID     string              `json:"failover_group_id,omitempty"`     /* 容灾出口组 ID（用于事件上报） */
}

/*
SyncTargetPayload 同步目标
*/
type SyncTargetPayload struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Weight  int    `json:"weight"`
	Enabled bool   `json:"enabled"`
}

/*
SyncRulesMessage 同步规则消息
功能：包含完整的规则列表，用于全量同步
*/
type SyncRulesMessage struct {
	Rules   []SyncRulePayload `json:"rules"`
	Force   bool              `json:"force"`
	Version string            `json:"version"`
}

/*
DeleteRuleMessage 删除规则消息
*/
type DeleteRuleMessage struct {
	TunnelID string `json:"tunnel_id"`
}

/*
OnTunnelCreated 隧道创建后触发同步
功能：将新隧道的规则推送到入口组和出口组的所有在线节点
*/
func (s *GormNodeSyncService) OnTunnelCreated(tunnel *models.Tunnel) error {
	s.logger.Info("隧道创建，触发规则同步",
		zap.String("tunnel_id", tunnel.ID),
		zap.String("name", tunnel.Name))

	/* 生成加密密钥（如果启用加密） */
	if tunnel.EnableEncryption {
		if _, err := s.encKeySvc.EnsureKeyForTunnel(tunnel); err != nil {
			s.logger.Error("生成加密密钥失败", zap.Error(err))
		}
	}

	/* 同步到入口组节点 */
	if tunnel.IngressGroupID != "" {
		if err := s.syncTunnelToGroup(tunnel, tunnel.IngressGroupID); err != nil {
			s.logger.Error("同步到入口组失败",
				zap.String("group_id", tunnel.IngressGroupID),
				zap.Error(err))
		}
	}

	/* 同步到出口组节点 */
	if tunnel.EgressGroupID != "" && tunnel.EgressGroupID != tunnel.IngressGroupID {
		if err := s.syncTunnelToGroup(tunnel, tunnel.EgressGroupID); err != nil {
			s.logger.Error("同步到出口组失败",
				zap.String("group_id", tunnel.EgressGroupID),
				zap.Error(err))
		}
	}

	return nil
}

/*
OnTunnelUpdated 隧道更新后触发同步
功能：将更新后的隧道规则推送到关联的所有节点
*/
func (s *GormNodeSyncService) OnTunnelUpdated(tunnel *models.Tunnel) error {
	s.logger.Info("隧道更新，触发规则同步",
		zap.String("tunnel_id", tunnel.ID))

	return s.OnTunnelCreated(tunnel) /* 更新和创建的同步逻辑相同 */
}

/*
OnTunnelDeleted 隧道删除后触发同步
功能：通知关联节点删除对应的转发规则
*/
func (s *GormNodeSyncService) OnTunnelDeleted(tunnelID, ingressGroupID, egressGroupID string) error {
	s.logger.Info("隧道删除，触发规则清理",
		zap.String("tunnel_id", tunnelID))

	deleteMsg := &DeleteRuleMessage{TunnelID: tunnelID}

	/* 通知入口组节点 */
	if ingressGroupID != "" {
		nodeIDs := s.getOnlineNodeIDsByGroup(ingressGroupID)
		if len(nodeIDs) > 0 {
			if err := s.wsSender.SendToGroup(nodeIDs, "delete_rule", deleteMsg); err != nil {
				s.logger.Error("通知入口组删除规则失败", zap.Error(err))
			}
		}
	}

	/* 通知出口组节点 */
	if egressGroupID != "" && egressGroupID != ingressGroupID {
		nodeIDs := s.getOnlineNodeIDsByGroup(egressGroupID)
		if len(nodeIDs) > 0 {
			if err := s.wsSender.SendToGroup(nodeIDs, "delete_rule", deleteMsg); err != nil {
				s.logger.Error("通知出口组删除规则失败", zap.Error(err))
			}
		}
	}

	return nil
}

/*
SyncAllRulesToNode 全量同步规则到指定节点
功能：将节点所在组的所有启用隧道规则推送到该节点，
通常在节点首次注册或重连时调用
*/
func (s *GormNodeSyncService) SyncAllRulesToNode(nodeID string) error {
	/* 查询节点信息 */
	var node models.Node
	if err := s.db.Preload("Groups").First(&node, "id = ?", nodeID).Error; err != nil {
		return fmt.Errorf("节点不存在: %s", nodeID)
	}

	/* 收集节点所在所有组的隧道 */
	allRules := make([]SyncRulePayload, 0)

	for _, group := range node.Groups {
		rules, err := s.buildRulesForGroup(group.ID)
		if err != nil {
			s.logger.Error("构建组规则失败",
				zap.String("group_id", group.ID),
				zap.Error(err))
			continue
		}
		allRules = append(allRules, rules...)
	}

	/* 推送全量规则 */
	syncMsg := &SyncRulesMessage{
		Rules:   allRules,
		Force:   true,
		Version: fmt.Sprintf("%d", time.Now().Unix()),
	}

	if err := s.wsSender.SendToNode(nodeID, "sync_rules", syncMsg); err != nil {
		return fmt.Errorf("推送规则到节点失败: %w", err)
	}

	s.logger.Info("全量同步规则到节点完成",
		zap.String("node_id", nodeID),
		zap.Int("rule_count", len(allRules)))

	return nil
}

/*
SyncAllRulesToGroup 全量同步规则到节点组
功能：将组内所有启用隧道的规则推送到组内所有在线节点
*/
func (s *GormNodeSyncService) SyncAllRulesToGroup(groupID string) error {
	nodeIDs := s.getOnlineNodeIDsByGroup(groupID)
	if len(nodeIDs) == 0 {
		s.logger.Debug("组内无在线节点", zap.String("group_id", groupID))
		return nil
	}

	rules, err := s.buildRulesForGroup(groupID)
	if err != nil {
		return fmt.Errorf("构建组规则失败: %w", err)
	}

	syncMsg := &SyncRulesMessage{
		Rules:   rules,
		Force:   true,
		Version: fmt.Sprintf("%d", time.Now().Unix()),
	}

	if err := s.wsSender.SendToGroup(nodeIDs, "sync_rules", syncMsg); err != nil {
		return fmt.Errorf("推送规则到组失败: %w", err)
	}

	s.logger.Info("全量同步规则到组完成",
		zap.String("group_id", groupID),
		zap.Int("node_count", len(nodeIDs)),
		zap.Int("rule_count", len(rules)))

	return nil
}

/*
CheckGlobalPortConflict 全局端口冲突检测
功能：在指定节点组中检查端口是否已被其他隧道占用
*/
func (s *GormNodeSyncService) CheckGlobalPortConflict(groupID string, port int, excludeTunnelID string) (bool, string, error) {
	var tunnel models.Tunnel
	query := s.db.
		Where("ingress_group_id = ? AND listen_port = ? AND enabled = ?", groupID, port, true)

	if excludeTunnelID != "" {
		query = query.Where("id != ?", excludeTunnelID)
	}

	err := query.First(&tunnel).Error
	if err == gorm.ErrRecordNotFound {
		return false, "", nil /* 无冲突 */
	}
	if err != nil {
		return false, "", fmt.Errorf("端口冲突检查失败: %w", err)
	}

	return true, tunnel.Name, nil /* 有冲突，返回占用的隧道名称 */
}

/*
syncTunnelToGroup 同步单个隧道到节点组
*/
func (s *GormNodeSyncService) syncTunnelToGroup(tunnel *models.Tunnel, groupID string) error {
	nodeIDs := s.getOnlineNodeIDsByGroup(groupID)
	if len(nodeIDs) == 0 {
		return nil
	}

	payload, err := s.buildRulePayload(tunnel)
	if err != nil {
		return err
	}

	syncMsg := &SyncRulesMessage{
		Rules:   []SyncRulePayload{*payload},
		Force:   false,
		Version: fmt.Sprintf("%d", time.Now().Unix()),
	}

	return s.wsSender.SendToGroup(nodeIDs, "sync_rules", syncMsg)
}

/*
buildRulesForGroup 构建节点组的全量规则列表
*/
func (s *GormNodeSyncService) buildRulesForGroup(groupID string) ([]SyncRulePayload, error) {
	var tunnels []models.Tunnel
	err := s.db.
		Preload("Targets").
		Preload("Rules").
		Where("enabled = ? AND (ingress_group_id = ? OR egress_group_id = ?)", true, groupID, groupID).
		Find(&tunnels).Error

	if err != nil {
		return nil, err
	}

	rules := make([]SyncRulePayload, 0, len(tunnels))
	for i := range tunnels {
		payload, err := s.buildRulePayload(&tunnels[i])
		if err != nil {
			s.logger.Warn("构建规则payload失败",
				zap.String("tunnel_id", tunnels[i].ID),
				zap.Error(err))
			continue
		}
		rules = append(rules, *payload)
	}

	return rules, nil
}

/*
buildRulePayload 构建单个隧道的同步规则消息体
*/
func (s *GormNodeSyncService) buildRulePayload(tunnel *models.Tunnel) (*SyncRulePayload, error) {
	payload := &SyncRulePayload{
		TunnelID:         tunnel.ID,
		TunnelName:       tunnel.Name,
		Protocol:         string(tunnel.Protocol),
		ListenPort:       tunnel.ListenPort,
		TargetAddress:    tunnel.TargetAddress,
		TargetPort:       tunnel.TargetPort,
		Enabled:          tunnel.Enabled,
		EnableEncryption: tunnel.EnableEncryption,
		EncryptionAlgo:   tunnel.EncryptionMethod,
		RateLimitBPS:     tunnel.RateLimitBPS,
		MaxConnections:   tunnel.MaxConnections,
		IdleTimeout:      tunnel.IdleTimeout,
		UserID:           tunnel.CreatedBy,
	}

	/* 填充目标列表 */
	for _, target := range tunnel.Targets {
		payload.Targets = append(payload.Targets, SyncTargetPayload{
			Host:    target.Host,
			Port:    target.Port,
			Weight:  target.Weight,
			Enabled: target.Enabled,
		})
	}

	/* 如果没有显式目标，使用隧道的目标地址 */
	if len(payload.Targets) == 0 {
		payload.Targets = []SyncTargetPayload{{
			Host:    tunnel.TargetAddress,
			Port:    tunnel.TargetPort,
			Weight:  1,
			Enabled: true,
		}}
	}

	/* 获取规则版本号 */
	var maxVersion int64
	s.db.Model(&models.Rule{}).
		Where("tunnel_id = ?", tunnel.ID).
		Select("COALESCE(MAX(version), 1)").
		Scan(&maxVersion)
	payload.Version = maxVersion

	/* 获取加密密钥 */
	if tunnel.EnableEncryption {
		key, err := s.encKeySvc.GetActiveKey(tunnel.ID)
		if err == nil {
			payload.EncryptionKey = key.KeyHex
		}
	}

	/*
		填充出口容灾策略
		当隧道绑定了出口组且该组配置了容灾组时，
		查询容灾组内在线节点的公网 IP，作为 FailoverTargets 下发
	*/
	if tunnel.EgressGroupID != "" {
		var egressGroup models.NodeGroup
		if err := s.db.First(&egressGroup, "id = ?", tunnel.EgressGroupID).Error; err == nil && egressGroup.FailoverGroupID != "" {
			payload.FailoverGroupID = egressGroup.FailoverGroupID
			payload.FailoverTimeout = egressGroup.FailoverTimeout
			payload.FailoverAutoRecover = egressGroup.FailoverAutoRecover

			/* 查询容灾组内在线节点作为备用目标 */
			var failoverNodes []models.Node
			s.db.Model(&models.Node{}).
				Joins("JOIN node_group_nodes ON node_group_nodes.node_id = nodes.id").
				Where("node_group_nodes.group_id = ? AND nodes.status = ?", egressGroup.FailoverGroupID, models.NodeStatusOnline).
				Find(&failoverNodes)

			for _, node := range failoverNodes {
				if node.PublicIP != "" {
					payload.FailoverTargets = append(payload.FailoverTargets, SyncTargetPayload{
						Host:    node.PublicIP,
						Port:    tunnel.TargetPort,
						Weight:  1,
						Enabled: true,
					})
				}
			}
		}
	}

	return payload, nil
}

/*
getOnlineNodeIDsByGroup 获取组内在线节点ID列表
*/
func (s *GormNodeSyncService) getOnlineNodeIDsByGroup(groupID string) []string {
	/* 查询组内所有节点ID */
	var nodeIDs []string
	s.db.Model(&models.Node{}).
		Joins("JOIN node_group_nodes ON node_group_nodes.node_id = nodes.id").
		Where("node_group_nodes.group_id = ? AND nodes.status = ?", groupID, models.NodeStatusOnline).
		Pluck("nodes.id", &nodeIDs)

	if len(nodeIDs) == 0 {
		return nil
	}

	/* 过滤出WebSocket在线的节点 */
	if s.wsSender == nil {
		return nodeIDs
	}

	onlineIDs := s.wsSender.GetOnlineNodeIDs()
	onlineSet := make(map[string]bool, len(onlineIDs))
	for _, id := range onlineIDs {
		onlineSet[id] = true
	}

	result := make([]string, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		if onlineSet[id] {
			result = append(result, id)
		}
	}

	return result
}

/*
GetNodeSyncStatus 获取节点同步状态
功能：返回节点当前的规则同步情况（便于前端展示）
*/
func (s *GormNodeSyncService) GetNodeSyncStatus(nodeID string) (map[string]interface{}, error) {
	var node models.Node
	if err := s.db.Preload("Groups").First(&node, "id = ?", nodeID).Error; err != nil {
		return nil, fmt.Errorf("节点不存在")
	}

	/* 统计节点应同步的规则数 */
	totalRules := 0
	for _, group := range node.Groups {
		var count int64
		s.db.Model(&models.Tunnel{}).
			Where("enabled = ? AND (ingress_group_id = ? OR egress_group_id = ?)", true, group.ID, group.ID).
			Count(&count)
		totalRules += int(count)
	}

	return map[string]interface{}{
		"node_id":        nodeID,
		"node_name":      node.Name,
		"node_status":    string(node.Status),
		"groups":         len(node.Groups),
		"expected_rules": totalRules,
		"last_sync":      time.Now().Format(time.RFC3339),
	}, nil
}
