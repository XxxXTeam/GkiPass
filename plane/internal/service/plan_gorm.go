package service

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

/*
GormPlanService 基于 GORM 的套餐服务
功能：管理套餐的完整生命周期（创建/查询/更新/删除），
以及用户订阅管理（订阅/续费/配额检查/流量统计）
*/
type GormPlanService struct {
	db     *gorm.DB
	logger *zap.Logger
}

/*
NewGormPlanService 创建基于 GORM 的套餐服务
*/
func NewGormPlanService(db *gorm.DB) *GormPlanService {
	return &GormPlanService{
		db:     db,
		logger: zap.L().Named("gorm-plan-service"),
	}
}

/* ==================== 套餐 CRUD ==================== */

/*
CreatePlan 创建套餐
*/
func (s *GormPlanService) CreatePlan(plan *models.Plan) error {
	if plan.Name == "" {
		return fmt.Errorf("套餐名称不能为空")
	}
	if plan.Price < 0 {
		return fmt.Errorf("套餐价格不能为负数")
	}
	if plan.Duration <= 0 {
		return fmt.Errorf("套餐有效期必须大于0")
	}

	if err := s.db.Create(plan).Error; err != nil {
		s.logger.Error("创建套餐失败", zap.String("name", plan.Name), zap.Error(err))
		return fmt.Errorf("创建套餐失败: %w", err)
	}

	s.logger.Info("套餐已创建", zap.String("id", plan.ID), zap.String("name", plan.Name))
	return nil
}

/*
GetPlan 获取套餐详情
*/
func (s *GormPlanService) GetPlan(id string) (*models.Plan, error) {
	var plan models.Plan
	if err := s.db.First(&plan, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("套餐不存在")
		}
		return nil, err
	}
	return &plan, nil
}

/*
ListPlans 列出套餐列表
参数 enabledOnly：true 仅列出启用的套餐（用户可见），false 列出全部（管理员可见）
*/
func (s *GormPlanService) ListPlans(enabledOnly bool) ([]models.Plan, error) {
	var plans []models.Plan
	q := s.db.Order("sort_order ASC, created_at ASC")
	if enabledOnly {
		q = q.Where("enabled = ?", true)
	}
	if err := q.Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

/*
UpdatePlan 更新套餐
*/
func (s *GormPlanService) UpdatePlan(id string, updates map[string]interface{}) (*models.Plan, error) {
	plan, err := s.GetPlan(id)
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(plan).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新套餐失败: %w", err)
	}

	s.logger.Info("套餐已更新", zap.String("id", id))
	return plan, nil
}

/*
DeletePlan 删除套餐（软删除）
功能：检查是否有活跃订阅使用此套餐，有则拒绝删除
*/
func (s *GormPlanService) DeletePlan(id string) error {
	/* 检查是否有活跃订阅 */
	var activeCount int64
	s.db.Model(&models.Subscription{}).
		Where("plan_id = ? AND status = 'active'", id).
		Count(&activeCount)

	if activeCount > 0 {
		return fmt.Errorf("该套餐有 %d 个活跃订阅，无法删除", activeCount)
	}

	if err := s.db.Delete(&models.Plan{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除套餐失败: %w", err)
	}

	s.logger.Info("套餐已删除", zap.String("id", id))
	return nil
}

/* ==================== 订阅管理 ==================== */

/*
SubscribeRequest 订阅请求
*/
type SubscribeRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
	Months int    `json:"months"`
}

/*
Subscribe 用户订阅套餐
功能：验证套餐有效性 → 检查重复订阅 → 计算到期时间 → 创建订阅记录
*/
func (s *GormPlanService) Subscribe(userID string, req *SubscribeRequest) (*models.Subscription, error) {
	/* 获取套餐 */
	plan, err := s.GetPlan(req.PlanID)
	if err != nil {
		return nil, err
	}
	if !plan.Enabled {
		return nil, fmt.Errorf("该套餐未启用")
	}

	/* 检查用户是否已有活跃订阅 */
	var activeCount int64
	s.db.Model(&models.Subscription{}).
		Where("user_id = ? AND status = 'active' AND expire_at > ?", userID, time.Now()).
		Count(&activeCount)

	if activeCount > 0 {
		return nil, fmt.Errorf("您已有活跃的订阅，请等待到期后再订阅")
	}

	/* 计算有效期 */
	months := req.Months
	if months <= 0 {
		months = plan.Duration
	}

	startAt := time.Now()
	var expireAt time.Time
	switch plan.DurationUnit {
	case "month":
		expireAt = startAt.AddDate(0, months, 0)
	case "year":
		expireAt = startAt.AddDate(months, 0, 0)
	case "permanent":
		expireAt = startAt.AddDate(100, 0, 0)
	default:
		expireAt = startAt.AddDate(0, months, 0)
	}

	sub := &models.Subscription{
		UserID:   userID,
		PlanID:   plan.ID,
		Status:   "active",
		StartAt:  startAt,
		ExpireAt: expireAt,
	}

	if err := s.db.Create(sub).Error; err != nil {
		return nil, fmt.Errorf("创建订阅失败: %w", err)
	}

	s.logger.Info("用户已订阅套餐",
		zap.String("userID", userID),
		zap.String("planID", plan.ID),
		zap.String("planName", plan.Name),
		zap.Time("expireAt", expireAt))

	return sub, nil
}

/*
GetActiveSubscription 获取用户当前活跃订阅（含套餐信息）
*/
func (s *GormPlanService) GetActiveSubscription(userID string) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.
		Preload("Plan").
		Where("user_id = ? AND status = 'active' AND expire_at > ?", userID, time.Now()).
		Order("created_at DESC").
		First(&sub).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &sub, nil
}

/*
CheckAndExpireSubscriptions 检查并标记过期订阅
功能：批量将过期的活跃订阅标记为 expired（可由定时任务调用）
*/
func (s *GormPlanService) CheckAndExpireSubscriptions() (int64, error) {
	result := s.db.Model(&models.Subscription{}).
		Where("status = 'active' AND expire_at <= ?", time.Now()).
		Update("status", "expired")

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		s.logger.Info("已标记过期订阅", zap.Int64("count", result.RowsAffected))
	}

	return result.RowsAffected, nil
}

/* ==================== 配额检查 ==================== */

/*
QuotaInfo 配额信息
功能：包含用户当前套餐的各项限额和使用情况
*/
type QuotaInfo struct {
	HasSubscription bool   `json:"has_subscription"`
	PlanName        string `json:"plan_name"`
	TrafficLimit    int64  `json:"traffic_limit"`
	SpeedLimit      int64  `json:"speed_limit"`
	ConnectionLimit int    `json:"connection_limit"`
	RuleLimit       int    `json:"rule_limit"`
	ExpireAt        string `json:"expire_at"`
}

/*
GetQuotaInfo 获取用户配额信息
*/
func (s *GormPlanService) GetQuotaInfo(userID string) (*QuotaInfo, error) {
	sub, err := s.GetActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	if sub == nil {
		return &QuotaInfo{HasSubscription: false}, nil
	}

	return &QuotaInfo{
		HasSubscription: true,
		PlanName:        sub.Plan.Name,
		TrafficLimit:    sub.Plan.TrafficLimit,
		SpeedLimit:      sub.Plan.SpeedLimit,
		ConnectionLimit: sub.Plan.ConnectionLimit,
		RuleLimit:       sub.Plan.RuleLimit,
		ExpireAt:        sub.ExpireAt.Format(time.RFC3339),
	}, nil
}

/*
CheckTunnelQuota 检查用户是否还能创建隧道
功能：根据套餐的 RuleLimit 校验当前隧道数是否达到上限
*/
func (s *GormPlanService) CheckTunnelQuota(userID string) error {
	sub, err := s.GetActiveSubscription(userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return fmt.Errorf("未订阅套餐，无法创建隧道")
	}

	if sub.Plan.RuleLimit <= 0 {
		return nil /* 无限制 */
	}

	/* 统计当前用户的隧道数 */
	var tunnelCount int64
	s.db.Model(&models.Tunnel{}).Where("created_by = ?", userID).Count(&tunnelCount)

	if int(tunnelCount) >= sub.Plan.RuleLimit {
		return fmt.Errorf("隧道数已达上限 (%d/%d)", tunnelCount, sub.Plan.RuleLimit)
	}

	return nil
}
