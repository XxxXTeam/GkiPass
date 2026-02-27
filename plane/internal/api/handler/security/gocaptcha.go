package security

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"
)

/*
GoCaptchaHandler GoCaptcha 验证码 API 处理器
功能：处理行为验证码的生成和验证请求
*/
type GoCaptchaHandler struct {
	app     *types.App
	service *service.GoCaptchaService
	logger  *zap.Logger
}

/*
NewGoCaptchaHandler 创建 GoCaptcha 处理器
*/
func NewGoCaptchaHandler(app *types.App) *GoCaptchaHandler {
	captchaSvc, err := service.NewGoCaptchaService(&app.Config.Captcha)
	if err != nil {
		zap.L().Error("初始化 GoCaptcha 服务失败", zap.Error(err))
		return &GoCaptchaHandler{
			app:    app,
			logger: zap.L().Named("gocaptcha-handler"),
		}
	}

	return &GoCaptchaHandler{
		app:     app,
		service: captchaSvc,
		logger:  zap.L().Named("gocaptcha-handler"),
	}
}

/*
Generate 生成验证码
功能：根据请求参数生成对应模式的行为验证码，返回图片和验证数据
路由：GET /api/v1/captcha/gocaptcha/generate
*/
func (h *GoCaptchaHandler) Generate(c *gin.Context) {
	if h.service == nil {
		response.GinError(c, http.StatusServiceUnavailable, "验证码服务未初始化", nil)
		return
	}

	mode := c.DefaultQuery("mode", h.app.Config.Captcha.GoCaptchaMode)
	if mode == "" {
		mode = "click"
	}

	result, err := h.service.Generate(mode)
	if err != nil {
		h.logger.Error("生成验证码失败", zap.String("mode", mode), zap.Error(err))
		response.GinInternalError(c, "生成验证码失败", err)
		return
	}

	response.GinSuccess(c, result)
}

/*
Verify 验证验证码
功能：验证用户提交的行为验证码数据是否正确
路由：POST /api/v1/captcha/gocaptcha/verify
*/
func (h *GoCaptchaHandler) Verify(c *gin.Context) {
	if h.service == nil {
		response.GinError(c, http.StatusServiceUnavailable, "验证码服务未初始化", nil)
		return
	}

	var req service.GoCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	success, err := h.service.Verify(&req)
	if err != nil {
		h.logger.Debug("验证码验证失败", zap.Error(err))
		response.GinSuccess(c, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	response.GinSuccess(c, gin.H{
		"success": success,
	})
}
