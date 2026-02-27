package dao

import (
	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 策略 CRUD ==================== */

/*
CreatePolicy 创建策略
*/
func (d *DAO) CreatePolicy(p *models.Policy) error {
	return d.DB.Create(p).Error
}

/*
GetPolicy 获取策略
*/
func (d *DAO) GetPolicy(id string) (*models.Policy, error) {
	var p models.Policy
	if err := d.DB.First(&p, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

/*
ListPolicies 列出策略
参数 policyType 可选过滤类型，enabled 可选过滤启用状态
*/
func (d *DAO) ListPolicies(policyType string, enabled *bool) ([]models.Policy, error) {
	var policies []models.Policy
	q := d.DB.Model(&models.Policy{})
	if policyType != "" {
		q = q.Where("type = ?", policyType)
	}
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}
	if err := q.Order("priority DESC, created_at ASC").Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

/*
UpdatePolicy 更新策略
*/
func (d *DAO) UpdatePolicy(p *models.Policy) error {
	return d.DB.Save(p).Error
}

/*
DeletePolicy 删除策略
*/
func (d *DAO) DeletePolicy(id string) error {
	return d.DB.Delete(&models.Policy{}, "id = ?", id).Error
}

/* ==================== 节点组配置 CRUD ==================== */

/*
GetNodeGroupConfig 获取节点组配置
*/
func (d *DAO) GetNodeGroupConfig(groupID string) (*models.NodeGroupConfig, error) {
	var cfg models.NodeGroupConfig
	if err := d.DB.Where("group_id = ?", groupID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

/*
UpsertNodeGroupConfig 创建或更新节点组配置
*/
func (d *DAO) UpsertNodeGroupConfig(cfg *models.NodeGroupConfig) error {
	existing, err := d.GetNodeGroupConfig(cfg.GroupID)
	if err != nil {
		return err
	}
	if existing != nil {
		cfg.ID = existing.ID
		return d.DB.Save(cfg).Error
	}
	return d.DB.Create(cfg).Error
}
