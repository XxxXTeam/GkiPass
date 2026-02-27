package billing

import (
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
PlanHandler 套餐处理器
功能：处理套餐的 CRUD、用户订阅和订阅查询
*/
type PlanHandler struct {
	app     *types.App
	planSvc *service.GormPlanService
	logger  *zap.Logger
}

/*
NewPlanHandler 创建套餐处理器
*/
func NewPlanHandler(app *types.App) *PlanHandler {
	return &PlanHandler{
		app:     app,
		planSvc: service.NewGormPlanService(app.DB.GormDB),
		logger:  zap.L().Named("plan-handler"),
	}
}

/*
CreatePlanRequest 创建/更新套餐请求
*/
type CreatePlanRequest struct {
	Name            string  `json:"name" binding:"required,min=1,max=64"`
	Description     string  `json:"description" binding:"omitempty,max=512"`
	Price           float64 `json:"price" binding:"gte=0"`
	Duration        int     `json:"duration" binding:"required,min=1,max=3650"`
	DurationUnit    string  `json:"duration_unit" binding:"omitempty,oneof=day month year"`
	TrafficLimit    int64   `json:"traffic_limit" binding:"gte=0"`
	SpeedLimit      int64   `json:"speed_limit" binding:"gte=0"`
	ConnectionLimit int     `json:"connection_limit" binding:"gte=0"`
	RuleLimit       int     `json:"rule_limit" binding:"gte=0"`
	NodeGroupIDs    string  `json:"node_group_ids" binding:"omitempty,max=1024"`
	Enabled         bool    `json:"enabled"`
	SortOrder       int     `json:"sort_order" binding:"gte=0,lte=9999"`
}

/*
Create 创建套餐（仅管理员）
路由：POST /api/v1/plans/create
*/
func (h *PlanHandler) Create(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	unit := req.DurationUnit
	if unit == "" {
		unit = "month"
	}

	plan := &models.Plan{
		Name:            req.Name,
		Description:     req.Description,
		Price:           req.Price,
		Duration:        req.Duration,
		DurationUnit:    unit,
		TrafficLimit:    req.TrafficLimit,
		SpeedLimit:      req.SpeedLimit,
		ConnectionLimit: req.ConnectionLimit,
		RuleLimit:       req.RuleLimit,
		NodeGroupIDs:    req.NodeGroupIDs,
		Enabled:         req.Enabled,
		SortOrder:       req.SortOrder,
	}

	if err := h.planSvc.CreatePlan(plan); err != nil {
		h.logger.Error("创建套餐失败", zap.Error(err))
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "套餐创建成功", plan)
}

/*
List 列出套餐
路由：GET /api/v1/plans
*/
func (h *PlanHandler) List(c *gin.Context) {
	/* 管理员查看全部，普通用户只看启用的 */
	enabledOnly := !middleware.IsAdmin(c)

	plans, err := h.planSvc.ListPlans(enabledOnly)
	if err != nil {
		h.logger.Error("列出套餐失败", zap.Error(err))
		response.GinInternalError(c, "获取套餐列表失败", err)
		return
	}

	response.GinSuccess(c, plans)
}

/*
Get 获取套餐详情
路由：GET /api/v1/plans/:id
*/
func (h *PlanHandler) Get(c *gin.Context) {
	id := c.Param("id")

	plan, err := h.planSvc.GetPlan(id)
	if err != nil {
		response.GinNotFound(c, err.Error())
		return
	}

	response.GinSuccess(c, plan)
}

/*
Update 更新套餐（仅管理员）
路由：POST /api/v1/plans/:id/update
*/
func (h *PlanHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	updates := map[string]interface{}{
		"name":             req.Name,
		"description":      req.Description,
		"price":            req.Price,
		"duration":         req.Duration,
		"traffic_limit":    req.TrafficLimit,
		"speed_limit":      req.SpeedLimit,
		"connection_limit": req.ConnectionLimit,
		"rule_limit":       req.RuleLimit,
		"node_group_ids":   req.NodeGroupIDs,
		"enabled":          req.Enabled,
		"sort_order":       req.SortOrder,
	}
	if req.DurationUnit != "" {
		updates["duration_unit"] = req.DurationUnit
	}

	plan, err := h.planSvc.UpdatePlan(id, updates)
	if err != nil {
		h.logger.Error("更新套餐失败", zap.String("id", id), zap.Error(err))
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "套餐更新成功", plan)
}

/*
Delete 删除套餐（仅管理员，软删除）
路由：POST /api/v1/plans/:id/delete
*/
func (h *PlanHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.planSvc.DeletePlan(id); err != nil {
		h.logger.Error("删除套餐失败", zap.String("id", id), zap.Error(err))
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "套餐已删除", nil)
}

/*
Subscribe 用户订阅套餐
路由：POST /api/v1/plans/:id/subscribe
*/
func (h *PlanHandler) Subscribe(c *gin.Context) {
	planID := c.Param("id")
	userID := middleware.GetUserID(c)

	var req service.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.PlanID = planID
	if req.Months <= 0 {
		req.Months = 1
	}

	sub, err := h.planSvc.Subscribe(userID, &req)
	if err != nil {
		h.logger.Error("订阅套餐失败",
			zap.String("userID", userID),
			zap.String("planID", planID),
			zap.Error(err))
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "订阅成功", sub)
}

/*
MySubscription 获取当前用户的订阅信息
路由：GET /api/v1/plans/my/subscription
*/
func (h *PlanHandler) MySubscription(c *gin.Context) {
	userID := middleware.GetUserID(c)

	sub, err := h.planSvc.GetActiveSubscription(userID)
	if err != nil {
		response.GinInternalError(c, "获取订阅信息失败", err)
		return
	}

	/* 用户无订阅，返回 null 而不是错误 */
	if sub == nil {
		response.GinSuccess(c, nil)
		return
	}

	response.GinSuccess(c, sub)
}
