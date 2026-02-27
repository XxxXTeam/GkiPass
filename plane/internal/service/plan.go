package service

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PlanService 套餐服务（兼容旧接口，内部使用 DAO）
type PlanService struct {
	dao *dao.DAO
}

// NewPlanService 创建套餐服务
func NewPlanService(d *dao.DAO) *PlanService {
	return &PlanService{dao: d}
}

// CreatePlan 创建套餐
func (s *PlanService) CreatePlan(plan *models.Plan) error {
	plan.ID = uuid.New().String()
	if err := s.dao.CreatePlan(plan); err != nil {
		logger.Error("创建套餐失败", zap.String("name", plan.Name), zap.Error(err))
		return err
	}
	logger.Info("套餐已创建", zap.String("id", plan.ID), zap.String("name", plan.Name))
	return nil
}

// GetPlan 获取套餐
func (s *PlanService) GetPlan(id string) (*models.Plan, error) {
	return s.dao.GetPlan(id)
}

// ListPlans 列出套餐
func (s *PlanService) ListPlans(enabledOnly bool) ([]models.Plan, error) {
	return s.dao.ListPlans(enabledOnly)
}

// UpdatePlan 更新套餐
func (s *PlanService) UpdatePlan(plan *models.Plan) error {
	if err := s.dao.UpdatePlan(plan); err != nil {
		logger.Error("更新套餐失败", zap.String("id", plan.ID), zap.Error(err))
		return err
	}
	logger.Info("套餐已更新", zap.String("id", plan.ID), zap.String("name", plan.Name))
	return nil
}

// DeletePlan 删除套餐
func (s *PlanService) DeletePlan(id string) error {
	subs, err := s.dao.GetSubscriptionsByPlanID(id)
	if err != nil {
		return err
	}
	if len(subs) > 0 {
		return fmt.Errorf("套餐正在被 %d 个用户使用，无法删除", len(subs))
	}
	if err := s.dao.DeletePlan(id); err != nil {
		logger.Error("删除套餐失败", zap.String("id", id), zap.Error(err))
		return err
	}
	logger.Info("套餐已删除", zap.String("id", id))
	return nil
}

// SubscribeUserToPlan 用户订阅套餐
func (s *PlanService) SubscribeUserToPlan(userID, planID string, months int) (*models.Subscription, error) {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return nil, err
	}
	if !plan.Enabled {
		return nil, fmt.Errorf("套餐未启用")
	}

	existingSub, _ := s.dao.GetActiveSubscription(userID)
	if existingSub != nil {
		return nil, fmt.Errorf("用户已有有效订阅")
	}

	startAt := time.Now()
	var expireAt time.Time
	if months <= 0 {
		months = plan.Duration
	}
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
		PlanID:   planID,
		Status:   "active",
		StartAt:  startAt,
		ExpireAt: expireAt,
	}
	sub.ID = uuid.New().String()

	if err := s.dao.CreateSubscription(sub); err != nil {
		logger.Error("创建订阅失败",
			zap.String("userID", userID),
			zap.String("planID", planID),
			zap.Error(err))
		return nil, err
	}

	logger.Info("用户已订阅套餐",
		zap.String("userID", userID),
		zap.String("planID", planID),
		zap.String("planName", plan.Name))

	return sub, nil
}

// GetUserSubscription 获取用户活跃订阅及套餐
func (s *PlanService) GetUserSubscription(userID string) (*models.Subscription, *models.Plan, error) {
	sub, err := s.dao.GetActiveSubscription(userID)
	if err != nil {
		return nil, nil, err
	}
	if sub == nil {
		return nil, nil, nil
	}

	if time.Now().After(sub.ExpireAt) {
		sub.Status = "expired"
		_ = s.dao.UpdateSubscription(sub)
		return nil, nil, nil
	}

	plan, err := s.GetPlan(sub.PlanID)
	if err != nil {
		return sub, nil, err
	}

	return sub, plan, nil
}

// CheckQuota 检查配额
func (s *PlanService) CheckQuota(userID string, checkType string) error {
	_, plan, err := s.GetUserSubscription(userID)
	if err != nil {
		return err
	}
	if plan == nil {
		return fmt.Errorf("未订阅套餐")
	}

	switch checkType {
	case "rules":
		if plan.RuleLimit > 0 {
			/* 统计当前隧道数 */
			var count int64
			s.dao.DB.Model(&models.Tunnel{}).Where("created_by = ?", userID).Count(&count)
			if int(count) >= plan.RuleLimit {
				return fmt.Errorf("规则数已达上限 (%d/%d)", count, plan.RuleLimit)
			}
		}
	case "traffic":
		/* 流量限制由 TrafficLimiter 实时检查，此处仅校验订阅有效性 */
	}

	return nil
}

// IncrementRuleCount 增加规则数（新模型由 CheckQuota 动态统计，此方法为兼容保留）
func (s *PlanService) IncrementRuleCount(userID string) error {
	return nil
}

// DecrementRuleCount 减少规则数（新模型由 CheckQuota 动态统计，此方法为兼容保留）
func (s *PlanService) DecrementRuleCount(userID string) error {
	return nil
}

// AddTrafficUsage 添加流量使用（新模型中流量统计移至独立表，此处为兼容保留）
func (s *PlanService) AddTrafficUsage(userID string, bytes int64) error {
	return nil
}
