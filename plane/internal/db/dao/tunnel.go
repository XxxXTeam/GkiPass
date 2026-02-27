package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 隧道 CRUD ==================== */

/*
GetTunnel 获取隧道
*/
func (d *DAO) GetTunnel(id string) (*models.Tunnel, error) {
	var tunnel models.Tunnel
	if err := d.DB.First(&tunnel, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tunnel, nil
}

/*
ListTunnels 列出隧道
参数 userID 可选过滤创建者
*/
func (d *DAO) ListTunnels(userID string, limit, offset int) ([]models.Tunnel, int64, error) {
	limit, offset = SanitizePagination(limit, offset, 500)
	var tunnels []models.Tunnel
	var total int64

	q := d.DB.Model(&models.Tunnel{})
	if userID != "" {
		q = q.Where("created_by = ?", userID)
	}
	q.Count(&total)
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&tunnels).Error; err != nil {
		return nil, 0, err
	}
	return tunnels, total, nil
}

/*
CreateTunnel 创建隧道
*/
func (d *DAO) CreateTunnel(tunnel *models.Tunnel) error {
	return d.DB.Create(tunnel).Error
}

/*
UpdateTunnel 更新隧道
*/
func (d *DAO) UpdateTunnel(tunnel *models.Tunnel) error {
	return d.DB.Save(tunnel).Error
}

/*
DeleteTunnel 删除隧道
*/
func (d *DAO) DeleteTunnel(id string) error {
	return d.DB.Delete(&models.Tunnel{}, "id = ?", id).Error
}

/* ==================== 流量统计 ==================== */

/*
CreateTrafficStats 创建流量统计记录
*/
func (d *DAO) CreateTrafficStats(stats *models.TrafficStats) error {
	return d.DB.Create(stats).Error
}

/*
ListTrafficStats 列出流量统计
*/
func (d *DAO) ListTrafficStats(userID, tunnelID string, limit, offset int) ([]models.TrafficStats, int64, error) {
	limit, offset = SanitizePagination(limit, offset, 200)
	var stats []models.TrafficStats
	var total int64

	q := d.DB.Model(&models.TrafficStats{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if tunnelID != "" {
		q = q.Where("tunnel_id = ?", tunnelID)
	}
	q.Count(&total)
	if err := q.Order("start_at DESC").Offset(offset).Limit(limit).Find(&stats).Error; err != nil {
		return nil, 0, err
	}
	return stats, total, nil
}

/*
GetTrafficSummary 获取流量汇总
返回：totalBytesIn, totalBytesOut, error
*/
func (d *DAO) GetTrafficSummary(userID, tunnelID string, startDate, endDate time.Time) (int64, int64, error) {
	var result struct {
		TotalIn  int64
		TotalOut int64
	}

	q := d.DB.Model(&models.TrafficStats{}).
		Select("COALESCE(SUM(bytes_in),0) as total_in, COALESCE(SUM(bytes_out),0) as total_out")

	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if tunnelID != "" {
		q = q.Where("tunnel_id = ?", tunnelID)
	}
	q = q.Where("start_at >= ? AND end_at <= ?", startDate, endDate)

	if err := q.Scan(&result).Error; err != nil {
		return 0, 0, err
	}
	return result.TotalIn, result.TotalOut, nil
}
