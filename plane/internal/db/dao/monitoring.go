package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ==================== 监控配置 ==================== */

/*
GetNodeMonitoringConfig 获取节点监控配置
功能：根据节点ID获取监控配置，不存在返回 nil
*/
func (d *DAO) GetNodeMonitoringConfig(nodeID string) (*models.NodeMonitoringConfig, error) {
	var cfg models.NodeMonitoringConfig
	if err := d.DB.Where("node_id = ?", nodeID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

/*
UpsertNodeMonitoringConfig 创建或更新节点监控配置
功能：如果配置存在则更新，否则新建
*/
func (d *DAO) UpsertNodeMonitoringConfig(cfg *models.NodeMonitoringConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	return d.DB.Save(cfg).Error
}

/* ==================== 监控数据 ==================== */

/*
CreateNodeMonitoringData 存储节点监控数据
功能：写入一条实时监控数据记录
*/
func (d *DAO) CreateNodeMonitoringData(data *models.NodeMonitoringData) error {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}
	return d.DB.Create(data).Error
}

/*
GetLatestNodeMonitoringData 获取节点最新监控数据
功能：按时间倒序取第一条
*/
func (d *DAO) GetLatestNodeMonitoringData(nodeID string) (*models.NodeMonitoringData, error) {
	var data models.NodeMonitoringData
	if err := d.DB.Where("node_id = ?", nodeID).Order("timestamp DESC").First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &data, nil
}

/*
ListNodeMonitoringData 列出时间范围内的监控数据
功能：分页查询指定节点在 [from, to] 时间段内的监控数据
*/
func (d *DAO) ListNodeMonitoringData(nodeID string, from, to time.Time, limit int) ([]*models.NodeMonitoringData, error) {
	var list []*models.NodeMonitoringData
	err := d.DB.Where("node_id = ? AND timestamp >= ? AND timestamp < ?", nodeID, from, to).
		Order("timestamp ASC").Limit(limit).Find(&list).Error
	return list, err
}

/*
DeleteOldMonitoringData 清理过期监控数据
功能：删除指定节点超过 retentionDays 天的旧数据
*/
func (d *DAO) DeleteOldMonitoringData(nodeID string, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return d.DB.Where("node_id = ? AND timestamp < ?", nodeID, cutoff).Delete(&models.NodeMonitoringData{}).Error
}

/* ==================== 性能历史 ==================== */

/*
CreateNodePerformanceHistory 创建性能历史记录
功能：写入一条聚合后的性能趋势数据
*/
func (d *DAO) CreateNodePerformanceHistory(h *models.NodePerformanceHistory) error {
	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	return d.DB.Create(h).Error
}

/* ==================== 告警规则 ==================== */

/*
ListNodeAlertRules 列出节点的告警规则
功能：获取指定节点的所有告警规则
*/
func (d *DAO) ListNodeAlertRules(nodeID string) ([]*models.NodeAlertRule, error) {
	var rules []*models.NodeAlertRule
	err := d.DB.Where("node_id = ?", nodeID).Find(&rules).Error
	return rules, err
}

/*
CreateNodeAlertRule 创建告警规则
功能：为指定节点新建一条告警规则
*/
func (d *DAO) CreateNodeAlertRule(rule *models.NodeAlertRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	return d.DB.Create(rule).Error
}

/*
GetNodeAlertRule 获取单条告警规则
功能：根据规则 ID 查询
*/
func (d *DAO) GetNodeAlertRule(id string) (*models.NodeAlertRule, error) {
	var rule models.NodeAlertRule
	if err := d.DB.First(&rule, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

/*
UpdateNodeAlertRule 更新告警规则
*/
func (d *DAO) UpdateNodeAlertRule(rule *models.NodeAlertRule) error {
	return d.DB.Save(rule).Error
}

/*
DeleteNodeAlertRule 删除告警规则
*/
func (d *DAO) DeleteNodeAlertRule(id string) error {
	return d.DB.Delete(&models.NodeAlertRule{}, "id = ?", id).Error
}

/* ==================== 告警历史 ==================== */

/*
CreateNodeAlertHistory 创建告警历史记录
功能：写入一条告警触发记录
*/
func (d *DAO) CreateNodeAlertHistory(alert *models.NodeAlertHistory) error {
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	return d.DB.Create(alert).Error
}

/*
ListNodeAlertHistory 列出节点的告警历史
功能：按触发时间倒序取 limit 条
*/
func (d *DAO) ListNodeAlertHistory(nodeID string, limit int) ([]*models.NodeAlertHistory, error) {
	var list []*models.NodeAlertHistory
	err := d.DB.Where("node_id = ?", nodeID).Order("triggered_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

/*
UpdateNodeAlertHistoryStatus 更新告警状态
功能：将告警标记为 acknowledged 或 resolved，并记录操作时间和操作人
*/
func (d *DAO) UpdateNodeAlertHistoryStatus(id, status, acknowledgedBy string) error {
	updates := map[string]interface{}{"status": status}
	now := time.Now()
	switch status {
	case "acknowledged":
		updates["acknowledged_at"] = &now
		updates["acknowledged_by"] = acknowledgedBy
	case "resolved":
		updates["resolved_at"] = &now
	}
	return d.DB.Model(&models.NodeAlertHistory{}).Where("id = ?", id).Updates(updates).Error
}

/* ==================== 监控权限 ==================== */

/*
GetMonitoringPermission 获取用户对节点的监控权限
功能：查询指定用户对指定节点的监控权限记录
*/
func (d *DAO) GetMonitoringPermission(userID, nodeID string) (*models.MonitoringPermission, error) {
	var perm models.MonitoringPermission
	if err := d.DB.Where("user_id = ? AND node_id = ?", userID, nodeID).First(&perm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &perm, nil
}

/*
CreateMonitoringPermission 创建监控权限
*/
func (d *DAO) CreateMonitoringPermission(perm *models.MonitoringPermission) error {
	if perm.ID == "" {
		perm.ID = uuid.New().String()
	}
	return d.DB.Create(perm).Error
}

/*
ListMonitoringPermissions 列出监控权限
功能：按用户ID筛选，为空则返回全部
*/
func (d *DAO) ListMonitoringPermissions(userID string) ([]*models.MonitoringPermission, error) {
	var list []*models.MonitoringPermission
	q := d.DB.Model(&models.MonitoringPermission{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Order("created_at DESC").Find(&list).Error
	return list, err
}

/*
UpdateMonitoringPermission 更新监控权限
*/
func (d *DAO) UpdateMonitoringPermission(perm *models.MonitoringPermission) error {
	return d.DB.Save(perm).Error
}

/*
DeleteMonitoringPermission 删除监控权限
*/
func (d *DAO) DeleteMonitoringPermission(id string) error {
	return d.DB.Delete(&models.MonitoringPermission{}, "id = ?", id).Error
}

/*
GetNodePerformanceHistory 获取节点性能历史
功能：按聚合类型和时间范围查询
*/
func (d *DAO) GetNodePerformanceHistory(nodeID, aggType string, from, to time.Time) ([]*models.NodePerformanceHistory, error) {
	var list []*models.NodePerformanceHistory
	err := d.DB.Where("node_id = ? AND aggregation_type = ? AND date >= ? AND date <= ?", nodeID, aggType, from, to).
		Order("aggregation_time ASC").Find(&list).Error
	return list, err
}
