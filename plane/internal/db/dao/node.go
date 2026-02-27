package dao

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 节点 CRUD ==================== */

/*
CreateNode 创建节点
*/
func (d *DAO) CreateNode(node *models.Node) error {
	return d.DB.Create(node).Error
}

/*
GetNode 获取节点
*/
func (d *DAO) GetNode(id string) (*models.Node, error) {
	var node models.Node
	if err := d.DB.First(&node, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}

/*
GetNodeByAPIKey 通过 API Key 获取节点
*/
func (d *DAO) GetNodeByAPIKey(apiKey string) (*models.Node, error) {
	var node models.Node
	if err := d.DB.Where("api_key = ?", apiKey).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}

/*
GetNodeByToken 通过 Token 获取节点
*/
func (d *DAO) GetNodeByToken(token string) (*models.Node, error) {
	var node models.Node
	if err := d.DB.Where("token = ?", token).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}

/*
ListNodes 列出节点
参数 groupID 可选过滤节点组，status 可选过滤状态
*/
func (d *DAO) ListNodes(groupID, status string, limit, offset int) ([]models.Node, error) {
	limit, offset = SanitizePagination(limit, offset, 1000)
	var nodes []models.Node
	q := d.DB.Model(&models.Node{})

	if groupID != "" {
		/* 通过多对多关联查询 */
		q = q.Joins("JOIN node_group_nodes ON node_group_nodes.node_id = nodes.id").
			Where("node_group_nodes.node_group_id = ?", groupID)
	}
	if status != "" {
		q = q.Where("nodes.status = ?", status)
	}

	if err := q.Order("nodes.created_at DESC").Limit(limit).Offset(offset).Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

/*
UpdateNode 更新节点
*/
func (d *DAO) UpdateNode(node *models.Node) error {
	return d.DB.Save(node).Error
}

/*
UpdateNodeStatus 更新节点状态
*/
func (d *DAO) UpdateNodeStatus(id string, status models.NodeStatus) error {
	updates := map[string]interface{}{"status": status}
	if status == models.NodeStatusOnline {
		updates["last_online"] = time.Now()
	}
	return d.DB.Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error
}

/*
DeleteNode 删除节点（软删除）
*/
func (d *DAO) DeleteNode(id string) error {
	result := d.DB.Delete(&models.Node{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("节点不存在")
	}
	return nil
}

/*
CountNodes 统计节点数量
*/
func (d *DAO) CountNodes(status string) (int64, error) {
	var count int64
	q := d.DB.Model(&models.Node{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	return count, q.Count(&count).Error
}

/* ==================== 节点组 CRUD ==================== */

/*
CreateNodeGroup 创建节点组
*/
func (d *DAO) CreateNodeGroup(group *models.NodeGroup) error {
	return d.DB.Create(group).Error
}

/*
GetNodeGroup 获取节点组
*/
func (d *DAO) GetNodeGroup(id string) (*models.NodeGroup, error) {
	var group models.NodeGroup
	if err := d.DB.First(&group, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

/*
GetNodeGroupWithNodes 获取节点组及其节点
*/
func (d *DAO) GetNodeGroupWithNodes(id string) (*models.NodeGroup, error) {
	var group models.NodeGroup
	if err := d.DB.Preload("Nodes").First(&group, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

/*
ListNodeGroups 列出节点组
*/
func (d *DAO) ListNodeGroups(role string) ([]models.NodeGroup, error) {
	var groups []models.NodeGroup
	q := d.DB.Model(&models.NodeGroup{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if err := q.Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

/*
UpdateNodeGroup 更新节点组
*/
func (d *DAO) UpdateNodeGroup(group *models.NodeGroup) error {
	return d.DB.Save(group).Error
}

/*
DeleteNodeGroup 删除节点组（软删除）
*/
func (d *DAO) DeleteNodeGroup(id string) error {
	result := d.DB.Delete(&models.NodeGroup{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("节点组不存在")
	}
	return nil
}

/*
GetNodesInGroup 获取组内所有节点
*/
func (d *DAO) GetNodesInGroup(groupID string) ([]models.Node, error) {
	var group models.NodeGroup
	if err := d.DB.Preload("Nodes").First(&group, "id = ?", groupID).Error; err != nil {
		return nil, err
	}
	return group.Nodes, nil
}

/*
AddNodeToGroup 将节点加入组
*/
func (d *DAO) AddNodeToGroup(nodeID, groupID string) error {
	return d.DB.Exec("INSERT OR IGNORE INTO node_group_nodes (node_id, node_group_id) VALUES (?, ?)", nodeID, groupID).Error
}

/*
RemoveNodeFromGroup 将节点从组中移除
*/
func (d *DAO) RemoveNodeFromGroup(nodeID, groupID string) error {
	return d.DB.Exec("DELETE FROM node_group_nodes WHERE node_id = ? AND node_group_id = ?", nodeID, groupID).Error
}
