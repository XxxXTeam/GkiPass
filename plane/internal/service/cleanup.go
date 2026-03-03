package service

import (
	"time"

	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/pkg/logger"

	"go.uber.org/zap"
)

/*
CleanupService 清理服务（定时任务）
功能：定期检查过期订阅、无效隧道，执行清理操作
*/
type CleanupService struct {
	dao      *dao.DAO
	stopChan chan struct{}
}

/*
NewCleanupService 创建清理服务
*/
func NewCleanupService(d *dao.DAO) *CleanupService {
	return &CleanupService{
		dao:      d,
		stopChan: make(chan struct{}),
	}
}

// Start 启动清理服务
func (s *CleanupService) Start() {
	s.cleanupExpiredSubscriptions()
	s.cleanupInactiveTunnels()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runCleanup()
		case <-s.stopChan:
			return
		}
	}
}

// Stop 停止清理服务
func (s *CleanupService) Stop() {
	close(s.stopChan)
}

// runCleanup 执行清理
func (s *CleanupService) runCleanup() {
	logger.Debug("执行定时清理任务")

	// 1. 清理过期订阅
	s.cleanupExpiredSubscriptions()

	// 2. 清理无效隧道
	s.cleanupInactiveTunnels()

	// 3. 重置流量（如果到达重置时间）
	s.resetTrafficIfNeeded()

	// 4. 清理过期加密密钥
	s.cleanupExpiredKeys()

	// 5. 清理旧监控数据（保留 7 天）
	s.cleanupOldMonitoringData()
}

/* cleanupExpiredSubscriptions 清理过期订阅 */
func (s *CleanupService) cleanupExpiredSubscriptions() {
	logger.Debug("检查过期订阅...")

	count, err := s.dao.ExpireSubscriptions()
	if err != nil {
		logger.Error("批量过期订阅失败", zap.Error(err))
		return
	}

	if count > 0 {
		logger.Info("已标记过期订阅", zap.Int64("count", count))
	}
}

/* cleanupInactiveTunnels 清理无效隧道（预留，待隧道配额系统完善后启用） */
func (s *CleanupService) cleanupInactiveTunnels() {
	logger.Debug("检查无效隧道...")
	/* TODO: 根据订阅状态批量禁用无有效订阅用户的隧道 */
}

// resetTrafficIfNeeded 重置流量（如果到达重置时间）
func (s *CleanupService) resetTrafficIfNeeded() {
	logger.Debug("检查流量重置...")
}

/* cleanupExpiredKeys 清理过期的隧道加密密钥 */
func (s *CleanupService) cleanupExpiredKeys() {
	result := s.dao.DB.
		Where("expires_at < ? AND active = ?", time.Now(), false).
		Delete(&TunnelEncryptionKey{})

	if result.Error != nil {
		logger.Error("清理过期密钥失败", zap.Error(result.Error))
		return
	}
	if result.RowsAffected > 0 {
		logger.Info("已清理过期加密密钥", zap.Int64("count", result.RowsAffected))
	}
}

/* cleanupOldMonitoringData 清理 7 天前的旧监控数据 */
func (s *CleanupService) cleanupOldMonitoringData() {
	cutoff := time.Now().AddDate(0, 0, -7)
	result := s.dao.DB.
		Where("created_at < ?", cutoff).
		Delete(&models.NodeMonitoringData{})

	if result.Error != nil {
		logger.Error("清理旧监控数据失败", zap.Error(result.Error))
		return
	}
	if result.RowsAffected > 0 {
		logger.Info("已清理旧监控数据", zap.Int64("count", result.RowsAffected))
	}
}
