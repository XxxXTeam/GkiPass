package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gkipass/plane/internal/db"
	"gkipass/plane/internal/pkg/logger"

	"go.uber.org/zap"
)

const (
	// JWT 密钥在 Redis 中的键名
	JWTSecretRedisKey = "system:jwt:secret"
	// JWT 密钥刷新间隔（24小时）
	JWTSecretRefreshInterval = 24 * time.Hour
	// JWT 密钥长度（64字节 = 512位）
	JWTSecretLength = 64
)

// JWTManager JWT 密钥管理器
type JWTManager struct {
	db            *db.Manager
	currentSecret string
	stopChan      chan struct{}
}

// NewJWTManager 创建 JWT 密钥管理器
func NewJWTManager(dbManager *db.Manager) *JWTManager {
	return &JWTManager{
		db:       dbManager,
		stopChan: make(chan struct{}),
	}
}

// Start 启动 JWT 密钥管理器
func (m *JWTManager) Start() error {

	// 初始化或加载 JWT 密钥
	secret, err := m.loadOrGenerateSecret()
	if err != nil {
		return fmt.Errorf("初始化 JWT 密钥失败: %w", err)
	}
	m.currentSecret = secret
	// 启动定期刷新协程
	go m.refreshLoop()
	return nil
}

// Stop 停止 JWT 密钥管理器
func (m *JWTManager) Stop() {
	close(m.stopChan)
}

func (m *JWTManager) GetSecret() string {
	return m.currentSecret
}

// loadOrGenerateSecret 加载或生成新的 JWT 密钥
func (m *JWTManager) loadOrGenerateSecret() (string, error) {
	// 检查 Redis 是否可用
	if !m.db.HasCache() {
		return m.generateSecret()
	}

	// 尝试从 Redis 加载
	var secret string
	err := m.db.Cache.Redis.Get(JWTSecretRedisKey, &secret)
	if err == nil && secret != "" {
		logger.Info("✓ 从 Redis 加载 JWT 密钥")
		return secret, nil
	}
	secret, err = m.generateSecret()
	if err != nil {
		return "", err
	}
	if err := m.saveSecret(secret); err != nil {
		logger.Warn("保存 JWT 密钥到 Redis 失败（将仅使用内存密钥）", zap.Error(err))
	}

	return secret, nil
}

// generateSecret 生成随机 JWT 密钥
func (m *JWTManager) generateSecret() (string, error) {
	bytes := make([]byte, JWTSecretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成随机密钥失败: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// saveSecret 保存 JWT 密钥到 Redis
func (m *JWTManager) saveSecret(secret string) error {
	if !m.db.HasCache() {
		return fmt.Errorf("Redis 不可用")
	}

	// 永久保存（不设置过期时间）
	return m.db.Cache.Redis.Set(JWTSecretRedisKey, secret, 0)
}

// refreshSecret 刷新 JWT 密钥
func (m *JWTManager) refreshSecret() error {

	newSecret, err := m.generateSecret()
	if err != nil {
		return fmt.Errorf("生成新密钥失败: %w", err)
	}

	// 保存到 Redis
	if err := m.saveSecret(newSecret); err != nil {
		return fmt.Errorf("保存新密钥失败: %w", err)
	}

	// 更新当前密钥
	m.currentSecret = newSecret
	logger.Info("✓ JWT 密钥已更新", zap.Int("length", len(newSecret)))

	return nil
}

/*
refreshLoop 后台密钥维护循环
功能：定期检查 Redis 中的密钥是否与内存一致（多实例同步），
但**不自动轮换密钥**——自动轮换会导致所有已发放的 JWT 立即失效，
用户被强制登出。密钥轮换应通过管理员手动触发。
*/
func (m *JWTManager) refreshLoop() {
	ticker := time.NewTicker(JWTSecretRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			/* 仅同步 Redis 中的密钥（多实例部署场景），不自动轮换 */
			if m.db.HasCache() {
				var redisSecret string
				if err := m.db.Cache.Redis.Get(JWTSecretRedisKey, &redisSecret); err == nil && redisSecret != "" {
					if redisSecret != m.currentSecret {
						m.currentSecret = redisSecret
						logger.Info("JWT 密钥已从 Redis 同步（多实例一致性）")
					}
				}
			}
		case <-m.stopChan:
			return
		}
	}
}

// GetJWTSecretFromRedis 从 Redis 获取 JWT 密钥（供外部使用）
func GetJWTSecretFromRedis(dbManager *db.Manager) (string, error) {
	if !dbManager.HasCache() {
		return "", fmt.Errorf("Redis 不可用")
	}

	var secret string
	err := dbManager.Cache.Redis.Get(JWTSecretRedisKey, &secret)
	if err != nil {
		return "", fmt.Errorf("获取 JWT 密钥失败: %w", err)
	}

	if secret == "" {
		return "", fmt.Errorf("JWT 密钥不存在")
	}

	return secret, nil
}
