package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
)

/*
GenerateJWT 生成 JWT 令牌
功能：使用 HMAC-SHA256 签名算法生成包含用户信息的 JWT 令牌
参数：userID 用户ID, username 用户名, role 角色, jwtSecret 签名密钥, expiresInHours 有效期(小时)
*/
func GenerateJWT(userID, username, role, jwtSecret string, expiresInHours int) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"iat":      now.Unix(),
		"exp":      now.Add(time.Duration(expiresInHours) * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("签名 JWT 令牌失败: %w", err)
	}
	return signed, nil
}

/*
JWTAuth 返回 Gin JWT 认证中间件
功能：从 Authorization 头提取 Bearer 令牌，使用 HMAC-SHA256 验证签名，
解析 claims 并注入 Gin 上下文供后续 handler 使用
*/
func JWTAuth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.GinUnauthorized(c, "缺少认证令牌")
			c.Abort()
			return
		}

		/* 提取 Bearer 令牌 */
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			response.GinUnauthorized(c, "认证头格式无效，需 Bearer <token>")
			c.Abort()
			return
		}

		/* 解析并验证 JWT */
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("不支持的签名方法: %v", t.Header["alg"])
			}
			return []byte(authService.GetJWTSecret()), nil
		})
		if err != nil || !token.Valid {
			response.GinUnauthorized(c, "无效或已过期的令牌")
			c.Abort()
			return
		}

		/* 提取 claims 并注入上下文 */
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.GinUnauthorized(c, "令牌 claims 解析失败")
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])
		c.Set("user_claims", claims)
		c.Next()
	}
}
