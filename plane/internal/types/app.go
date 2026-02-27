package types

import (
	"gkipass/plane/internal/db"
	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/config"
)

/*
App 应用实例
功能：全局应用上下文，包含配置、数据库管理器和 GORM 数据访问层
*/
type App struct {
	Config *config.Config
	DB     *db.Manager
	DAO    *dao.DAO /* GORM 统一数据访问层 */
}

/*
NewApp 创建新的应用实例
*/
func NewApp(cfg *config.Config, dbManager *db.Manager) *App {
	return &App{
		Config: cfg,
		DB:     dbManager,
		DAO:    dao.New(dbManager.GormDB),
	}
}
