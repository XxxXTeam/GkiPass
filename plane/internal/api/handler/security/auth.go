package security

import (
	"time"

	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
AuthHandler 认证处理器
功能：处理用户登录、登出和令牌刷新
*/
type AuthHandler struct {
	app     *types.App
	userSvc *service.GormUserService
	logger  *zap.Logger
}

/*
NewAuthHandler 创建认证处理器
*/
func NewAuthHandler(app *types.App) *AuthHandler {
	return &AuthHandler{
		app:     app,
		userSvc: service.NewGormUserService(app.DB.GormDB),
		logger:  zap.L().Named("auth-handler"),
	}
}

/*
LoginRequest 登录请求
*/
type LoginRequest struct {
	Username    string `json:"username" binding:"required,max=32"`
	Password    string `json:"password" binding:"required,max=128"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

/*
LoginResponse 登录响应
*/
type LoginResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
}

/*
Login 用户登录
功能：验证码校验 → 凭据认证 → 生成JWT → 返回令牌
路由：POST /api/v1/auth/login
*/
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	/* 验证码检查（如果启用） */
	if h.app.Config.Captcha.Enabled && h.app.Config.Captcha.EnableLogin {
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

	/* 使用 GormUserService 认证 */
	user, err := h.userSvc.Authenticate(req.Username, req.Password)
	if err != nil {
		h.logger.Debug("登录认证失败",
			zap.String("username", req.Username),
			zap.String("client_ip", c.ClientIP()),
			zap.Error(err))
		/* 统一返回模糊错误信息，防止用户名枚举攻击 */
		response.GinUnauthorized(c, "用户名或密码错误")
		return
	}

	/* 生成 JWT 令牌 */
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

	response.GinSuccess(c, LoginResponse{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Username,
		Role:      string(user.Role),
		ExpiresAt: expiresAt.Unix(),
	})
}

/*
Logout 用户登出
功能：清除 Redis 中的会话缓存（如果启用）
路由：POST /api/v1/auth/logout
*/
func (h *AuthHandler) Logout(c *gin.Context) {
	if h.app.DB.HasCache() {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			_ = h.app.DB.Cache.Redis.DeleteSession(token)
		}
	}

	response.GinSuccessWithMessage(c, "已成功登出", nil)
}

/*
RefreshToken 刷新JWT令牌
功能：验证当前令牌有效性 → 生成新令牌（延长有效期）
路由：POST /api/v1/auth/refresh
*/
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	usernameStr := middleware.GetUsername(c)
	roleStr := middleware.GetRole(c)

	if userIDStr == "" {
		response.GinUnauthorized(c, "无效的会话")
		return
	}

	/* 确认用户仍然有效 */
	user, err := h.userSvc.GetUser(userIDStr)
	if err != nil {
		response.GinUnauthorized(c, "用户不存在")
		return
	}
	if !user.Enabled {
		response.GinForbidden(c, "账户已被禁用")
		return
	}

	/* 生成新令牌 */
	token, err := middleware.GenerateJWT(
		userIDStr,
		usernameStr,
		roleStr,
		h.app.Config.Auth.JWTSecret,
		h.app.Config.Auth.JWTExpiration,
	)
	if err != nil {
		response.GinInternalError(c, "生成令牌失败", err)
		return
	}

	expiresAt := time.Now().Add(time.Duration(h.app.Config.Auth.JWTExpiration) * time.Hour)

	response.GinSuccess(c, LoginResponse{
		Token:     token,
		UserID:    userIDStr,
		Username:  usernameStr,
		Role:      roleStr,
		ExpiresAt: expiresAt.Unix(),
	})
}
