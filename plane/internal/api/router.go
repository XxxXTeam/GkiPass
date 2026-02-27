package api

import (
	"time"

	"gkipass/plane/internal/api/handler/billing"
	"gkipass/plane/internal/api/handler/node"
	"gkipass/plane/internal/api/handler/security"
	"gkipass/plane/internal/api/handler/system"
	"gkipass/plane/internal/api/handler/tunnel"
	"gkipass/plane/internal/api/handler/user"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter 设置路由
func SetupRouter(app *App, wsServer *ws.Server) *gin.Engine {
	// 设置Gin模式
	if app.Config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 全局中间件
	router.Use(middleware.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.BodyLimit(2 << 20)) /* 2MB 请求体上限，防止 OOM */
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(app.Config.Server.CORSAllowedOrigins))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"cache":  app.DB.HasCache(),
		})
	})

	/*
		Prometheus /metrics 和 /ws/stats 包含敏感运行指标，
		仅允许本地/内网访问，生产环境应通过反向代理进一步限制。
		此处使用 localOnly 中间件限制为 127.0.0.1/::1 访问。
	*/
	router.GET("/metrics", localOnlyGuard(), gin.WrapH(promhttp.Handler()))

	// WebSocket 端点（节点连接）
	router.GET("/ws/node", wsServer.HandleWebSocket)

	// WebSocket 状态（仅本地访问）
	router.GET("/ws/stats", localOnlyGuard(), func(c *gin.Context) {
		c.JSON(200, wsServer.GetStats())
	})

	// API v1
	v1 := router.Group("/api/v1")
	{
		captchaHandler := security.NewCaptchaHandler(app)
		v1.GET("/captcha/config", captchaHandler.GetCaptchaConfig)
		v1.GET("/captcha/image", captchaHandler.GenerateImageCaptcha)

		/* GoCaptcha 行为验证码路由 */
		goCaptchaHandler := security.NewGoCaptchaHandler(app)
		v1.GET("/captcha/gocaptcha/generate", goCaptchaHandler.Generate)
		v1.POST("/captcha/gocaptcha/verify", goCaptchaHandler.Verify)

		// 公开公告
		announcementHandler := system.NewAnnouncementHandler(app)
		v1.GET("/announcements", announcementHandler.ListActiveAnnouncements)
		v1.GET("/announcements/:id", announcementHandler.GetAnnouncement)

		/* 登录限流器：每个 IP 每 15 分钟最多 10 次登录尝试 */
		loginLimiter := middleware.NewLoginRateLimiter(10, 15*time.Minute)

		// 认证路由（无需JWT）
		auth := v1.Group("/auth")
		{
			authHandler := security.NewAuthHandler(app)
			userHandler := user.NewUserHandler(app)

			auth.POST("/register", userHandler.Register)
			auth.POST("/login", loginLimiter.Middleware(), authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)

			// GitHub OAuth
			if app.Config.Auth.GitHub.Enabled {
				oauthHandler := security.NewOAuthHandler(app)
				auth.GET("/github", oauthHandler.GitHubLoginURL)
				auth.POST("/github/callback", oauthHandler.GitHubCallback)
			}
		}

		// 需要JWT认证的路由
		authorized := v1.Group("")
		authService := service.NewAuthService()
		authService.SetJWTSecret(app.Config.Auth.JWTSecret)
		authorized.Use(middleware.JWTAuth(authService))
		{
			// 用户管理
			users := authorized.Group("/users")
			{
				userHandler := user.NewUserHandler(app)
				users.GET("/me", userHandler.GetCurrentUser)              // 获取当前用户完整信息
				users.GET("/permissions", userHandler.GetUserPermissions) // 获取用户权限详情
				users.GET("/profile", userHandler.GetProfile)             // 获取基本信息（保留兼容）
				users.POST("/password/update", userHandler.UpdatePassword)

				// 管理员功能
				users.GET("", middleware.AdminAuth(), userHandler.ListUsers)
				users.POST("/:id/status/update", middleware.AdminAuth(), userHandler.ToggleUserStatus)
				users.POST("/:id/role/update", middleware.AdminAuth(), userHandler.UpdateUserRole)
				users.POST("/:id/delete", middleware.AdminAuth(), userHandler.DeleteUser)
			}

			// 节点组管理
			groups := authorized.Group("/node-groups")
			{
				groupHandler := node.NewNodeGroupHandler(app)
				configHandler := node.NewNodeGroupConfigHandler(app)

				/* 所有用户可查看节点组列表和详情 */
				groups.GET("/list", groupHandler.List)
				groups.GET("/:id", groupHandler.Get)
				groups.GET("/:id/config", configHandler.GetNodeGroupConfig)

				/* 管理员专用：节点组增删改和配置修改 */
				groups.POST("/create", middleware.AdminAuth(), groupHandler.Create)
				groups.POST("/:id/update", middleware.AdminAuth(), groupHandler.Update)
				groups.POST("/:id/delete", middleware.AdminAuth(), groupHandler.Delete)
				groups.POST("/:id/config/update", middleware.AdminAuth(), configHandler.UpdateNodeGroupConfig)
				groups.POST("/:id/config/reset", middleware.AdminAuth(), configHandler.ResetNodeGroupConfig)
			}

			// 节点管理
			nodes := authorized.Group("/nodes")
			{
				nodeHandler := node.NewNodeHandler(app)
				ckHandler := security.NewCKHandler(app)
				statusHandler := node.NewNodeStatusHandler(app)
				certHandler := node.NewNodeCertHandler(app)

				/* 所有用户可查看节点（可用节点根据套餐过滤） */
				nodes.GET("/available", nodeHandler.GetAvailableNodes)
				nodes.GET("/list", nodeHandler.List)
				nodes.GET("/:id", nodeHandler.Get)
				nodes.GET("/:id/status", statusHandler.GetNodeStatus)
				nodes.GET("/status/list", statusHandler.ListNodesStatus)
				nodes.GET("/group/:group_id/status", statusHandler.GetNodesByGroup)
				nodes.GET("/:id/cert/info", certHandler.GetCertInfo)

				/* 管理员专用：节点增删改、CK 管理、证书操作 */
				nodes.POST("/create", middleware.AdminAuth(), nodeHandler.Create)
				nodes.POST("/:id/update", middleware.AdminAuth(), nodeHandler.Update)
				nodes.POST("/:id/delete", middleware.AdminAuth(), nodeHandler.Delete)
				nodes.POST("/:id/heartbeat", nodeHandler.Heartbeat)
				nodes.POST("/:id/generate-ck", middleware.AdminAuth(), ckHandler.GenerateNodeCK)
				nodes.GET("/:id/connection-keys", middleware.AdminAuth(), ckHandler.ListNodeCKs)
				nodes.POST("/connection-keys/:ck_id/revoke", middleware.AdminAuth(), ckHandler.RevokeCK)
				nodes.POST("/:id/cert/generate", middleware.AdminAuth(), certHandler.GenerateCert)
				nodes.GET("/:id/cert/download", middleware.AdminAuth(), certHandler.DownloadCert)
				nodes.POST("/:id/cert/renew", middleware.AdminAuth(), certHandler.RenewCert)
			}

			// 节点部署 API
			deployHandler := node.NewNodeDeployHandler(app)

			// 节点组内创建节点
			groups.POST("/:id/nodes", deployHandler.CreateNode)
			groups.GET("/:id/nodes", deployHandler.ListNodesInGroup)

			// 节点注册（公开API，供服务器调用）
			v1.POST("/nodes/register", deployHandler.RegisterNode)
			// 注意：心跳API已在上面的nodes路由组中注册（nodes.POST("/:id/heartbeat", nodeHandler.Heartbeat)）

			// 策略管理
			policies := authorized.Group("/policies")
			{
				policyHandler := tunnel.NewPolicyHandler(app)
				/* 所有用户可查看策略 */
				policies.GET("/list", policyHandler.List)
				policies.GET("/:id", policyHandler.Get)
				/* 管理员专用：策略增删改和部署 */
				policies.POST("/create", middleware.AdminAuth(), policyHandler.Create)
				policies.POST("/:id/update", middleware.AdminAuth(), policyHandler.Update)
				policies.POST("/:id/delete", middleware.AdminAuth(), policyHandler.Delete)
				policies.POST("/:id/deploy", middleware.AdminAuth(), policyHandler.Deploy)
			}

			// 证书管理
			certs := authorized.Group("/certificates")
			{
				certHandler := security.NewCertificateHandler(app)
				/* 管理员专用：证书全部操作 */
				certs.Use(middleware.AdminAuth())
				certs.POST("/ca", certHandler.GenerateCA)
				certs.POST("/leaf", certHandler.GenerateLeaf)
				certs.GET("", certHandler.List)
				certs.GET("/:id", certHandler.Get)
				certs.POST("/:id/revoke", certHandler.Revoke)
				certs.GET("/:id/download", certHandler.Download)
			}

			// 套餐管理
			plans := authorized.Group("/plans")
			{
				planHandler := billing.NewPlanHandler(app)
				plans.GET("", planHandler.List)
				plans.GET("/:id", planHandler.Get)
				plans.POST("/:id/subscribe", middleware.QuotaCheck(app.DB.GormDB), planHandler.Subscribe)
				plans.GET("/my/subscription", planHandler.MySubscription)

				// 仅管理员
				adminPlans := plans.Group("")
				adminPlans.Use(middleware.AdminAuth())
				{
					adminPlans.POST("/create", planHandler.Create)
					adminPlans.POST("/:id/update", planHandler.Update)
					adminPlans.POST("/:id/delete", planHandler.Delete)
				}
			}

			// 隧道管理
			tunnels := authorized.Group("/tunnels")
			tunnels.Use(middleware.QuotaCheck(app.DB.GormDB))
			{
				tunnelHandler := tunnel.NewGinTunnelHandler(app)
				tunnels.GET("/list", tunnelHandler.List)
				tunnels.GET("/:id", tunnelHandler.Get)
				tunnels.POST("/create", tunnelHandler.Create)
				tunnels.POST("/:id/update", tunnelHandler.Update)
				tunnels.POST("/:id/delete", tunnelHandler.Delete)
				tunnels.POST("/:id/toggle", tunnelHandler.Toggle)
			}

			// 统计和监控
			stats := authorized.Group("/statistics")
			{
				statsHandler := user.NewStatisticsHandler(app)
				stats.GET("/nodes/:id", statsHandler.GetNodeStats)
				stats.GET("/overview", statsHandler.GetOverview)
				stats.POST("/report", statsHandler.ReportStats)
			}

			// 流量统计
			traffic := authorized.Group("/traffic")
			{
				trafficHandler := tunnel.NewTrafficStatsHandler(app)
				traffic.GET("/stats", trafficHandler.ListTrafficStats)
				traffic.GET("/summary", trafficHandler.GetTrafficSummary)
				traffic.POST("/report", trafficHandler.ReportTraffic) // 节点上报流量
			}

			// 节点监控
			monitoring := authorized.Group("/monitoring")
			{
				monitoringHandler := system.NewMonitoringHandler(app)
				monitoring.GET("/overview", monitoringHandler.ListNodeMonitoringOverview)
				monitoring.GET("/summary", monitoringHandler.NodeMonitoringSummary)
				monitoring.GET("/nodes/:id/status", monitoringHandler.GetNodeMonitoringStatus)
				monitoring.GET("/nodes/:id/data", monitoringHandler.GetNodeMonitoringData)
				monitoring.GET("/nodes/:id/history", monitoringHandler.GetNodePerformanceHistory)
				monitoring.GET("/nodes/:id/config", monitoringHandler.GetNodeMonitoringConfig)
				monitoring.POST("/nodes/:id/config/update", middleware.AdminAuth(), monitoringHandler.UpdateNodeMonitoringConfig)
				monitoring.GET("/nodes/:id/alerts", monitoringHandler.GetNodeAlerts)
				monitoring.GET("/nodes/:id/alert-rules", monitoringHandler.ListAlertRules)
				monitoring.POST("/nodes/:id/alert-rules", middleware.AdminAuth(), monitoringHandler.CreateAlertRule)
				monitoring.PUT("/alert-rules/:rule_id", middleware.AdminAuth(), monitoringHandler.UpdateAlertRule)
				monitoring.DELETE("/alert-rules/:rule_id", middleware.AdminAuth(), monitoringHandler.DeleteAlertRule)
				monitoring.POST("/alerts/:alert_id/acknowledge", middleware.AdminAuth(), monitoringHandler.AcknowledgeAlert)
				monitoring.POST("/alerts/:alert_id/resolve", middleware.AdminAuth(), monitoringHandler.ResolveAlert)
				monitoring.GET("/permissions", middleware.AdminAuth(), monitoringHandler.ListMonitoringPermissions)
				monitoring.POST("/permissions", middleware.AdminAuth(), monitoringHandler.CreateMonitoringPermission)
				monitoring.GET("/my-permissions", monitoringHandler.GetMyMonitoringPermissions)
			}

			// 容灾事件
			failover := authorized.Group("/failover")
			{
				failoverHandler := system.NewFailoverHandler(app)
				failover.GET("/active", failoverHandler.GetActiveFailovers)
				failover.GET("/tunnels/:tunnel_id/history", failoverHandler.GetTunnelFailoverHistory)
				failover.GET("/groups/:group_id/summary", failoverHandler.GetGroupFailoverSummary)
			}

			// 节点数据上报API（公开API，供节点调用）
			v1.POST("/monitoring/report/:node_id", system.NewMonitoringHandler(app).ReportNodeMonitoringData)

			// 管理员专用统计
			adminStats := authorized.Group("/admin/statistics")
			adminStats.Use(middleware.AdminAuth())
			{
				statsHandler := user.NewStatisticsHandler(app)
				adminStats.GET("/overview", statsHandler.GetAdminOverview)
			}

			// 钱包管理
			wallet := authorized.Group("/wallet")
			{
				walletHandler := user.NewWalletHandler(app)
				wallet.GET("/balance", walletHandler.GetBalance)
				wallet.GET("/transactions", walletHandler.ListTransactions)
				wallet.POST("/recharge", walletHandler.Recharge) // 旧版保留兼容
			}

			// 支付管理
			payment := authorized.Group("/payment")
			{
				paymentHandler := user.NewPaymentHandler(app)
				payment.POST("/recharge", paymentHandler.CreateRechargeOrder)
				payment.GET("/orders/:id", paymentHandler.QueryOrderStatus)
			}

			// 订阅管理
			subscriptions := authorized.Group("/subscriptions")
			{
				subscriptionHandler := user.NewSubscriptionHandler(app)
				subscriptions.GET("/current", subscriptionHandler.GetCurrentSubscription)
				subscriptions.GET("", middleware.AdminAuth(), subscriptionHandler.ListSubscriptions)
			}

			// 通知管理
			notificationHandler := system.NewNotificationHandler(app)
			notifications := authorized.Group("/notifications")
			{
				notifications.GET("", notificationHandler.List)
				notifications.POST("/:id/read", notificationHandler.MarkAsRead)
				notifications.POST("/read-all", notificationHandler.MarkAllAsRead)
				notifications.POST("/:id/delete", notificationHandler.Delete)
			}

			// 管理员专用路由
			admin := authorized.Group("/admin")
			admin.Use(middleware.AdminAuth())
			{
				// 支付配置管理
				paymentConfigHandler := billing.NewPaymentConfigHandler(app)
				paymentHandler := user.NewPaymentHandler(app)
				admin.GET("/payment/configs", paymentConfigHandler.ListConfigs)
				admin.GET("/payment/config/:id", paymentConfigHandler.GetConfig)
				admin.POST("/payment/config/:id/update", paymentConfigHandler.UpdateConfig)
				admin.POST("/payment/config/:id/toggle", paymentConfigHandler.ToggleConfig)
				admin.POST("/payment/manual-recharge", paymentHandler.ManualRecharge)

				// 系统设置
				settingsHandler := system.NewSettingsHandler(app)
				admin.GET("/settings/captcha", settingsHandler.GetCaptchaSettings)
				admin.POST("/settings/captcha/update", settingsHandler.UpdateCaptchaSettings)
				admin.GET("/settings/general", settingsHandler.GetGeneralSettings)
				admin.POST("/settings/general/update", settingsHandler.UpdateGeneralSettings)
				admin.GET("/settings/security", settingsHandler.GetSecuritySettings)
				admin.POST("/settings/security/update", settingsHandler.UpdateSecuritySettings)
				admin.GET("/settings/notification", settingsHandler.GetNotificationSettings)
				admin.POST("/settings/notification/update", settingsHandler.UpdateNotificationSettings)

				// 公告管理
				admin.GET("/announcements", announcementHandler.ListAll)
				admin.POST("/announcements/create", announcementHandler.Create)
				admin.POST("/announcements/:id/update", announcementHandler.Update)
				admin.POST("/announcements/:id/delete", announcementHandler.Delete)

				// 通知管理（创建全局通知）
				admin.POST("/notifications", notificationHandler.Create)
			}
		}
	}

	/* 前端静态文件服务 + SPA fallback（仅在 out/ 含构建产物时启用） */
	SetupFrontend(router)

	return router
}

/*
localOnlyGuard 本地访问限制中间件
功能：仅允许 127.0.0.1 / ::1 / localhost 访问，
用于保护 /metrics 和 /ws/stats 等敏感运维端点。
生产环境应额外通过反向代理限制访问。
*/
func localOnlyGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip != "127.0.0.1" && ip != "::1" && ip != "localhost" {
			c.JSON(403, gin.H{
				"success": false,
				"message": "此端点仅允许本地访问",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
