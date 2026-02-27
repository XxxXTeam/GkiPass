package user

import (
	"strconv"
	"time"

	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
UserHandler 用户处理器
功能：处理用户注册、信息查询、密码修改、管理员操作等
*/
type UserHandler struct {
	app     *types.App
	userSvc *service.GormUserService
	planSvc *service.GormPlanService
	logger  *zap.Logger
}

/*
NewUserHandler 创建用户处理器
*/
func NewUserHandler(app *types.App) *UserHandler {
	return &UserHandler{
		app:     app,
		userSvc: service.NewGormUserService(app.DB.GormDB),
		planSvc: service.NewGormPlanService(app.DB.GormDB),
		logger:  zap.L().Named("user-handler"),
	}
}

/*
RegisterRequest 注册请求
*/
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=32"`
	Password    string `json:"password" binding:"required,min=8,max=128"`
	Email       string `json:"email" binding:"required,email,max=128"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

/*
Register 用户注册
功能：验证码校验 → 字段校验 + 密码强度 → 创建用户+钱包（事务） → 自动登录
路由：POST /api/v1/auth/register
*/
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	/* 验证码检查（如果启用） */
	if h.app.Config.Captcha.Enabled && h.app.Config.Captcha.EnableRegister {
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			response.GinBadRequest(c, "请输入验证码")
			return
		}

		if h.app.DB.HasCache() {
			valid, err := h.app.DB.Cache.Redis.VerifyAndDeleteCaptcha(req.CaptchaID, req.CaptchaCode)
			if err != nil || !valid {
				response.GinBadRequest(c, "验证码无效或已过期")
				return
			}
		}
	}

	/* 使用 GormUserService 注册（内部已包含字段校验 + 密码强度 + 重复检查 + 事务创建） */
	result, err := h.userSvc.Register(&service.GormRegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	})
	if err != nil {
		h.logger.Warn("注册失败",
			zap.String("username", req.Username),
			zap.Error(err))
		response.GinBadRequest(c, err.Error())
		return
	}

	user := result.User

	/* 生成 JWT 令牌（注册后自动登录） */
	token, err := middleware.GenerateJWT(
		user.ID,
		user.Username,
		string(user.Role),
		h.app.Config.Auth.JWTSecret,
		h.app.Config.Auth.JWTExpiration,
	)
	if err != nil {
		h.logger.Error("生成令牌失败", zap.Error(err))
		response.GinInternalError(c, "生成令牌失败", err)
		return
	}

	expiresAt := time.Now().Add(time.Duration(h.app.Config.Auth.JWTExpiration) * time.Hour)

	response.GinSuccessWithMessage(c, "注册成功", gin.H{
		"token":         token,
		"user_id":       user.ID,
		"username":      user.Username,
		"role":          string(user.Role),
		"expires_at":    expiresAt.Unix(),
		"is_first_user": result.IsFirstUser,
	})
}

/*
GetProfile 获取用户基本信息
路由：GET /api/v1/users/profile
*/
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.userSvc.GetUser(userID)
	if err != nil {
		response.GinNotFound(c, "用户不存在")
		return
	}

	response.GinSuccess(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"enabled":    user.Enabled,
		"avatar":     user.Avatar,
		"created_at": user.CreatedAt,
		"last_login": user.LastLogin,
	})
}

/*
UpdatePasswordRequest 修改密码请求
*/
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,max=128"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

/*
UpdatePassword 修改密码
功能：验证旧密码 → 校验新密码强度 → 更新
路由：POST /api/v1/users/password/update
*/
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.userSvc.UpdatePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "密码已更新", nil)
}

/*
ListUsers 列出所有用户（管理员）
功能：支持分页，返回用户列表（密码字段已通过 json:"-" 自动隐藏）
路由：GET /api/v1/users
*/
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.userSvc.ListUsers(page, pageSize)
	if err != nil {
		response.GinInternalError(c, "获取用户列表失败", err)
		return
	}

	response.GinSuccess(c, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

/*
GetCurrentUser 获取当前用户完整信息（包括订阅、钱包、权限等）
路由：GET /api/v1/users/me
*/
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	/* 使用 GORM Preload 一次性获取用户+订阅+钱包 */
	user, err := h.userSvc.GetUserWithRelations(userID)
	if err != nil {
		response.GinNotFound(c, "用户不存在")
		return
	}

	/* 构建用户数据 */
	userData := gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"is_admin":   user.Role == "admin",
		"enabled":    user.Enabled,
		"avatar":     user.Avatar,
		"created_at": user.CreatedAt,
		"last_login": user.LastLogin,
	}

	/* 订阅信息 */
	if len(user.Subscriptions) > 0 {
		sub := user.Subscriptions[0]
		userData["subscription"] = gin.H{
			"id":        sub.ID,
			"plan_id":   sub.PlanID,
			"status":    sub.Status,
			"start_at":  sub.StartAt,
			"expire_at": sub.ExpireAt,
			"plan":      sub.Plan,
		}
	} else {
		userData["subscription"] = nil
	}

	/* 钱包信息 */
	if user.Wallet != nil {
		userData["wallet"] = gin.H{
			"id":            user.Wallet.ID,
			"balance":       user.Wallet.Balance,
			"frozen_amount": user.Wallet.FrozenAmount,
		}
	} else {
		userData["wallet"] = gin.H{"id": "", "balance": 0, "frozen_amount": 0}
	}

	/* 权限信息 */
	var permissions []string
	if user.Role == "admin" {
		permissions = []string{
			"user:read", "user:write", "user:delete",
			"node:read", "node:write", "node:delete",
			"tunnel:read", "tunnel:write", "tunnel:delete",
			"plan:read", "plan:write", "plan:delete",
			"announcement:read", "announcement:write", "announcement:delete",
			"settings:read", "settings:write",
			"statistics:read",
			"admin:access",
		}
	} else {
		permissions = []string{
			"tunnel:read", "tunnel:write", "tunnel:delete:own",
			"node:read:own",
			"profile:read", "profile:write",
			"wallet:read", "subscription:read",
		}
	}
	userData["permissions"] = permissions

	response.GinSuccess(c, userData)
}

/*
ToggleUserStatus 启用/禁用用户（管理员）
路由：POST /api/v1/users/:id/status/update
*/
func (h *UserHandler) ToggleUserStatus(c *gin.Context) {
	targetUserID := c.Param("id")
	currentUserID := middleware.GetUserID(c)

	newStatus, err := h.userSvc.ToggleUserStatus(targetUserID, currentUserID)
	if err != nil {
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "用户状态已更新", gin.H{
		"user_id": targetUserID,
		"enabled": newStatus,
	})
}

/*
UpdateUserRoleRequest 更新角色请求
*/
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

/*
UpdateUserRole 更新用户角色（管理员）
路由：POST /api/v1/users/:id/role/update
*/
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	targetUserID := c.Param("id")
	currentUserID := middleware.GetUserID(c)

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	if err := h.userSvc.UpdateUserRole(targetUserID, currentUserID, req.Role); err != nil {
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "用户角色已更新", gin.H{
		"user_id": targetUserID,
		"role":    req.Role,
	})
}

/*
DeleteUser 删除用户（管理员、软删除）
路由：POST /api/v1/users/:id/delete
*/
func (h *UserHandler) DeleteUser(c *gin.Context) {
	targetUserID := c.Param("id")
	currentUserID := middleware.GetUserID(c)

	if err := h.userSvc.DeleteUser(targetUserID, currentUserID); err != nil {
		response.GinBadRequest(c, err.Error())
		return
	}

	response.GinSuccessWithMessage(c, "用户已删除", nil)
}

/*
GetUserPermissions 获取用户权限详情
路由：GET /api/v1/users/permissions
*/
func (h *UserHandler) GetUserPermissions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	roleStr := middleware.GetRole(c)

	permissionDetails := gin.H{
		"user_id":     userID,
		"role":        roleStr,
		"permissions": map[string]interface{}{},
	}

	if roleStr == "admin" {
		permissionDetails["permissions"] = map[string]interface{}{
			"user":         map[string]bool{"read": true, "create": true, "update": true, "delete": true},
			"node":         map[string]bool{"read": true, "create": true, "update": true, "delete": true},
			"tunnel":       map[string]bool{"read": true, "create": true, "update": true, "delete": true},
			"plan":         map[string]bool{"read": true, "create": true, "update": true, "delete": true},
			"announcement": map[string]bool{"read": true, "create": true, "update": true, "delete": true},
			"settings":     map[string]bool{"read": true, "update": true},
			"statistics":   map[string]bool{"read": true},
		}
		permissionDetails["admin"] = true
		permissionDetails["can_access_admin_panel"] = true
	} else {
		permissionDetails["permissions"] = map[string]interface{}{
			"tunnel":       map[string]bool{"read": true, "create": true, "update_own": true, "delete_own": true},
			"node":         map[string]bool{"read_own": true},
			"profile":      map[string]bool{"read": true, "update": true},
			"wallet":       map[string]bool{"read": true},
			"subscription": map[string]bool{"read": true},
		}
		permissionDetails["admin"] = false
		permissionDetails["can_access_admin_panel"] = false
	}

	response.GinSuccess(c, permissionDetails)
}
