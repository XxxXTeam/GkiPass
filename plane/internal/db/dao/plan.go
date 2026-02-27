package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 套餐管理 ==================== */

/*
CreatePlan 创建套餐
*/
func (d *DAO) CreatePlan(plan *models.Plan) error {
	return d.DB.Create(plan).Error
}

/*
GetPlan 获取套餐
*/
func (d *DAO) GetPlan(id string) (*models.Plan, error) {
	var plan models.Plan
	if err := d.DB.First(&plan, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

/*
ListPlans 列出套餐
*/
func (d *DAO) ListPlans(enabledOnly bool) ([]models.Plan, error) {
	var plans []models.Plan
	q := d.DB.Order("sort_order ASC, created_at ASC")
	if enabledOnly {
		q = q.Where("enabled = true")
	}
	if err := q.Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

/*
UpdatePlan 更新套餐
*/
func (d *DAO) UpdatePlan(plan *models.Plan) error {
	return d.DB.Save(plan).Error
}

/*
DeletePlan 删除套餐（软删除）
*/
func (d *DAO) DeletePlan(id string) error {
	return d.DB.Delete(&models.Plan{}, "id = ?", id).Error
}

/* ==================== 订阅管理 ==================== */

/*
CreateSubscription 创建订阅
*/
func (d *DAO) CreateSubscription(sub *models.Subscription) error {
	return d.DB.Create(sub).Error
}

/*
GetActiveSubscription 获取用户活跃订阅
*/
func (d *DAO) GetActiveSubscription(userID string) (*models.Subscription, error) {
	var sub models.Subscription
	if err := d.DB.Preload("Plan").
		Where("user_id = ? AND status = 'active' AND expire_at > ?", userID, time.Now()).
		Order("created_at DESC").First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

/*
GetSubscriptionsByPlanID 获取套餐的所有活跃订阅
*/
func (d *DAO) GetSubscriptionsByPlanID(planID string) ([]models.Subscription, error) {
	var subs []models.Subscription
	if err := d.DB.Where("plan_id = ? AND status = 'active'", planID).Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

/*
UpdateSubscription 更新订阅
*/
func (d *DAO) UpdateSubscription(sub *models.Subscription) error {
	return d.DB.Save(sub).Error
}

/*
ExpireSubscriptions 批量过期订阅
*/
func (d *DAO) ExpireSubscriptions() (int64, error) {
	result := d.DB.Model(&models.Subscription{}).
		Where("status = 'active' AND expire_at <= ?", time.Now()).
		Update("status", "expired")
	return result.RowsAffected, result.Error
}

/*
ListSubscriptions 管理员分页列出所有订阅
*/
func (d *DAO) ListSubscriptions(limit, offset int) ([]models.Subscription, int64, error) {
	limit, offset = SanitizePagination(limit, offset, 200)
	var subs []models.Subscription
	var total int64
	if err := d.DB.Model(&models.Subscription{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := d.DB.Preload("Plan").Preload("User").
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&subs).Error; err != nil {
		return nil, 0, err
	}
	return subs, total, nil
}

/*
CountActiveSubscriptions 统计活跃订阅数
*/
func (d *DAO) CountActiveSubscriptions() (int64, error) {
	var count int64
	err := d.DB.Model(&models.Subscription{}).
		Where("status = 'active' AND expire_at > ?", time.Now()).
		Count(&count).Error
	return count, err
}

/*
ListExpiredSubscriptions 列出刚过期的订阅
*/
func (d *DAO) ListExpiredSubscriptions() ([]models.Subscription, error) {
	var subs []models.Subscription
	if err := d.DB.Where("status = 'expired'").
		Order("updated_at DESC").Limit(100).Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}
