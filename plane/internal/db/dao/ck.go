package dao

import (
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== ConnectionKey CRUD ==================== */

/*
GetCKByKey 根据 Key 获取连接密钥
功能：用于 CK 认证中间件验证密钥有效性
*/
func (d *DAO) GetCKByKey(key string) (*models.ConnectionKey, error) {
	var ck models.ConnectionKey
	if err := d.DB.Where("`key` = ? AND revoked = false AND expires_at > ?", key, time.Now()).First(&ck).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ck, nil
}

/*
GetCKByNodeID 根据节点 ID 获取连接密钥
*/
func (d *DAO) GetCKByNodeID(nodeID string) (*models.ConnectionKey, error) {
	var ck models.ConnectionKey
	if err := d.DB.Where("node_id = ? AND revoked = false AND expires_at > ?", nodeID, time.Now()).First(&ck).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ck, nil
}

/*
RevokeCK 撤销连接密钥
*/
func (d *DAO) RevokeCK(id string) error {
	return d.DB.Model(&models.ConnectionKey{}).Where("id = ?", id).Update("revoked", true).Error
}

/*
ListCKsByNodeID 获取节点的所有连接密钥
*/
func (d *DAO) ListCKsByNodeID(nodeID string) ([]models.ConnectionKey, error) {
	var cks []models.ConnectionKey
	if err := d.DB.Where("node_id = ?", nodeID).Order("created_at DESC").Find(&cks).Error; err != nil {
		return nil, err
	}
	return cks, nil
}
