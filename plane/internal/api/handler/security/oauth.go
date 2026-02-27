package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// OAuthHandler OAuth处理器
type OAuthHandler struct {
	app          *types.App
	githubConfig *oauth2.Config
}

// NewOAuthHandler 创建OAuth处理器
func NewOAuthHandler(app *types.App) *OAuthHandler {
	handler := &OAuthHandler{
		app: app,
	}

	// 初始化GitHub OAuth配置
	if app.Config.Auth.GitHub.Enabled {
		handler.githubConfig = &oauth2.Config{
			ClientID:     app.Config.Auth.GitHub.ClientID,
			ClientSecret: app.Config.Auth.GitHub.ClientSecret,
			RedirectURL:  app.Config.Auth.GitHub.RedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}
	}

	return handler
}

// GitHubUser GitHub用户信息
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubLoginRequest GitHub登录请求
type GitHubLoginRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// GitHubLoginURL 获取GitHub登录URL
func (h *OAuthHandler) GitHubLoginURL(c *gin.Context) {
	if h.githubConfig == nil {
		response.GinBadRequest(c, "GitHub OAuth is not enabled")
		return
	}

	// 生成随机state
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	// 如果有Redis，缓存state（5分钟）
	if h.app.DB.HasCache() {
		_ = h.app.DB.Cache.Redis.SetWithExpire("oauth:state:"+state, "pending", 5*time.Minute)
	}

	url := h.githubConfig.AuthCodeURL(state)
	response.GinSuccess(c, gin.H{
		"url":   url,
		"state": state,
	})
}

// GitHubCallback GitHub回调处理
func (h *OAuthHandler) GitHubCallback(c *gin.Context) {
	if h.githubConfig == nil {
		response.GinBadRequest(c, "GitHub OAuth is not enabled")
		return
	}

	var req GitHubLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 验证state（如果有Redis）
	if h.app.DB.HasCache() {
		stateKey := "oauth:state:" + req.State
		val, _ := h.app.DB.Cache.Redis.GetString(stateKey)
		if val != "pending" {
			response.GinBadRequest(c, "Invalid state parameter")
			return
		}
		// 删除已使用的state
		_ = h.app.DB.Cache.Redis.Delete(stateKey)
	}

	// 交换code获取token
	ctx := context.Background()
	token, err := h.githubConfig.Exchange(ctx, req.Code)
	if err != nil {
		logger.Error("GitHub OAuth token 交换失败", zap.Error(err))
		response.GinBadRequest(c, "GitHub 认证失败，请重试")
		return
	}

	// 获取GitHub用户信息
	client := h.githubConfig.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		response.InternalError(c, "Failed to get user info")
		return
	}
	defer resp.Body.Close()

	/* 限制外部 API 响应体最大 1MB */
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var githubUser GitHubUser
	if err := json.Unmarshal(data, &githubUser); err != nil {
		response.InternalError(c, "Failed to parse user info")
		return
	}

	// 如果邮箱为空，尝试获取主邮箱
	if githubUser.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			emailData, _ := io.ReadAll(io.LimitReader(emailResp.Body, 1<<20))
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if json.Unmarshal(emailData, &emails) == nil {
				for _, e := range emails {
					if e.Primary {
						githubUser.Email = e.Email
						break
					}
				}
			}
		}
	}

	providerID := fmt.Sprintf("%d", githubUser.ID)
	user, err := h.app.DAO.GetUserByProvider("github", providerID)
	if err != nil {
		response.InternalError(c, "Database error")
		return
	}

	if user == nil {
		username := githubUser.Login
		existingUser, _ := h.app.DAO.GetUserByUsername(username)
		if existingUser != nil {
			username = fmt.Sprintf("%s_%d", githubUser.Login, githubUser.ID)
		}

		user = &models.User{
			Username:   username,
			Email:      githubUser.Email,
			Avatar:     githubUser.AvatarURL,
			Provider:   "github",
			ProviderID: providerID,
			Role:       models.RoleUser,
			Enabled:    true,
		}
		user.ID = uuid.New().String()

		if err := h.app.DAO.CreateUser(user); err != nil {
			response.InternalError(c, "Failed to create user")
			return
		}
	}

	_ = h.app.DAO.UpdateUserLastLogin(user.ID)

	// 生成JWT token
	jwtToken, err := middleware.GenerateJWT(
		user.ID,
		user.Username,
		string(user.Role),
		h.app.Config.Auth.JWTSecret,
		h.app.Config.Auth.JWTExpiration,
	)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	// 计算过期时间
	expiresAt := time.Now().Add(time.Duration(h.app.Config.Auth.JWTExpiration) * time.Hour)

	response.GinSuccess(c, gin.H{
		"token":      jwtToken,
		"user_id":    user.ID,
		"username":   user.Username,
		"avatar":     user.Avatar,
		"role":       user.Role,
		"expires_at": expiresAt.Unix(),
	})
}
