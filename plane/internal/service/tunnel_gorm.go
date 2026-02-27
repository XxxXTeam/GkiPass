package service

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

/*
  GormTunnelService 基于 GORM 的隧道服务
  功能：管理隧道的完整生命周期（创建/查询/更新/删除/切换状态），
  自动同步创建转发规则，支持加密配置和流量统计
*/
type GormTunnelService struct {
	db     *gorm.DB
	logger *zap.Logger
}

/*
  NewGormTunnelService 创建基于 GORM 的隧道服务
*/
func NewGormTunnelService(db *gorm.DB) *GormTunnelService {
	return &GormTunnelService{
		db:     db,
		logger: zap.L().Named("gorm-tunnel-service"),
	}
}

/*
  CreateTunnelRequest 创建隧道请求
  功能：定义创建隧道时的输入参数
*/
type CreateTunnelRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	IngressNodeID    string `json:"ingress_node_id"`
	EgressNodeID     string `json:"egress_node_id"`
	IngressGroupID   string `json:"ingress_group_id"`
	EgressGroupID    string `json:"egress_group_id"`
	Protocol         string `json:"protocol"`
	IngressProtocol  string `json:"ingress_protocol"`
	EgressProtocol   string `json:"egress_protocol"`
	ListenPort       int    `json:"listen_port" binding:"required"`
	TargetAddress    string `json:"target_address" binding:"required"`
	TargetPort       int    `json:"target_port" binding:"required"`
	EnableEncryption bool   `json:"enable_encryption"`
	EncryptionMethod string `json:"encryption_method"`
	RateLimitBPS     int64  `json:"rate_limit_bps"`
	MaxConnections   int    `json:"max_connections"`
	IdleTimeout      int    `json:"idle_timeout"`
	LoadBalanceMode  string `json:"load_balance_mode"`
}

/*
  CreateTunnel 创建隧道
  功能：在事务中创建隧道和关联的默认转发规则
  流程：验证 → 创建隧道 → 创建默认规则 → 提交
*/
func (s *GormTunnelService) CreateTunnel(req *CreateTunnelRequest, userID string) (*models.Tunnel, error) {
	/* 参数验证 */
	if req.Name == "" {
		return nil, fmt.Errorf("隧道名称不能为空")
	}
	if req.ListenPort <= 0 || req.ListenPort > 65535 {
		return nil, fmt.Errorf("监听端口必须在 1-65535 之间")
	}
	if req.TargetPort <= 0 || req.TargetPort > 65535 {
		return nil, fmt.Errorf("目标端口必须在 1-65535 之间")
	}
	if req.TargetAddress == "" {
		return nil, fmt.Errorf("目标地址不能为空")
	}

	/* 设置默认值 */
	protocol := models.TunnelProtocol(req.Protocol)
	if protocol == "" {
		protocol = models.ProtocolTCP
	}
	ingressProtocol := models.TunnelProtocol(req.IngressProtocol)
	if ingressProtocol == "" {
		ingressProtocol = protocol
	}
	egressProtocol := models.TunnelProtocol(req.EgressProtocol)
	if egressProtocol == "" {
		egressProtocol = protocol
	}
	encryptionMethod := req.EncryptionMethod
	if encryptionMethod == "" {
		encryptionMethod = "aes-256-gcm"
	}
	idleTimeout := req.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 300
	}
	loadBalanceMode := req.LoadBalanceMode
	if loadBalanceMode == "" {
		loadBalanceMode = "round-robin"
	}

	/* 检查端口冲突 */
	if err := s.checkPortConflict(req.IngressGroupID, req.ListenPort, ""); err != nil {
		return nil, err
	}

	tunnel := &models.Tunnel{
		Name:             req.Name,
		Description:      req.Description,
		Enabled:          true,
		CreatedBy:        userID,
		IngressNodeID:    req.IngressNodeID,
		EgressNodeID:     req.EgressNodeID,
		IngressGroupID:   req.IngressGroupID,
		EgressGroupID:    req.EgressGroupID,
		Protocol:         protocol,
		IngressProtocol:  ingressProtocol,
		EgressProtocol:   egressProtocol,
		ListenPort:       req.ListenPort,
		TargetAddress:    req.TargetAddress,
		TargetPort:       req.TargetPort,
		EnableEncryption: req.EnableEncryption,
		EncryptionMethod: encryptionMethod,
		RateLimitBPS:     req.RateLimitBPS,
		MaxConnections:   req.MaxConnections,
		IdleTimeout:      idleTimeout,
		LoadBalanceMode:  loadBalanceMode,
	}

	/* 事务中创建隧道和默认规则 */
	err := s.db.Transaction(func(tx *gorm.DB) error {
		/* 创建隧道 */
		if err := tx.Create(tunnel).Error; err != nil {
			return fmt.Errorf("创建隧道失败: %w", err)
		}

		/* 创建默认转发规则 */
		rule := &models.Rule{
			Name:             tunnel.Name + " - 默认规则",
			Description:      "隧道自动创建的默认转发规则",
			Enabled:          true,
			Priority:         0,
			Version:          1,
			CreatedBy:        userID,
			TunnelID:         tunnel.ID,
			GroupID:          tunnel.IngressGroupID,
			Protocol:         tunnel.Protocol,
			ListenPort:       tunnel.ListenPort,
			TargetAddress:    tunnel.TargetAddress,
			TargetPort:       tunnel.TargetPort,
			IngressNodeID:    tunnel.IngressNodeID,
			EgressNodeID:     tunnel.EgressNodeID,
			IngressGroupID:   tunnel.IngressGroupID,
			EgressGroupID:    tunnel.EgressGroupID,
			IngressProtocol:  tunnel.IngressProtocol,
			EgressProtocol:   tunnel.EgressProtocol,
			EnableEncryption: tunnel.EnableEncryption,
			RateLimitBPS:     tunnel.RateLimitBPS,
			MaxConnections:   tunnel.MaxConnections,
			IdleTimeout:      tunnel.IdleTimeout,
		}

		if err := tx.Create(rule).Error; err != nil {
			return fmt.Errorf("创建默认规则失败: %w", err)
		}

		return nil
	})

	if err != nil {
		s.logger.Error("创建隧道事务失败",
			zap.String("name", req.Name),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("隧道创建成功",
		zap.String("id", tunnel.ID),
		zap.String("name", tunnel.Name),
		zap.String("created_by", userID))

	return tunnel, nil
}

/*
  GetTunnel 获取隧道详情
  功能：根据 ID 查询隧道，预加载关联的规则和目标
*/
func (s *GormTunnelService) GetTunnel(id string) (*models.Tunnel, error) {
	var tunnel models.Tunnel
	err := s.db.
		Preload("Rules").
		Preload("Targets").
		First(&tunnel, "id = ?", id).Error

	if err != nil {
		return nil, fmt.Errorf("隧道不存在: %s", id)
	}

	return &tunnel, nil
}

/*
  ListTunnels 列出隧道
  功能：查询隧道列表，支持按用户ID和启用状态过滤
*/
func (s *GormTunnelService) ListTunnels(userID string, enabledOnly bool) ([]models.Tunnel, error) {
	var tunnels []models.Tunnel
	query := s.db.Preload("Rules").Preload("Targets")

	if userID != "" {
		query = query.Where("created_by = ?", userID)
	}
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}

	if err := query.Order("created_at DESC").Find(&tunnels).Error; err != nil {
		return nil, fmt.Errorf("查询隧道列表失败: %w", err)
	}

	return tunnels, nil
}

/*
  UpdateTunnel 更新隧道
  功能：更新隧道配置并同步更新关联规则
*/
func (s *GormTunnelService) UpdateTunnel(id string, req *CreateTunnelRequest) (*models.Tunnel, error) {
	var tunnel models.Tunnel
	if err := s.db.First(&tunnel, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("隧道不存在: %s", id)
	}

	/* 检查端口冲突（排除自身） */
	groupID := req.IngressGroupID
	if groupID == "" {
		groupID = tunnel.IngressGroupID
	}
	if req.ListenPort != tunnel.ListenPort {
		if err := s.checkPortConflict(groupID, req.ListenPort, id); err != nil {
			return nil, err
		}
	}

	/* 事务中更新隧道和规则 */
	err := s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"name":              req.Name,
			"description":       req.Description,
			"ingress_node_id":   req.IngressNodeID,
			"egress_node_id":    req.EgressNodeID,
			"ingress_group_id":  req.IngressGroupID,
			"egress_group_id":   req.EgressGroupID,
			"listen_port":       req.ListenPort,
			"target_address":    req.TargetAddress,
			"target_port":       req.TargetPort,
			"enable_encryption": req.EnableEncryption,
			"rate_limit_bps":    req.RateLimitBPS,
			"max_connections":   req.MaxConnections,
			"idle_timeout":      req.IdleTimeout,
		}

		if req.Protocol != "" {
			updates["protocol"] = req.Protocol
			updates["ingress_protocol"] = req.IngressProtocol
			updates["egress_protocol"] = req.EgressProtocol
		}
		if req.EncryptionMethod != "" {
			updates["encryption_method"] = req.EncryptionMethod
		}
		if req.LoadBalanceMode != "" {
			updates["load_balance_mode"] = req.LoadBalanceMode
		}

		if err := tx.Model(&tunnel).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新隧道失败: %w", err)
		}

		/* 同步更新关联规则的端口和目标 */
		ruleUpdates := map[string]interface{}{
			"listen_port":       req.ListenPort,
			"target_address":    req.TargetAddress,
			"target_port":       req.TargetPort,
			"ingress_node_id":   req.IngressNodeID,
			"egress_node_id":    req.EgressNodeID,
			"ingress_group_id":  req.IngressGroupID,
			"egress_group_id":   req.EgressGroupID,
			"enable_encryption": req.EnableEncryption,
			"rate_limit_bps":    req.RateLimitBPS,
			"max_connections":   req.MaxConnections,
			"idle_timeout":      req.IdleTimeout,
		}

		if err := tx.Model(&models.Rule{}).
			Where("tunnel_id = ?", id).
			Updates(ruleUpdates).Error; err != nil {
			return fmt.Errorf("同步更新规则失败: %w", err)
		}

		/* 递增规则版本号 */
		if err := tx.Model(&models.Rule{}).
			Where("tunnel_id = ?", id).
			UpdateColumn("version", gorm.Expr("version + 1")).Error; err != nil {
			s.logger.Warn("递增规则版本号失败", zap.Error(err))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	/* 重新查询完整数据 */
	return s.GetTunnel(id)
}

/*
  DeleteTunnel 删除隧道
  功能：在事务中删除隧道及其关联的规则、目标、ACL
*/
func (s *GormTunnelService) DeleteTunnel(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		/* 删除关联的 ACL 规则 */
		if err := tx.Where("rule_id IN (?)",
			tx.Model(&models.Rule{}).Select("id").Where("tunnel_id = ?", id),
		).Delete(&models.ACLRule{}).Error; err != nil {
			s.logger.Warn("删除关联 ACL 失败", zap.Error(err))
		}

		/* 删除关联的规则 */
		if err := tx.Where("tunnel_id = ?", id).Delete(&models.Rule{}).Error; err != nil {
			return fmt.Errorf("删除关联规则失败: %w", err)
		}

		/* 删除关联的目标 */
		if err := tx.Where("tunnel_id = ?", id).Delete(&models.TunnelTarget{}).Error; err != nil {
			return fmt.Errorf("删除关联目标失败: %w", err)
		}

		/* 删除隧道 */
		result := tx.Delete(&models.Tunnel{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("删除隧道失败: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("隧道不存在: %s", id)
		}

		return nil
	})
}

/*
  ToggleTunnel 切换隧道启用/禁用状态
  功能：同时更新隧道和其关联规则的启用状态
*/
func (s *GormTunnelService) ToggleTunnel(id string, enabled bool) (*models.Tunnel, error) {
	return nil, s.db.Transaction(func(tx *gorm.DB) error {
		/* 更新隧道状态 */
		if err := tx.Model(&models.Tunnel{}).
			Where("id = ?", id).
			Update("enabled", enabled).Error; err != nil {
			return fmt.Errorf("切换隧道状态失败: %w", err)
		}

		/* 同步更新规则状态 */
		if err := tx.Model(&models.Rule{}).
			Where("tunnel_id = ?", id).
			Update("enabled", enabled).Error; err != nil {
			s.logger.Warn("同步更新规则状态失败", zap.Error(err))
		}

		return nil
	})
}

/*
  UpdateTraffic 更新隧道流量统计
  功能：原子性更新隧道的入站/出站流量计数器和最后活跃时间
*/
func (s *GormTunnelService) UpdateTraffic(tunnelID string, bytesIn, bytesOut int64) error {
	return s.db.Model(&models.Tunnel{}).
		Where("id = ?", tunnelID).
		Updates(map[string]interface{}{
			"bytes_in":         gorm.Expr("bytes_in + ?", bytesIn),
			"bytes_out":        gorm.Expr("bytes_out + ?", bytesOut),
			"connection_count": gorm.Expr("connection_count + 1"),
			"last_active":      time.Now(),
		}).Error
}

/*
  GetTunnelsByGroupID 获取节点组关联的所有隧道
  功能：查询入口组或出口组匹配的已启用隧道
*/
func (s *GormTunnelService) GetTunnelsByGroupID(groupID string) ([]models.Tunnel, error) {
	var tunnels []models.Tunnel
	err := s.db.
		Preload("Rules").
		Preload("Targets").
		Where("enabled = ? AND (ingress_group_id = ? OR egress_group_id = ?)", true, groupID, groupID).
		Find(&tunnels).Error

	if err != nil {
		return nil, fmt.Errorf("查询组隧道列表失败: %w", err)
	}

	return tunnels, nil
}

/*
  GetRulesByTunnelID 获取隧道的所有规则
  功能：查询指定隧道的转发规则列表，预加载 ACL
*/
func (s *GormTunnelService) GetRulesByTunnelID(tunnelID string) ([]models.Rule, error) {
	var rules []models.Rule
	err := s.db.
		Preload("ACLRules").
		Where("tunnel_id = ?", tunnelID).
		Order("priority ASC").
		Find(&rules).Error

	if err != nil {
		return nil, fmt.Errorf("查询隧道规则失败: %w", err)
	}

	return rules, nil
}

/*
  checkPortConflict 检查端口冲突
  功能：在同一个入口节点组中检测端口是否已被占用
  excludeTunnelID 排除当前隧道（用于更新场景）
*/
func (s *GormTunnelService) checkPortConflict(ingressGroupID string, port int, excludeTunnelID string) error {
	if ingressGroupID == "" {
		return nil /* 无入口组时跳过检查 */
	}

	query := s.db.Model(&models.Tunnel{}).
		Where("ingress_group_id = ? AND listen_port = ? AND enabled = ?", ingressGroupID, port, true)

	if excludeTunnelID != "" {
		query = query.Where("id != ?", excludeTunnelID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("端口冲突检查失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("端口 %d 在入口节点组中已被占用", port)
	}

	return nil
}
