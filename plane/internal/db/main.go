package db

import (
	"fmt"
	"log"

	"gkipass/plane/internal/db/cache"
	"gkipass/plane/internal/db/database"

	"gorm.io/gorm"
)

/*
Manager 数据库管理器
功能：统一管理 GORM 数据库连接和 Redis 缓存
*/
type Manager struct {
	GormDB *gorm.DB /* GORM 统一数据库 */
	Cache  *Cache   /* Redis 缓存（可选） */

	redisPool *cache.Pool
}

/*
Config 数据库配置
功能：支持多数据库类型（SQLite/MySQL/PostgreSQL）+ Redis 缓存
*/
type Config struct {
	/* 数据库类型：sqlite, mysql, postgres */
	DBType string

	/* SQLite 配置 */
	SQLitePath string

	/* MySQL/PostgreSQL 配置 */
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBCharset  string

	/* 连接池 */
	MaxOpenConns int
	MaxIdleConns int

	/* 日志级别 */
	DBLogLevel string

	/* Redis 配置 */
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

/*
NewManager 创建数据库管理器
功能：初始化 GORM 数据库 + 旧 SQLite 兼容层 + Redis 缓存
自动执行 AutoMigrate 创建/更新表结构
*/
func NewManager(cfg *Config) (*Manager, error) {
	manager := &Manager{}

	/* 1. 初始化 GORM 数据库（主要） */
	dbType := cfg.DBType
	if dbType == "" {
		dbType = "sqlite"
	}

	gormCfg := &database.Config{
		Type:         database.DBType(dbType),
		Host:         cfg.DBHost,
		Port:         cfg.DBPort,
		User:         cfg.DBUser,
		Password:     cfg.DBPassword,
		DBName:       cfg.DBName,
		SSLMode:      cfg.DBSSLMode,
		Charset:      cfg.DBCharset,
		SQLitePath:   cfg.SQLitePath,
		MaxOpenConns: cfg.MaxOpenConns,
		MaxIdleConns: cfg.MaxIdleConns,
		LogLevel:     cfg.DBLogLevel,
	}

	/* 设置连接池默认值 */
	if gormCfg.MaxOpenConns == 0 {
		gormCfg.MaxOpenConns = 25
	}
	if gormCfg.MaxIdleConns == 0 {
		gormCfg.MaxIdleConns = 5
	}
	if gormCfg.LogLevel == "" {
		gormCfg.LogLevel = "warn"
	}

	gormDB, err := database.NewDatabase(gormCfg)
	if err != nil {
		return nil, fmt.Errorf("初始化 GORM 数据库失败: %w", err)
	}
	manager.GormDB = gormDB

	/* 自动迁移表结构 */
	if err := database.AutoMigrate(gormDB); err != nil {
		return nil, fmt.Errorf("数据库自动迁移失败: %w", err)
	}

	/* 2. 初始化 Redis 缓存（可选） */
	if cfg.RedisAddr != "" {
		redisPool, err := cache.NewPool(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		if err != nil {
			log.Printf("⚠ Redis 连接失败: %v（继续运行，无缓存）", err)
		} else {
			manager.Cache = NewCache(redisPool.Get())
			manager.redisPool = redisPool
			log.Printf("✓ Redis 已连接: %s", cfg.RedisAddr)
		}
	}

	return manager, nil
}

/*
Close 关闭所有数据库连接
*/
func (m *Manager) Close() error {
	var errs []error

	/* 关闭 GORM 数据库 */
	if m.GormDB != nil {
		if sqlDB, err := m.GormDB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("GORM 关闭失败: %w", err))
			}
		}
	}

	/* 关闭 Redis */
	if m.redisPool != nil {
		if err := m.redisPool.Close(); err != nil {
			errs = append(errs, fmt.Errorf("Redis 关闭失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭数据库错误: %v", errs)
	}

	return nil
}

/*
HasCache 检查是否有 Redis 缓存可用
*/
func (m *Manager) HasCache() bool {
	return m.Cache != nil && m.Cache.Redis != nil
}

/*
GetGormDB 获取 GORM 数据库实例
功能：供新服务层使用的便捷方法
*/
func (m *Manager) GetGormDB() *gorm.DB {
	return m.GormDB
}
