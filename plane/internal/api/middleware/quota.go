package middleware

import (
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

/*
QuotaCheck 配额检查中间件
功能：检查用户是否有活跃订阅，将订阅信息注入上下文
管理员自动跳过检查
*/
func QuotaCheck(gormDB *gorm.DB) gin.HandlerFunc {
	planSvc := service.NewGormPlanService(gormDB)

	return func(c *gin.Context) {
		/* 管理员跳过检查 */
		role, exists := c.Get("role")
		if exists && role == "admin" {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			response.GinUnauthorized(c, "未认证")
			c.Abort()
			return
		}

		/* 获取用户配额信息并注入上下文 */
		quota, err := planSvc.GetQuotaInfo(userID.(string))
		if err != nil {
			response.GinInternalError(c, "检查订阅失败", err)
			c.Abort()
			return
		}
		c.Set("quota", quota)

		c.Next()
	}
}

/*
RuleQuotaCheck 规则配额检查（用于创建隧道）
功能：检查用户是否还能创建隧道（未达套餐上限）
*/
func RuleQuotaCheck(gormDB *gorm.DB) gin.HandlerFunc {
	planSvc := service.NewGormPlanService(gormDB)

	return func(c *gin.Context) {
		/* 管理员跳过检查 */
		role, exists := c.Get("role")
		if exists && role == "admin" {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			response.GinUnauthorized(c, "未认证")
			c.Abort()
			return
		}

		if err := planSvc.CheckTunnelQuota(userID.(string)); err != nil {
			logger.Warn("规则配额不足",
				zap.String("userID", userID.(string)),
				zap.Error(err))
			response.GinForbidden(c, err.Error())
			c.Abort()
			return
		}

		c.Next()
	}
}

/*
TrafficQuotaCheck 流量配额检查
功能：检查用户流量是否超出套餐限额（预留接口，待流量统计完善后启用）
*/
func TrafficQuotaCheck(gormDB *gorm.DB) gin.HandlerFunc {
	/* 预留：待流量统计完善后使用 planSvc := service.NewGormPlanService(gormDB) */

	return func(c *gin.Context) {
		/* 管理员跳过检查 */
		role, exists := c.Get("role")
		if exists && role == "admin" {
			c.Next()
			return
		}

		/* TODO: 待流量统计完善后启用流量配额检查 */
		c.Next()
	}
}
