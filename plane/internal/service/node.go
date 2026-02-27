package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"gkipass/plane/internal/model"
)

/*
NodeService 节点服务
功能：提供节点的 CRUD 操作、状态管理和分组查询
*/
type NodeService struct {
	db     *sql.DB
	logger *zap.Logger
}

/*
NewNodeService 创建节点服务
*/
func NewNodeService() *NodeService {
	return &NodeService{
		logger: zap.L().Named("node-service"),
	}
}

/*
NewNodeServiceWithDB 创建带数据库连接的节点服务
*/
func NewNodeServiceWithDB(db *sql.DB) *NodeService {
	return &NodeService{
		db:     db,
		logger: zap.L().Named("node-service"),
	}
}

/*
GetNode 获取节点
功能：根据 ID 查询节点详细信息
*/
func (s *NodeService) GetNode(id string) (*struct{ Status string }, error) {
	if s.db == nil {
		return &struct{ Status string }{Status: "online"}, nil
	}

	var status string
	err := s.db.QueryRow(`SELECT status FROM nodes WHERE id = ?`, id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("节点不存在: %s", id)
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}

	return &struct{ Status string }{Status: status}, nil
}

/*
ListNodes 列出所有节点
功能：从数据库查询全部节点列表
*/
func (s *NodeService) ListNodes() ([]*model.Node, error) {
	if s.db == nil {
		return []*model.Node{}, nil
	}

	rows, err := s.db.Query(`
		SELECT id, name, description, status, created_at, updated_at, last_online,
			hardware_id, system_info, ip_address, version, role
		FROM nodes ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询节点列表失败: %w", err)
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		var node model.Node
		if err := rows.Scan(
			&node.ID, &node.Name, &node.Description, &node.Status,
			&node.CreatedAt, &node.UpdatedAt, &node.LastOnline,
			&node.HardwareID, &node.SystemInfo, &node.IPAddress, &node.Version, &node.Role,
		); err != nil {
			return nil, fmt.Errorf("扫描节点数据失败: %w", err)
		}
		nodes = append(nodes, &node)
	}

	return nodes, nil
}

/*
CreateNode 创建节点
功能：将新节点信息写入数据库
*/
func (s *NodeService) CreateNode(node *model.Node) error {
	if node == nil {
		return errors.New("节点不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO nodes (id, name, description, status, created_at, updated_at, last_online,
			hardware_id, system_info, ip_address, version, role)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, node.ID, node.Name, node.Description, node.Status,
		now, now, now,
		node.HardwareID, node.SystemInfo, node.IPAddress, node.Version, node.Role)

	if err != nil {
		s.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("创建节点失败: %w", err)
	}

	s.logger.Info("创建节点成功", zap.String("id", node.ID), zap.String("name", node.Name))
	return nil
}

/*
UpdateNode 更新节点
功能：更新数据库中已有节点的配置信息
*/
func (s *NodeService) UpdateNode(node *model.Node) error {
	if node == nil {
		return errors.New("节点不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	_, err := s.db.Exec(`
		UPDATE nodes SET name = ?, description = ?, status = ?, updated_at = ?,
			hardware_id = ?, system_info = ?, ip_address = ?, version = ?, role = ?
		WHERE id = ?
	`, node.Name, node.Description, node.Status, time.Now(),
		node.HardwareID, node.SystemInfo, node.IPAddress, node.Version, node.Role,
		node.ID)

	if err != nil {
		s.logger.Error("更新节点失败", zap.String("id", node.ID), zap.Error(err))
		return fmt.Errorf("更新节点失败: %w", err)
	}

	s.logger.Info("更新节点成功", zap.String("id", node.ID))
	return nil
}

/*
GetNodeGroups 获取节点组列表（可选nodeID过滤）
功能：查询节点所属的分组，若不传 nodeID 则返回所有分组
*/
func (s *NodeService) GetNodeGroups(nodeID ...string) ([]*model.NodeGroup, error) {
	if s.db == nil {
		return []*model.NodeGroup{}, nil
	}

	if len(nodeID) > 0 && nodeID[0] != "" {
		return model.GetNodeGroups(s.db, nodeID[0])
	}
	return model.ListNodeGroups(s.db)
}

/*
DeleteNode 删除节点
功能：从数据库中删除节点及其分组关联
*/
func (s *NodeService) DeleteNode(id string) error {
	if id == "" {
		return errors.New("节点ID不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	/* 删除节点分组关联 */
	if _, err := tx.Exec(`DELETE FROM node_group_nodes WHERE node_id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点分组关联失败: %w", err)
	}

	/* 删除节点 */
	if _, err := tx.Exec(`DELETE FROM nodes WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("删除节点失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	s.logger.Info("删除节点成功", zap.String("id", id))
	return nil
}

/*
UpdateNodeStatus 更新节点状态
功能：更新节点的在线状态和最后在线时间
*/
func (s *NodeService) UpdateNodeStatus(id, status string) error {
	if id == "" {
		return errors.New("节点ID不能为空")
	}
	if s.db == nil {
		return errors.New("数据库未初始化")
	}

	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE nodes SET status = ?, last_online = ?, updated_at = ? WHERE id = ?
	`, status, now, now, id)

	if err != nil {
		s.logger.Error("更新节点状态失败", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("更新节点状态失败: %w", err)
	}

	return nil
}
