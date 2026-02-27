package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

/*
  Manager 数据库管理器
  功能：统一管理 GORM 数据库连接和 Redis 缓存连接，
  提供初始化、迁移、关闭等生命周期管理
*/
type Manager struct {
	DB    *gorm.DB
	Redis *RedisClient

	dbConfig    *Config
	redisConfig *RedisConfig
}

/*
  ManagerConfig 管理器配置
  功能：聚合数据库和 Redis 的配置信息
*/
type ManagerConfig struct {
	Database *Config      `yaml:"database" json:"database"`
	Redis    *RedisConfig `yaml:"redis" json:"redis"`
}

/*
  DefaultManagerConfig 返回默认管理器配置
  功能：提供开箱即用的数据库和 Redis 配置
*/
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		Database: DefaultConfig(),
		Redis:    DefaultRedisConfig(),
	}
}

/*
  NewManager 创建数据库管理器
  功能：初始化数据库和 Redis 连接，自动执行数据库迁移
*/
func NewManager(cfg *ManagerConfig) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultManagerConfig()
	}

	manager := &Manager{
		dbConfig:    cfg.Database,
		redisConfig: cfg.Redis,
	}

	/* 初始化数据库连接 */
	db, err := NewDatabase(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	manager.DB = db

	/* 自动迁移 */
	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	/* 初始化 Redis（可选组件） */
	if cfg.Redis != nil && cfg.Redis.Addr != "" {
		redisClient, err := NewRedisClient(cfg.Redis)
		if err != nil {
			log.Printf("⚠ Redis 连接失败: %v（继续运行，不使用缓存）", err)
		} else {
			manager.Redis = redisClient
		}
	}

	return manager, nil
}

/*
  HasRedis 检查 Redis 是否可用
  功能：判断 Redis 客户端是否已连接且可用
*/
func (m *Manager) HasRedis() bool {
	return m.Redis != nil && m.Redis.IsAvailable()
}

/*
  Close 关闭所有连接
  功能：优雅地关闭数据库和 Redis 连接
*/
func (m *Manager) Close() error {
	var errs []error

	/* 关闭数据库连接 */
	if m.DB != nil {
		sqlDB, err := m.DB.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("关闭数据库连接失败: %w", err))
			}
		}
	}

	/* 关闭 Redis 连接 */
	if m.Redis != nil {
		if err := m.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 Redis 连接失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭连接时发生错误: %v", errs)
	}

	log.Println("✓ 所有数据库连接已关闭")
	return nil
}

/*
  GetDBType 获取当前数据库类型
  功能：返回当前使用的数据库引擎类型
*/
func (m *Manager) GetDBType() DBType {
	return m.dbConfig.Type
}

/*
  HealthCheck 健康检查
  功能：检测数据库和 Redis 的连接状态
*/
func (m *Manager) HealthCheck() map[string]interface{} {
	result := map[string]interface{}{
		"database_type": string(m.dbConfig.Type),
	}

	/* 检查数据库连接 */
	if m.DB != nil {
		sqlDB, err := m.DB.DB()
		if err == nil {
			if err := sqlDB.Ping(); err == nil {
				result["database_status"] = "connected"
				stats := sqlDB.Stats()
				result["database_stats"] = map[string]interface{}{
					"open_connections": stats.OpenConnections,
					"in_use":          stats.InUse,
					"idle":            stats.Idle,
				}
			} else {
				result["database_status"] = "error"
				result["database_error"] = err.Error()
			}
		}
	}

	/* 检查 Redis 连接 */
	if m.Redis != nil {
		if m.Redis.IsAvailable() {
			result["redis_status"] = "connected"
		} else {
			result["redis_status"] = "disconnected"
		}
	} else {
		result["redis_status"] = "not_configured"
	}

	return result
}
