package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

/*
DBType 数据库类型枚举
功能：定义系统支持的数据库引擎类型
*/
type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "postgres"
)

/*
Config 数据库连接配置
功能：统一管理不同数据库引擎的连接参数
*/
type Config struct {
	Type     DBType `yaml:"type" json:"type"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	DBName   string `yaml:"db_name" json:"db_name"`
	SSLMode  string `yaml:"ssl_mode" json:"ssl_mode"`
	Charset  string `yaml:"charset" json:"charset"`

	/* SQLite 专用 */
	SQLitePath string `yaml:"sqlite_path" json:"sqlite_path"`

	/* 连接池配置 */
	MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`

	/* 日志配置 */
	LogLevel string `yaml:"log_level" json:"log_level"`
}

/*
DefaultConfig 返回默认的 SQLite 配置
功能：提供开箱即用的数据库配置
*/
func DefaultConfig() *Config {
	return &Config{
		Type:            DBTypeSQLite,
		SQLitePath:      "./data/gkipass.db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		LogLevel:        "warn",
		Charset:         "utf8mb4",
		SSLMode:         "disable",
	}
}

/*
NewDatabase 创建数据库连接
功能：根据配置类型初始化对应的数据库引擎连接
支持 SQLite、MySQL、PostgreSQL 三种数据库
*/
func NewDatabase(cfg *Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Type {
	case DBTypeSQLite:
		dialector = buildSQLiteDialector(cfg)
	case DBTypeMySQL:
		dialector = buildMySQLDialector(cfg)
	case DBTypePostgres:
		dialector = buildPostgresDialector(cfg)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s, 支持: sqlite/mysql/postgres", cfg.Type)
	}

	/* 配置 GORM 日志级别 */
	gormLogger := buildGormLogger(cfg.LogLevel)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败 [%s]: %w", cfg.Type, err)
	}

	/* 配置连接池 */
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	log.Printf("✓ 数据库连接成功 [%s]", cfg.Type)
	return db, nil
}

/*
AutoMigrate 自动迁移数据库表结构
功能：根据 GORM 模型定义自动创建或更新数据库表
*/
func AutoMigrate(db *gorm.DB) error {
	log.Println("开始自动迁移数据库表结构...")

	err := db.AutoMigrate(
		/* 用户相关 */
		&models.User{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Wallet{},
		&models.Transaction{},
		&models.Subscription{},

		/* 节点相关 */
		&models.Node{},
		&models.NodeGroup{},
		&models.NodeMetrics{},
		&models.NodeCertificate{},
		&models.ConnectionKey{},

		/* 隧道和规则 */
		&models.Tunnel{},
		&models.TunnelTarget{},
		&models.Rule{},
		&models.ACLRule{},
		&models.TrafficStats{},

		/* 策略和节点组配置 */
		&models.Policy{},
		&models.NodeGroupConfig{},

		/* 系统相关 */
		&models.Plan{},
		&models.Order{},
		&models.Announcement{},
		&models.Notification{},
		&models.SystemSetting{},
		&models.PaymentConfig{},
		&models.PaymentMonitor{},
		&models.AuditLog{},

		/* 监控相关 */
		&models.NodeMonitoringConfig{},
		&models.NodeMonitoringData{},
		&models.NodePerformanceHistory{},
		&models.NodeAlertRule{},
		&models.NodeAlertHistory{},
		&models.MonitoringPermission{},
	)

	if err != nil {
		return fmt.Errorf("数据库自动迁移失败: %w", err)
	}

	/* 容灾事件表 - 手动建表（避免循环引用 service 包） */
	if !db.Migrator().HasTable("failover_events") {
		type FailoverEvent struct {
			models.BaseModel
			NodeID          string    `gorm:"type:varchar(36);index;not null"`
			TunnelID        string    `gorm:"type:varchar(36);index;not null"`
			EventType       string    `gorm:"type:varchar(16);not null"`
			FromGroupID     string    `gorm:"type:varchar(36);not null"`
			ToGroupID       string    `gorm:"type:varchar(36);not null"`
			Reason          string    `gorm:"type:varchar(256)"`
			FailureDuration int       `gorm:"default:0"`
			Timestamp       time.Time `gorm:"index;not null"`
		}
		if err := db.Table("failover_events").AutoMigrate(&FailoverEvent{}); err != nil {
			log.Printf("⚠ 创建 failover_events 表失败: %v", err)
		}
	}

	/* 隧道加密密钥表 - 手动建表（避免循环引用 service 包） */
	if !db.Migrator().HasTable("tunnel_encryption_keys") {
		type TunnelEncryptionKey struct {
			models.BaseModel
			TunnelID    string    `gorm:"type:varchar(36);index;not null"`
			Algorithm   string    `gorm:"type:varchar(32);not null"`
			KeyHex      string    `gorm:"type:varchar(128);not null"`
			KeySize     int       `gorm:"not null"`
			Version     int       `gorm:"default:1;not null"`
			Active      bool      `gorm:"default:true;not null"`
			ExpiresAt   time.Time `gorm:"index"`
			RotatedFrom string    `gorm:"type:varchar(36)"`
		}
		if err := db.Table("tunnel_encryption_keys").AutoMigrate(&TunnelEncryptionKey{}); err != nil {
			log.Printf("⚠ 创建 tunnel_encryption_keys 表失败: %v", err)
		}
	}

	log.Println("✓ 数据库表结构迁移完成")
	return nil
}

/*
buildSQLiteDialector 构建 SQLite 连接器
功能：初始化 SQLite 数据库文件和连接参数
*/
func buildSQLiteDialector(cfg *Config) gorm.Dialector {
	dbPath := cfg.SQLitePath
	if dbPath == "" {
		dbPath = "./data/gkipass.db"
	}

	/* 确保目录存在 */
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("创建数据库目录失败: %v", err)
	}

	return sqlite.Open(dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
}

/*
buildMySQLDialector 构建 MySQL 连接器
功能：生成 MySQL DSN 并初始化连接
*/
func buildMySQLDialector(cfg *Config) gorm.Dialector {
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, port, cfg.DBName, charset)

	return mysql.Open(dsn)
}

/*
buildPostgresDialector 构建 PostgreSQL 连接器
功能：生成 PostgreSQL DSN 并初始化连接
*/
func buildPostgresDialector(cfg *Config) gorm.Dialector {
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Host, port, cfg.User, cfg.Password, cfg.DBName, sslMode)

	return postgres.Open(dsn)
}

/*
buildGormLogger 根据配置构建 GORM 日志记录器
功能：控制 ORM 层的 SQL 日志输出级别
*/
func buildGormLogger(level string) logger.Interface {
	var logLevel logger.LogLevel
	switch level {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Warn
	}

	return logger.Default.LogMode(logLevel)
}
