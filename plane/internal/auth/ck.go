package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"
)

// CKManager Connection Key 管理器
type CKManager struct {
}

// NewCKManager 创建 CK 管理器
func NewCKManager() *CKManager {
	return &CKManager{}
}

/*
GenerateCK 生成加密安全的 Connection Key
功能：使用 crypto/rand 生成 256 位熵的随机 CK，失败时返回 error 而非空字符串。
*/
func GenerateCK() (string, error) {
	bytes := make([]byte, 32) // 32字节 = 64位hex
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("CK 随机数生成失败: %w", err)
	}
	return "gkp_" + hex.EncodeToString(bytes), nil
}

// ValidateCK 验证 CK 格式
func ValidateCK(ck string) error {
	if len(ck) != 68 { // gkp_ + 64位hex
		return fmt.Errorf("CK 长度错误")
	}

	if ck[:4] != "gkp_" {
		return fmt.Errorf("CK 前缀错误")
	}

	// 验证是否为有效的hex
	_, err := hex.DecodeString(ck[4:])
	if err != nil {
		return fmt.Errorf("CK 格式错误: %w", err)
	}

	return nil
}

/* CreateNodeCK 为节点创建 CK，失败时返回 error 而非空 CK 对象 */
func CreateNodeCK(nodeID string, expiresIn time.Duration) (*models.ConnectionKey, error) {
	ck, err := GenerateCK()
	if err != nil {
		return nil, err
	}
	return &models.ConnectionKey{
		NodeID:    nodeID,
		Key:       ck,
		Type:      "node",
		Label:     "node-ck",
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

/* CreateUserCK 为用户创建 CK，失败时返回 error 而非空 CK 对象 */
func CreateUserCK(userID string, expiresIn time.Duration) (*models.ConnectionKey, error) {
	ck, err := GenerateCK()
	if err != nil {
		return nil, err
	}
	return &models.ConnectionKey{
		NodeID:    userID,
		Key:       ck,
		Type:      "user",
		Label:     "user-ck",
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

/* GenerateID 生成唯一 ID，失败时返回 error */
func GenerateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("ID 随机数生成失败: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// IsExpired 检查 CK 是否过期
func IsExpired(ck *models.ConnectionKey) bool {
	return time.Now().After(ck.ExpiresAt)
}
