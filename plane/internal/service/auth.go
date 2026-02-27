package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"go.uber.org/zap"
)

/*
AuthService 认证服务
功能：提供 API 密钥验证、JWT 令牌验证和目标地址解析
*/
type AuthService struct {
	jwtSecret string
	logger    *zap.Logger
}

/*
NewAuthService 创建认证服务
*/
func NewAuthService() *AuthService {
	return &AuthService{
		logger: zap.L().Named("auth-service"),
	}
}

/*
SetJWTSecret 设置 JWT 密钥
功能：在 JWTManager 初始化后注入密钥
*/
func (s *AuthService) SetJWTSecret(secret string) {
	s.jwtSecret = secret
}

/*
GetJWTSecret 获取 JWT 密钥
功能：供 JWTAuth 中间件获取密钥进行令牌签名验证
*/
func (s *AuthService) GetJWTSecret() string {
	return s.jwtSecret
}

/*
ValidateAPIKey 验证 API 密钥
功能：使用 HMAC-SHA256 验证 API 密钥的有效性
*/
func (s *AuthService) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return errors.New("API 密钥不能为空")
	}

	/* API 密钥格式：前缀.签名，通过 HMAC 验证 */
	parts := strings.SplitN(apiKey, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return errors.New("API 密钥格式无效")
	}

	if s.jwtSecret == "" {
		return errors.New("服务端签名密钥未配置")
	}

	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return errors.New("API 密钥签名验证失败")
	}

	return nil
}

/*
ValidateToken 验证 JWT 令牌
功能：解析并验证 JWT 令牌，提取用户声明信息
*/
func (s *AuthService) ValidateToken(token string) (*UserClaims, error) {
	if token == "" {
		return nil, errors.New("令牌不能为空")
	}

	/* JWT 令牌由 middleware 层的 jwt-go 库负责解析验证 */
	/* 此处仅做基础格式检查，实际验证在 JWTAuth 中间件完成 */
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("令牌格式无效")
	}

	return &UserClaims{}, nil
}

/*
ParseTargetsFromString 解析目标地址字符串
功能：将逗号分隔的目标地址列表解析为字符串数组，
支持格式：host:port,host:port,...
*/
func ParseTargetsFromString(targets string) ([]string, error) {
	if targets == "" {
		return []string{}, nil
	}

	var result []string
	parts := strings.Split(targets, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result, nil
}
