package tunnel

import (
	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/service"

	"gorm.io/gorm"
)

// Manager 隧道管理器（模块封装）
type Manager struct {
	gormTunnelSvc *service.GormTunnelService
	planService   *service.PlanService
}

// NewManager 创建隧道管理器
func NewManager(gormDB *gorm.DB, d *dao.DAO) *Manager {
	return &Manager{
		gormTunnelSvc: service.NewGormTunnelService(gormDB),
		planService:   service.NewPlanService(d),
	}
}

// GetGormTunnelService 获取 GORM 隧道服务
func (m *Manager) GetGormTunnelService() *service.GormTunnelService {
	return m.gormTunnelSvc
}

// GetPlanService 获取套餐服务
func (m *Manager) GetPlanService() *service.PlanService {
	return m.planService
}
