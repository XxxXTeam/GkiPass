package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 节点监控指标 ==================== */

/*
CreateNodeMetrics 创建节点指标数据
*/
func (d *DAO) CreateNodeMetrics(m *models.NodeMetrics) error {
	return d.DB.Create(m).Error
}

/*
GetLatestNodeMetrics 获取节点最新监控数据
*/
func (d *DAO) GetLatestNodeMetrics(nodeID string) (*models.NodeMetrics, error) {
	var m models.NodeMetrics
	if err := d.DB.Where("node_id = ?", nodeID).Order("created_at DESC").First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

/*
ListNodeMetrics 获取节点监控历史数据
*/
func (d *DAO) ListNodeMetrics(nodeID string, from, to time.Time, limit int) ([]models.NodeMetrics, error) {
	var metrics []models.NodeMetrics
	q := d.DB.Where("node_id = ?", nodeID)
	if !from.IsZero() {
		q = q.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("created_at <= ?", to)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Order("created_at DESC").Find(&metrics).Error; err != nil {
		return nil, err
	}
	return metrics, nil
}

/*
DeleteOldMetrics 删除过期监控数据
*/
func (d *DAO) DeleteOldMetrics(before time.Time) (int64, error) {
	result := d.DB.Where("created_at < ?", before).Delete(&models.NodeMetrics{})
	return result.RowsAffected, result.Error
}

/* ==================== 订单管理 ==================== */

/*
CreateOrder 创建订单
*/
func (d *DAO) CreateOrder(order *models.Order) error {
	return d.DB.Create(order).Error
}

/*
GetOrder 获取订单
*/
func (d *DAO) GetOrder(id string) (*models.Order, error) {
	var order models.Order
	if err := d.DB.First(&order, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

/*
GetOrderByExternalID 通过外部ID获取订单
*/
func (d *DAO) GetOrderByExternalID(externalID string) (*models.Order, error) {
	var order models.Order
	if err := d.DB.Where("external_id = ?", externalID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

/*
GetOrderByUser 获取用户的指定订单
*/
func (d *DAO) GetOrderByUser(id, userID string) (*models.Order, error) {
	var order models.Order
	if err := d.DB.Where("id = ? AND user_id = ?", id, userID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

/*
UpdateOrderStatus 更新订单状态
*/
func (d *DAO) UpdateOrderStatus(id, status string, externalID ...string) error {
	updates := map[string]interface{}{"status": status}
	if status == "completed" || status == "paid" {
		now := time.Now()
		updates["paid_at"] = &now
	}
	if len(externalID) > 0 && externalID[0] != "" {
		updates["external_id"] = externalID[0]
	}
	return d.DB.Model(&models.Order{}).Where("id = ?", id).Updates(updates).Error
}

/*
ListOrders 列出用户订单
*/
func (d *DAO) ListOrders(userID string, page, pageSize int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	q := d.DB.Model(&models.Order{}).Where("user_id = ?", userID)
	q.Count(&total)

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

/* ==================== 审计日志 ==================== */

/*
CreateAuditLog 创建审计日志
*/
func (d *DAO) CreateAuditLog(log *models.AuditLog) error {
	return d.DB.Create(log).Error
}
