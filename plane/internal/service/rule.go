package service

import (
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"gkipass/plane/internal/model"
)

/*
RuleService 规则服务
功能：提供转发规则的 CRUD 操作和节点规则同步能力
*/
type RuleService struct {
	db     *sql.DB
	logger *zap.Logger
}

/*
NewRuleService 创建规则服务
*/
func NewRuleService() *RuleService {
	return &RuleService{
		logger: zap.L().Named("rule-service"),
	}
}

/*
NewRuleServiceWithDB 创建带数据库连接的规则服务
*/
func NewRuleServiceWithDB(db *sql.DB) *RuleService {
	return &RuleService{
		db:     db,
		logger: zap.L().Named("rule-service"),
	}
}

/*
GetRule 获取规则
功能：根据 ID 从数据库查询单条规则及其关联的 ACL 规则和选项
*/
func (s *RuleService) GetRule(id string) (*model.Rule, error) {
	if id == "" {
		return nil, errors.New("规则ID不能为空")
	}
	if s.db == nil {
		return nil, errors.New("数据库未初始化")
	}
	return model.GetRule(s.db, id)
}

/*
ListRules 列出所有规则
功能：从数据库查询全部规则列表，包括 ACL 和选项
*/
func (s *RuleService) ListRules() ([]*model.Rule, error) {
	if s.db == nil {
		return []*model.Rule{}, nil
	}
	return model.ListRules(s.db)
}

/*
CreateRule 创建规则
功能：将新规则写入数据库，包括关联的 ACL 规则和选项
*/
func (s *RuleService) CreateRule(rule *model.Rule) error {
	if rule == nil {
		return errors.New("规则不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	if err := model.CreateRule(s.db, rule); err != nil {
		s.logger.Error("创建规则失败", zap.String("name", rule.Name), zap.Error(err))
		return fmt.Errorf("创建规则失败: %w", err)
	}

	s.logger.Info("创建规则成功", zap.String("id", rule.ID), zap.String("name", rule.Name))
	return nil
}

/*
UpdateRule 更新规则
功能：更新数据库中已有规则的配置信息
*/
func (s *RuleService) UpdateRule(rule *model.Rule) error {
	if rule == nil {
		return errors.New("规则不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	if err := model.UpdateRule(s.db, rule); err != nil {
		s.logger.Error("更新规则失败", zap.String("id", rule.ID), zap.Error(err))
		return fmt.Errorf("更新规则失败: %w", err)
	}

	s.logger.Info("更新规则成功", zap.String("id", rule.ID), zap.String("name", rule.Name))
	return nil
}

/*
DeleteRule 删除规则
功能：从数据库中删除规则及其关联的 ACL 和选项
*/
func (s *RuleService) DeleteRule(id string) error {
	if id == "" {
		return errors.New("规则ID不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	if err := model.DeleteRule(s.db, id); err != nil {
		s.logger.Error("删除规则失败", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("删除规则失败: %w", err)
	}

	s.logger.Info("删除规则成功", zap.String("id", id))
	return nil
}

/*
ResetRuleStats 重置规则统计
功能：将指定规则的连接数和流量统计归零
*/
func (s *RuleService) ResetRuleStats(id string) error {
	if id == "" {
		return errors.New("规则ID不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	if err := model.ResetRuleStats(s.db, id); err != nil {
		s.logger.Error("重置规则统计失败", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("重置规则统计失败: %w", err)
	}

	s.logger.Info("重置规则统计成功", zap.String("id", id))
	return nil
}

/*
GetNodeRules 获取节点的规则列表
功能：查询分配给指定节点（入口或出口）的所有转发规则
*/
func (s *RuleService) GetNodeRules(nodeID string) ([]*model.Rule, error) {
	if nodeID == "" {
		return nil, errors.New("节点ID不能为空")
	}
	if s.db == nil {
		return []*model.Rule{}, nil
	}

	rows, err := s.db.Query(`
		SELECT 
			id, name, description, enabled, priority, version, created_at, updated_at, created_by,
			protocol, listen_port, target_address, target_port, ingress_node_id, egress_node_id,
			ingress_group_id, egress_group_id, ingress_protocol, egress_protocol,
			enable_encryption, rate_limit_bps, max_connections, idle_timeout,
			connection_count, bytes_in, bytes_out, last_active
		FROM rules
		WHERE ingress_node_id = ? OR egress_node_id = ?
		ORDER BY priority DESC, name ASC
	`, nodeID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("查询节点规则失败: %w", err)
	}
	defer rows.Close()

	var rules []*model.Rule
	for rows.Next() {
		var rule model.Rule
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.Enabled, &rule.Priority, &rule.Version,
			&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedBy, &rule.Protocol, &rule.ListenPort,
			&rule.TargetAddress, &rule.TargetPort, &rule.IngressNodeID, &rule.EgressNodeID,
			&rule.IngressGroupID, &rule.EgressGroupID, &rule.IngressProtocol, &rule.EgressProtocol,
			&rule.EnableEncryption, &rule.RateLimitBPS, &rule.MaxConnections, &rule.IdleTimeout,
			&rule.ConnectionCount, &rule.BytesIn, &rule.BytesOut, &rule.LastActive,
		); err != nil {
			return nil, fmt.Errorf("扫描规则数据失败: %w", err)
		}
		rules = append(rules, &rule)
	}

	return rules, nil
}

/*
SyncRulesToNode 同步规则到节点
功能：获取节点的所有规则并通过 WebSocket 下发到节点端执行
*/
func (s *RuleService) SyncRulesToNode(nodeID string) error {
	if nodeID == "" {
		return errors.New("节点ID不能为空")
	}

	rules, err := s.GetNodeRules(nodeID)
	if err != nil {
		return fmt.Errorf("获取节点规则失败: %w", err)
	}

	s.logger.Info("准备同步规则到节点",
		zap.String("node_id", nodeID),
		zap.Int("rule_count", len(rules)))

	/* 规则同步通过 WebSocket 协议下发，此处仅获取规则数据 */
	/* 实际下发逻辑由 ws.Server 的 SyncRules 方法处理 */

	return nil
}
