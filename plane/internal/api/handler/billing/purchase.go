package billing

import (
	"fmt"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PurchasePlanRequest 购买套餐请求
type PurchasePlanRequest struct {
	PlanID        string `json:"plan_id" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}

// PurchasePlan 购买套餐
func (h *PlanHandler) PurchasePlan(c *gin.Context) {
	var req PurchasePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	plan, err := h.app.DAO.GetPlan(req.PlanID)
	if err != nil || plan == nil {
		response.GinNotFound(c, "Plan not found")
		return
	}

	/* 创建购买订单 */
	orderID := uuid.New().String()
	order := &models.Order{
		UserID:      userID,
		Type:        "purchase",
		Status:      "pending",
		Amount:      plan.Price,
		PayMethod:   req.PaymentMethod,
		PlanID:      plan.ID,
		Description: fmt.Sprintf("购买套餐：%s", plan.Name),
	}
	order.ID = orderID

	if err := h.app.DAO.CreateOrder(order); err != nil {
		logger.Error("创建套餐购买订单失败", zap.Error(err))
		response.InternalError(c, "Failed to create purchase order")
		return
	}

	logger.Info("创建套餐购买订单",
		zap.String("orderID", orderID),
		zap.String("userID", userID),
		zap.String("planID", plan.ID),
		zap.Float64("price", plan.Price))

	response.GinSuccess(c, gin.H{
		"order_id": orderID,
		"plan":     plan,
		"amount":   plan.Price,
		"status":   "pending",
	})
}

// ActivateSubscription 激活订阅（支付成功回调）
func (h *PlanHandler) ActivateSubscription(orderID, userID, planID string) error {
	plan, err := h.app.DAO.GetPlan(planID)
	if err != nil || plan == nil {
		return fmt.Errorf("plan not found")
	}

	existingSub, err := h.app.DAO.GetActiveSubscription(userID)
	if err != nil {
		return err
	}

	now := time.Now()

	/* 根据 DurationUnit 计算有效期，与 GormPlanService.Subscribe 逻辑统一 */
	calcExpire := func(base time.Time) time.Time {
		switch plan.DurationUnit {
		case "year":
			return base.AddDate(plan.Duration, 0, 0)
		case "permanent":
			return base.AddDate(100, 0, 0)
		default: /* month */
			return base.AddDate(0, plan.Duration, 0)
		}
	}

	var expireAt time.Time
	if existingSub != nil && existingSub.ExpireAt.After(now) {
		/* 续期：在现有到期时间基础上延长 */
		expireAt = calcExpire(existingSub.ExpireAt)
	} else {
		expireAt = calcExpire(now)
	}

	sub := &models.Subscription{}
	sub.ID = uuid.New().String()
	sub.UserID = userID
	sub.PlanID = planID
	sub.Status = "active"
	sub.StartAt = now
	sub.ExpireAt = expireAt
	sub.AutoRenew = false

	if existingSub != nil {
		sub.ID = existingSub.ID
		if err := h.app.DAO.UpdateSubscription(sub); err != nil {
			return err
		}
	} else {
		if err := h.app.DAO.CreateSubscription(sub); err != nil {
			return err
		}
	}

	logger.Info("激活订阅成功",
		zap.String("userID", userID),
		zap.String("planID", planID),
		zap.Time("expireAt", expireAt))

	return nil
}
