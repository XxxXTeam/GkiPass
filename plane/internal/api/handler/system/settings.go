package system

import (
	"encoding/json"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SettingsHandler 设置处理器
type SettingsHandler struct {
	app *types.App
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler(app *types.App) *SettingsHandler {
	return &SettingsHandler{app: app}
}

// GetCaptchaSettings 获取验证码设置
func (h *SettingsHandler) GetCaptchaSettings(c *gin.Context) {
	setting, err := h.app.DAO.GetSystemSetting("captcha")
	if err != nil {
		// 如果出错（例如表不存在），记录日志并返回默认配置
		logger.Warn("Failed to get captcha settings from database, using config defaults", zap.Error(err))
		response.GinSuccess(c, h.app.Config.Captcha)
		return
	}

	if setting == nil {
		// 返回默认设置
		response.GinSuccess(c, h.app.Config.Captcha)
		return
	}

	var captchaSettings map[string]interface{}
	if err := json.Unmarshal([]byte(setting.Value), &captchaSettings); err != nil {
		logger.Error("Failed to parse captcha settings", zap.Error(err))
		response.GinSuccess(c, h.app.Config.Captcha)
		return
	}

	response.GinSuccess(c, captchaSettings)
}

// UpdateCaptchaSettingsRequest 更新验证码设置请求
type UpdateCaptchaSettingsRequest struct {
	Enabled            bool   `json:"enabled"`
	Type               string `json:"type" binding:"omitempty,oneof=image turnstile gocaptcha"`
	EnableLogin        bool   `json:"enable_login"`
	EnableRegister     bool   `json:"enable_register"`
	ImageWidth         int    `json:"image_width" binding:"omitempty,min=100,max=800"`
	ImageHeight        int    `json:"image_height" binding:"omitempty,min=30,max=200"`
	CodeLength         int    `json:"code_length" binding:"omitempty,min=4,max=8"`
	Expiration         int    `json:"expiration" binding:"omitempty,min=30,max=600"`
	TurnstileSiteKey   string `json:"turnstile_site_key" binding:"omitempty,max=256"`
	TurnstileSecretKey string `json:"turnstile_secret_key" binding:"omitempty,max=256"`
}

// UpdateCaptchaSettings 更新验证码设置
func (h *SettingsHandler) UpdateCaptchaSettings(c *gin.Context) {
	var req UpdateCaptchaSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	valueJSON, err := json.Marshal(req)
	if err != nil {
		response.InternalError(c, "Failed to marshal settings")
		return
	}

	setting := &models.SystemSetting{
		Key:      "captcha",
		Value:    string(valueJSON),
		Category: "captcha",
		Type:     "json",
	}

	if err := h.app.DAO.UpsertSystemSetting(setting); err != nil {
		response.InternalError(c, "Failed to update settings")
		return
	}

	// 同步更新配置
	h.app.Config.Captcha.Enabled = req.Enabled
	h.app.Config.Captcha.Type = req.Type
	h.app.Config.Captcha.EnableLogin = req.EnableLogin
	h.app.Config.Captcha.EnableRegister = req.EnableRegister

	response.SuccessWithMessage(c, "Settings updated successfully", req)
}

// GeneralSettings 通用设置结构
type GeneralSettings struct {
	SiteName                 string `json:"site_name"`
	SiteDescription          string `json:"site_description"`
	SiteLogo                 string `json:"site_logo"`
	AllowRegister            bool   `json:"allow_register"`
	RequireEmailVerification bool   `json:"require_email_verification"`
}

// GetGeneralSettings 获取通用设置
func (h *SettingsHandler) GetGeneralSettings(c *gin.Context) {
	defaultSettings := GeneralSettings{
		SiteName:                 "GKI Pass",
		SiteDescription:          "企业级双向隧道控制平台",
		SiteLogo:                 "",
		AllowRegister:            true,
		RequireEmailVerification: false,
	}

	setting, err := h.app.DAO.GetSystemSetting("general")
	if err != nil {
		logger.Warn("Failed to get general settings from database, using defaults", zap.Error(err))
		response.GinSuccess(c, defaultSettings)
		return
	}

	if setting == nil {
		response.GinSuccess(c, defaultSettings)
		return
	}

	var generalSettings GeneralSettings
	if err := json.Unmarshal([]byte(setting.Value), &generalSettings); err != nil {
		logger.Error("Failed to parse general settings", zap.Error(err))
		response.GinSuccess(c, defaultSettings)
		return
	}

	response.GinSuccess(c, generalSettings)
}

// UpdateGeneralSettingsRequest 更新通用设置请求
type UpdateGeneralSettingsRequest struct {
	SiteName                 string `json:"site_name" binding:"omitempty,max=128"`
	SiteDescription          string `json:"site_description" binding:"omitempty,max=512"`
	SiteLogo                 string `json:"site_logo" binding:"omitempty,max=1024"`
	AllowRegister            bool   `json:"allow_register"`
	RequireEmailVerification bool   `json:"require_email_verification"`
}

// UpdateGeneralSettings 更新通用设置
func (h *SettingsHandler) UpdateGeneralSettings(c *gin.Context) {
	var req UpdateGeneralSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	valueJSON, err := json.Marshal(req)
	if err != nil {
		response.InternalError(c, "Failed to marshal settings")
		return
	}

	setting := &models.SystemSetting{
		Key:      "general",
		Value:    string(valueJSON),
		Category: "general",
		Type:     "json",
	}

	if err := h.app.DAO.UpsertSystemSetting(setting); err != nil {
		response.InternalError(c, "Failed to update settings")
		return
	}

	response.SuccessWithMessage(c, "General settings updated successfully", req)
}

// SecuritySettings 安全设置结构
type SecuritySettings struct {
	PasswordMinLength        int  `json:"password_min_length"`
	PasswordRequireUppercase bool `json:"password_require_uppercase"`
	PasswordRequireLowercase bool `json:"password_require_lowercase"`
	PasswordRequireNumber    bool `json:"password_require_number"`
	PasswordRequireSpecial   bool `json:"password_require_special"`
	LoginMaxAttempts         int  `json:"login_max_attempts"`
	LoginLockoutDuration     int  `json:"login_lockout_duration"` // 分钟
	Enable2FA                bool `json:"enable_2fa"`
	SessionTimeout           int  `json:"session_timeout"` // 小时
}

// GetSecuritySettings 获取安全设置
func (h *SettingsHandler) GetSecuritySettings(c *gin.Context) {
	defaultSettings := SecuritySettings{
		PasswordMinLength:        6,
		PasswordRequireUppercase: false,
		PasswordRequireLowercase: false,
		PasswordRequireNumber:    false,
		PasswordRequireSpecial:   false,
		LoginMaxAttempts:         5,
		LoginLockoutDuration:     30,
		Enable2FA:                false,
		SessionTimeout:           24,
	}

	setting, err := h.app.DAO.GetSystemSetting("security")
	if err != nil {
		logger.Warn("Failed to get security settings from database, using defaults", zap.Error(err))
		response.GinSuccess(c, defaultSettings)
		return
	}

	if setting == nil {
		response.GinSuccess(c, defaultSettings)
		return
	}

	var securitySettings SecuritySettings
	if err := json.Unmarshal([]byte(setting.Value), &securitySettings); err != nil {
		logger.Error("Failed to parse security settings", zap.Error(err))
		response.GinSuccess(c, defaultSettings)
		return
	}

	response.GinSuccess(c, securitySettings)
}

// UpdateSecuritySettings 更新安全设置
func (h *SettingsHandler) UpdateSecuritySettings(c *gin.Context) {
	var req SecuritySettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	valueJSON, err := json.Marshal(req)
	if err != nil {
		response.InternalError(c, "Failed to marshal settings")
		return
	}

	setting := &models.SystemSetting{
		Key:      "security",
		Value:    string(valueJSON),
		Category: "security",
		Type:     "json",
	}

	if err := h.app.DAO.UpsertSystemSetting(setting); err != nil {
		response.InternalError(c, "Failed to update settings")
		return
	}

	response.SuccessWithMessage(c, "Security settings updated successfully", req)
}

// NotificationSettings 通知设置结构
type NotificationSettings struct {
	EmailEnabled         bool   `json:"email_enabled"`
	EmailHost            string `json:"email_host"`
	EmailPort            int    `json:"email_port"`
	EmailUsername        string `json:"email_username"`
	EmailPassword        string `json:"email_password"`
	EmailFrom            string `json:"email_from"`
	SystemNotifyEnabled  bool   `json:"system_notify_enabled"`
	WebhookEnabled       bool   `json:"webhook_enabled"`
	WebhookURL           string `json:"webhook_url"`
	NotifyOnUserRegister bool   `json:"notify_on_user_register"`
	NotifyOnPayment      bool   `json:"notify_on_payment"`
	NotifyOnNodeOffline  bool   `json:"notify_on_node_offline"`
}

// GetNotificationSettings 获取通知设置
func (h *SettingsHandler) GetNotificationSettings(c *gin.Context) {
	defaultSettings := NotificationSettings{
		EmailEnabled:         false,
		EmailHost:            "smtp.example.com",
		EmailPort:            587,
		EmailUsername:        "",
		EmailPassword:        "",
		EmailFrom:            "noreply@example.com",
		SystemNotifyEnabled:  true,
		WebhookEnabled:       false,
		WebhookURL:           "",
		NotifyOnUserRegister: true,
		NotifyOnPayment:      true,
		NotifyOnNodeOffline:  true,
	}

	setting, err := h.app.DAO.GetSystemSetting("notification")
	if err != nil {
		logger.Warn("Failed to get notification settings from database, using defaults", zap.Error(err))
		response.GinSuccess(c, defaultSettings)
		return
	}

	if setting == nil {
		response.GinSuccess(c, defaultSettings)
		return
	}

	var notificationSettings NotificationSettings
	if err := json.Unmarshal([]byte(setting.Value), &notificationSettings); err != nil {
		logger.Error("Failed to parse notification settings", zap.Error(err))
		response.GinSuccess(c, defaultSettings)
		return
	}

	/* 脱敏：邮箱密码不通过 API 返回明文 */
	if notificationSettings.EmailPassword != "" {
		notificationSettings.EmailPassword = "********"
	}

	response.GinSuccess(c, notificationSettings)
}

// UpdateNotificationSettings 更新通知设置
func (h *SettingsHandler) UpdateNotificationSettings(c *gin.Context) {
	var req NotificationSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证邮件设置
	if req.EmailEnabled {
		if req.EmailHost == "" || req.EmailFrom == "" {
			response.GinBadRequest(c, "Email host and from address are required when email is enabled")
			return
		}
		if req.EmailPort < 1 || req.EmailPort > 65535 {
			response.GinBadRequest(c, "Invalid email port")
			return
		}
	}

	valueJSON, err := json.Marshal(req)
	if err != nil {
		response.InternalError(c, "Failed to marshal settings")
		return
	}

	setting := &models.SystemSetting{
		Key:      "notification",
		Value:    string(valueJSON),
		Category: "notification",
		Type:     "json",
	}

	if err := h.app.DAO.UpsertSystemSetting(setting); err != nil {
		response.InternalError(c, "Failed to update settings")
		return
	}

	logger.Info("更新通知设置",
		zap.Bool("email_enabled", req.EmailEnabled),
		zap.Bool("system_notify_enabled", req.SystemNotifyEnabled))

	response.SuccessWithMessage(c, "Notification settings updated successfully", req)
}
