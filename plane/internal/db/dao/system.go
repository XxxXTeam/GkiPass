package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/* ==================== 系统设置 ==================== */

/*
GetSystemSetting 根据 key 获取系统设置
*/
func (d *DAO) GetSystemSetting(key string) (*models.SystemSetting, error) {
	var setting models.SystemSetting
	if err := d.DB.Where("`key` = ?", key).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

/*
UpsertSystemSetting 创建或更新系统设置
*/
func (d *DAO) UpsertSystemSetting(setting *models.SystemSetting) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "category", "type"}),
	}).Create(setting).Error
}

/*
ListSystemSettings 按分类列出系统设置
*/
func (d *DAO) ListSystemSettings(category string) ([]models.SystemSetting, error) {
	var settings []models.SystemSetting
	q := d.DB.Model(&models.SystemSetting{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Order("`key` ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

/*
DeleteSystemSetting 删除系统设置
*/
func (d *DAO) DeleteSystemSetting(key string) error {
	return d.DB.Where("`key` = ?", key).Delete(&models.SystemSetting{}).Error
}

/* ==================== 公告管理 ==================== */

/*
CreateAnnouncement 创建公告
*/
func (d *DAO) CreateAnnouncement(a *models.Announcement) error {
	return d.DB.Create(a).Error
}

/*
GetAnnouncement 获取公告
*/
func (d *DAO) GetAnnouncement(id string) (*models.Announcement, error) {
	var a models.Announcement
	if err := d.DB.First(&a, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

/*
UpdateAnnouncement 更新公告
*/
func (d *DAO) UpdateAnnouncement(a *models.Announcement) error {
	return d.DB.Save(a).Error
}

/*
DeleteAnnouncement 删除公告
*/
func (d *DAO) DeleteAnnouncement(id string) error {
	return d.DB.Delete(&models.Announcement{}, "id = ?", id).Error
}

/*
ListAnnouncements 列出所有公告（管理员）
*/
func (d *DAO) ListAnnouncements(page, pageSize int) ([]models.Announcement, int64, error) {
	var announcements []models.Announcement
	var total int64

	d.DB.Model(&models.Announcement{}).Count(&total)

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if err := d.DB.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&announcements).Error; err != nil {
		return nil, 0, err
	}
	return announcements, total, nil
}

/*
ListActiveAnnouncements 列出有效公告（用户可见）
*/
func (d *DAO) ListActiveAnnouncements() ([]models.Announcement, error) {
	var announcements []models.Announcement
	now := time.Now()
	if err := d.DB.Where("enabled = true AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", now, now).
		Order("priority DESC, created_at DESC").
		Find(&announcements).Error; err != nil {
		return nil, err
	}
	return announcements, nil
}

/* ==================== 通知管理 ==================== */

/*
CreateNotification 创建通知
*/
func (d *DAO) CreateNotification(n *models.Notification) error {
	return d.DB.Create(n).Error
}

/*
GetNotification 获取通知
*/
func (d *DAO) GetNotification(id string) (*models.Notification, error) {
	var n models.Notification
	if err := d.DB.First(&n, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

/*
ListNotifications 获取用户通知列表
*/
func (d *DAO) ListNotifications(userID string, page, pageSize int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	q := d.DB.Model(&models.Notification{}).Where("user_id = ?", userID)
	q.Count(&total)

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&notifications).Error; err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

/*
MarkNotificationAsRead 标记通知为已读
*/
func (d *DAO) MarkNotificationAsRead(id string) error {
	now := time.Now()
	return d.DB.Model(&models.Notification{}).Where("id = ?", id).
		Updates(map[string]interface{}{"read": true, "read_at": &now}).Error
}

/*
MarkAllNotificationsAsRead 标记用户所有通知为已读
*/
func (d *DAO) MarkAllNotificationsAsRead(userID string) error {
	now := time.Now()
	return d.DB.Model(&models.Notification{}).
		Where("user_id = ? AND `read` = false", userID).
		Updates(map[string]interface{}{"read": true, "read_at": &now}).Error
}

/*
DeleteNotification 删除通知
*/
func (d *DAO) DeleteNotification(id string) error {
	return d.DB.Delete(&models.Notification{}, "id = ?", id).Error
}

/* ==================== 支付配置 ==================== */

/*
GetPaymentConfig 获取支付配置
*/
func (d *DAO) GetPaymentConfig(id string) (*models.PaymentConfig, error) {
	var config models.PaymentConfig
	if err := d.DB.First(&config, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

/*
ListPaymentConfigs 列出支付配置
*/
func (d *DAO) ListPaymentConfigs() ([]models.PaymentConfig, error) {
	var configs []models.PaymentConfig
	if err := d.DB.Order("sort_order ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

/*
UpdatePaymentConfig 更新支付配置
*/
func (d *DAO) UpdatePaymentConfig(config *models.PaymentConfig) error {
	return d.DB.Save(config).Error
}
