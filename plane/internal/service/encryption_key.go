package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

/*
  EncryptionKeyService 加密密钥管理服务
  功能：为隧道生成、存储和分发加密密钥，
  支持 AES-128/192/256-GCM 和 ChaCha20-Poly1305 算法，
  自动管理密钥生命周期和轮换
*/
type EncryptionKeyService struct {
	db     *gorm.DB
	logger *zap.Logger
}

/*
  NewEncryptionKeyService 创建加密密钥管理服务
*/
func NewEncryptionKeyService(db *gorm.DB) *EncryptionKeyService {
	return &EncryptionKeyService{
		db:     db,
		logger: zap.L().Named("encryption-key-service"),
	}
}

/*
  TunnelEncryptionKey 隧道加密密钥模型
  功能：存储隧道的加密密钥和相关元数据
*/
type TunnelEncryptionKey struct {
	models.BaseModel
	TunnelID    string    `gorm:"type:varchar(36);index;not null" json:"tunnel_id"`
	Algorithm   string    `gorm:"type:varchar(32);not null" json:"algorithm"`
	KeyHex      string    `gorm:"type:varchar(128);not null" json:"-"`
	KeySize     int       `gorm:"not null" json:"key_size"`
	Version     int       `gorm:"default:1;not null" json:"version"`
	Active      bool      `gorm:"default:true;not null" json:"active"`
	ExpiresAt   time.Time `gorm:"index" json:"expires_at"`
	RotatedFrom string    `gorm:"type:varchar(36)" json:"rotated_from"`
}

func (TunnelEncryptionKey) TableName() string {
	return "tunnel_encryption_keys"
}

/*
  GenerateKeyForTunnel 为隧道生成加密密钥
  功能：根据隧道配置的加密算法生成对应长度的随机密钥
  支持的算法：aes-128-gcm, aes-192-gcm, aes-256-gcm, chacha20-poly1305
*/
func (s *EncryptionKeyService) GenerateKeyForTunnel(tunnelID string, algorithm string) (*TunnelEncryptionKey, error) {
	/* 确定密钥长度 */
	keySize, err := s.getKeySize(algorithm)
	if err != nil {
		return nil, err
	}

	/* 生成随机密钥 */
	keyBytes := make([]byte, keySize)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("生成随机密钥失败: %w", err)
	}

	/* 将旧密钥标记为不活跃 */
	if err := s.db.Model(&TunnelEncryptionKey{}).
		Where("tunnel_id = ? AND active = ?", tunnelID, true).
		Update("active", false).Error; err != nil {
		s.logger.Warn("停用旧密钥失败", zap.Error(err))
	}

	/* 创建新密钥记录 */
	key := &TunnelEncryptionKey{
		TunnelID:  tunnelID,
		Algorithm: algorithm,
		KeyHex:    hex.EncodeToString(keyBytes),
		KeySize:   keySize,
		Version:   1,
		Active:    true,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), /* 30天后过期 */
	}

	/* 查询当前最大版本号 */
	var maxVersion int
	s.db.Model(&TunnelEncryptionKey{}).
		Where("tunnel_id = ?", tunnelID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)
	key.Version = maxVersion + 1

	if err := s.db.Create(key).Error; err != nil {
		return nil, fmt.Errorf("存储加密密钥失败: %w", err)
	}

	s.logger.Info("隧道加密密钥已生成",
		zap.String("tunnel_id", tunnelID),
		zap.String("algorithm", algorithm),
		zap.Int("version", key.Version))

	return key, nil
}

/*
  GetActiveKey 获取隧道当前活跃的加密密钥
  功能：查询指定隧道的当前活跃密钥，用于节点端加密通信
*/
func (s *EncryptionKeyService) GetActiveKey(tunnelID string) (*TunnelEncryptionKey, error) {
	var key TunnelEncryptionKey
	err := s.db.
		Where("tunnel_id = ? AND active = ? AND expires_at > ?", tunnelID, true, time.Now()).
		Order("version DESC").
		First(&key).Error

	if err != nil {
		return nil, fmt.Errorf("未找到隧道 %s 的活跃密钥", tunnelID)
	}

	return &key, nil
}

/*
  GetKeyBytes 获取密钥的原始字节
  功能：将十六进制存储的密钥解码为字节数组
*/
func (s *EncryptionKeyService) GetKeyBytes(key *TunnelEncryptionKey) ([]byte, error) {
	return hex.DecodeString(key.KeyHex)
}

/*
  RotateKey 密钥轮换
  功能：生成新密钥替换旧密钥，记录轮换来源以支持审计
*/
func (s *EncryptionKeyService) RotateKey(tunnelID string) (*TunnelEncryptionKey, error) {
	/* 获取当前活跃密钥 */
	currentKey, err := s.GetActiveKey(tunnelID)
	if err != nil {
		/* 没有活跃密钥时使用默认算法创建 */
		return s.GenerateKeyForTunnel(tunnelID, "aes-256-gcm")
	}

	/* 生成新密钥 */
	newKey, err := s.GenerateKeyForTunnel(tunnelID, currentKey.Algorithm)
	if err != nil {
		return nil, err
	}

	/* 记录轮换来源 */
	newKey.RotatedFrom = currentKey.ID
	if err := s.db.Save(newKey).Error; err != nil {
		return nil, fmt.Errorf("更新密钥轮换记录失败: %w", err)
	}

	s.logger.Info("密钥轮换完成",
		zap.String("tunnel_id", tunnelID),
		zap.String("old_key_id", currentKey.ID),
		zap.String("new_key_id", newKey.ID))

	return newKey, nil
}

/*
  EnsureKeyForTunnel 确保隧道拥有加密密钥
  功能：如果隧道启用了加密但没有密钥，自动生成一个
*/
func (s *EncryptionKeyService) EnsureKeyForTunnel(tunnel *models.Tunnel) (*TunnelEncryptionKey, error) {
	if !tunnel.EnableEncryption {
		return nil, nil /* 未启用加密，无需密钥 */
	}

	/* 尝试获取现有活跃密钥 */
	key, err := s.GetActiveKey(tunnel.ID)
	if err == nil {
		return key, nil /* 已有活跃密钥 */
	}

	/* 生成新密钥 */
	algorithm := tunnel.EncryptionMethod
	if algorithm == "" {
		algorithm = "aes-256-gcm"
	}

	return s.GenerateKeyForTunnel(tunnel.ID, algorithm)
}

/*
  CleanExpiredKeys 清理过期密钥
  功能：删除已过期且非活跃的密钥记录，保留最近 5 个版本
*/
func (s *EncryptionKeyService) CleanExpiredKeys() (int64, error) {
	result := s.db.
		Where("expires_at < ? AND active = ?", time.Now(), false).
		Delete(&TunnelEncryptionKey{})

	if result.Error != nil {
		return 0, fmt.Errorf("清理过期密钥失败: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		s.logger.Info("已清理过期密钥", zap.Int64("count", result.RowsAffected))
	}

	return result.RowsAffected, nil
}

/*
  getKeySize 根据算法返回密钥长度（字节）
*/
func (s *EncryptionKeyService) getKeySize(algorithm string) (int, error) {
	switch algorithm {
	case "aes-128-gcm":
		return 16, nil
	case "aes-192-gcm":
		return 24, nil
	case "aes-256-gcm":
		return 32, nil
	case "chacha20-poly1305":
		return 32, nil
	default:
		return 0, fmt.Errorf("不支持的加密算法: %s, 支持: aes-128-gcm, aes-192-gcm, aes-256-gcm, chacha20-poly1305", algorithm)
	}
}
