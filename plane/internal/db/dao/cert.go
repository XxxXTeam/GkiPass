package dao

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/gorm"
)

/* ==================== 证书管理 ==================== */

/*
CreateCertificate 创建证书
*/
func (d *DAO) CreateCertificate(cert *models.NodeCertificate) error {
	return d.DB.Create(cert).Error
}

/*
GetCertificate 获取证书
*/
func (d *DAO) GetCertificate(id string) (*models.NodeCertificate, error) {
	var cert models.NodeCertificate
	if err := d.DB.First(&cert, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

/*
GetCertByNodeID 获取节点的证书
*/
func (d *DAO) GetCertByNodeID(nodeID string) (*models.NodeCertificate, error) {
	var cert models.NodeCertificate
	if err := d.DB.Where("node_id = ? AND revoked = false", nodeID).
		Order("created_at DESC").First(&cert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

/*
ListCertificates 列出证书
*/
func (d *DAO) ListCertificates(certType string, revoked *bool) ([]models.NodeCertificate, error) {
	var certs []models.NodeCertificate
	q := d.DB.Model(&models.NodeCertificate{})
	if certType != "" {
		q = q.Where("type = ?", certType)
	}
	if revoked != nil {
		q = q.Where("revoked = ?", *revoked)
	}
	if err := q.Order("created_at DESC").Find(&certs).Error; err != nil {
		return nil, err
	}
	return certs, nil
}

/*
ListCertificatesByNode 列出节点的所有证书
*/
func (d *DAO) ListCertificatesByNode(nodeID string) ([]models.NodeCertificate, error) {
	var certs []models.NodeCertificate
	if err := d.DB.Where("node_id = ?", nodeID).Order("created_at DESC").Find(&certs).Error; err != nil {
		return nil, err
	}
	return certs, nil
}

/*
RevokeCertificate 吊销证书
*/
func (d *DAO) RevokeCertificate(id string) error {
	result := d.DB.Model(&models.NodeCertificate{}).Where("id = ?", id).Update("revoked", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("证书不存在")
	}
	return nil
}

/* ==================== Connection Key 管理 ==================== */

/*
CreateConnectionKey 创建 CK
*/
func (d *DAO) CreateConnectionKey(ck *models.ConnectionKey) error {
	return d.DB.Create(ck).Error
}

/*
GetConnectionKeyByKey 根据 Key 获取 CK
*/
func (d *DAO) GetConnectionKeyByKey(key string) (*models.ConnectionKey, error) {
	var ck models.ConnectionKey
	if err := d.DB.Where("`key` = ? AND revoked = false AND expires_at > ?", key, time.Now()).
		First(&ck).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ck, nil
}

/*
GetConnectionKeysByNodeID 获取节点的所有 CK
*/
func (d *DAO) GetConnectionKeysByNodeID(nodeID string) ([]models.ConnectionKey, error) {
	var cks []models.ConnectionKey
	if err := d.DB.Where("node_id = ?", nodeID).Order("created_at DESC").Find(&cks).Error; err != nil {
		return nil, err
	}
	return cks, nil
}

/*
RevokeConnectionKey 撤销 CK
*/
func (d *DAO) RevokeConnectionKey(id string) error {
	return d.DB.Model(&models.ConnectionKey{}).Where("id = ?", id).Update("revoked", true).Error
}

/*
DeleteExpiredConnectionKeys 删除过期的 CK
*/
func (d *DAO) DeleteExpiredConnectionKeys() error {
	return d.DB.Where("expires_at < ?", time.Now()).Delete(&models.ConnectionKey{}).Error
}
