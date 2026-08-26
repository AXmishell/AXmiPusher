// Package api 路由装配。
package api

import (
	"context"
	"time"

	"messagepusher/internal/api/handler"
	"messagepusher/internal/api/middleware"
	"messagepusher/internal/app"
	"messagepusher/internal/compat/serverchan"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// NewRouter 装配全部路由。
func NewRouter(a *app.App) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	// 安装路由(无需安装状态)。
	a.InstallRoutes(r)

	// 健康检查。
	r.GET("/api/v1/health", func(c *gin.Context) {
		redisStatus := "disabled"
		if a.Redis != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := a.Redis.Ping(ctx).Err(); err != nil {
				redisStatus = "down"
			} else {
				redisStatus = "ok"
			}
		}
		response.OK(c, gin.H{
			"status":    "ok",
			"installed": isInstalled(),
			"version":   "1.1.0-gitflow",
			"redis":     redisStatus,
			"limiter":   a.Limiter.Type(),
			"breaker":   map[bool]string{true: "redis", false: "memory"}[a.RedisMode],
		})
	})

	// 业务 API(要求已安装)。
	getAuth := func() middleware.AuthService { return a.Auth }
	getAdminAuth := func() middleware.AdminAuthService { return a.Auth }
	biz := r.Group("/api/v1", middleware.RequireInstalled())
	{
		// 认证。
		auth := biz.Group("/auth")
		{
			auth.POST("/register", handler.Register(a))
			auth.POST("/login", handler.Login(a))
			auth.POST("/login/totp", handler.LoginTotp(a))
			auth.GET("/me", middleware.RequireAuth(getAuth), handler.Me(a))
			auth.PUT("/profile", middleware.RequireAuth(getAuth), handler.UpdateProfile(a))
			auth.PUT("/email", middleware.RequireAuth(getAuth), handler.ChangeEmail(a))
			auth.POST("/change-password", middleware.RequireAuth(getAuth), handler.ChangePassword(a))
			auth.POST("/totp/setup", middleware.RequireAuth(getAuth), handler.SetupTotp(a))
			auth.POST("/totp/confirm", middleware.RequireAuth(getAuth), handler.ConfirmTotp(a))
			auth.POST("/totp/disable", middleware.RequireAuth(getAuth), handler.DisableTotp(a))
		}

		// API Key 管理(登录态)。
		keys := biz.Group("/api-keys", middleware.RequireAuth(getAuth))
		{
			keys.GET("", handler.ListKeys(a))
			keys.POST("", handler.CreateKey(a))
			keys.DELETE("/:id", handler.DeleteKey(a))
		}

		// 兼容 key 管理(登录态)。
		compatKeys := biz.Group("/compat-keys", middleware.RequireAuth(getAuth))
		{
			compatKeys.GET("", handler.ListCompatKeys(a))
			compatKeys.POST("", handler.CreateCompatKey(a))
			compatKeys.DELETE("/:id", handler.DeleteCompatKey(a))
		}

		// 消息(登录态或 API Key)。
		messages := biz.Group("/messages", middleware.RequireAuthOrAPIKey(getAuth))
		{
			messages.POST("", handler.SendMessage(a))
			messages.GET("", handler.QueryMessages(a))
			messages.GET("/:id", handler.GetMessage(a))
		}

		// 回调订阅(登录态)。
		callbacks := biz.Group("/callbacks", middleware.RequireAuth(getAuth))
		{
			callbacks.GET("", handler.ListCallbacks(a))
			callbacks.POST("", handler.CreateCallback(a))
			callbacks.DELETE("/:id", handler.DeleteCallback(a))
		}

		// 统计(登录态)。
		stats := biz.Group("/stats", middleware.RequireAuth(getAuth))
		{
			stats.GET("/messages", handler.StatsMessages(a))
			stats.GET("/overview", handler.StatsOverview(a))
		}

		// 模板(登录态, 租户侧 CRUD + 提交审核)。
		templates := biz.Group("/templates", middleware.RequireAuth(getAuth))
		{
			templates.GET("", handler.ListTemplates(a))
			templates.POST("", handler.CreateTemplate(a))
			templates.GET("/:id", handler.GetTemplate(a))
			templates.PUT("/:id", handler.UpdateTemplate(a))
			templates.DELETE("/:id", handler.DeleteTemplate(a))
		}

		// 支付/订阅(登录态)。
		pay := biz.Group("/pay", middleware.RequireAuth(getAuth))
		{
			pay.GET("/plans", handler.ListPayPlans(a))
			pay.GET("/subscription", handler.GetSubscription(a))
			pay.POST("/orders", handler.CreatePayOrder(a))
			pay.GET("/orders/:id", handler.QueryPayOrder(a))
			pay.POST("/orders/:id/simulate", handler.SimulatePay(a)) // 本地调试用
		}

		// 渠道配置(登录态, 租户覆盖)。
		channels := biz.Group("/channels", middleware.RequireAuth(getAuth))
		{
			channels.GET("", handler.ListChannels(a))
			channels.GET("/health", handler.ChannelHealth(a))
			channels.PUT("/:type", handler.UpdateChannel(a))
			channels.DELETE("/:type", handler.DeleteChannel(a))
		}

		// 批量任务(登录态)。
		batch := biz.Group("/batch-tasks", middleware.RequireAuth(getAuth))
		{
			batch.POST("", handler.CreateBatchTask(a))
			batch.GET("", handler.ListBatchTasks(a))
			batch.GET("/:id", handler.GetBatchTask(a))
			batch.POST("/:id/cancel", handler.CancelBatchTask(a))
		}

		// 站内信收件箱(登录态)。
		inbox := biz.Group("/inbox", middleware.RequireAuth(getAuth))
		{
			inbox.GET("", handler.ListInbox(a))
			inbox.GET("/unread-count", handler.UnreadInboxCount(a))
			inbox.PUT("/:id/read", handler.MarkInboxRead(a))
			inbox.PUT("/read-all", handler.MarkAllInboxRead(a))
		}

		// 管理员后台 API(独立管理员体系, 支持多管理员)。
		admin := biz.Group("/admin")
		{
			// 管理员认证(登录公开, 其余需管理员 JWT)。
			admin.POST("/auth/login", handler.AdminLogin(a))
			admin.POST("/auth/login/totp", handler.AdminLoginTotp(a))
			admin.GET("/auth/me", middleware.RequireAdminAuth(getAdminAuth), handler.AdminMe(a))
			admin.PUT("/auth/profile", middleware.RequireAdminAuth(getAdminAuth), handler.AdminUpdateProfile(a))
			admin.PUT("/auth/email", middleware.RequireAdminAuth(getAdminAuth), handler.AdminChangeEmail(a))
			admin.POST("/auth/change-password", middleware.RequireAdminAuth(getAdminAuth), handler.AdminChangePassword(a))
			admin.POST("/auth/totp/setup", middleware.RequireAdminAuth(getAdminAuth), handler.AdminSetupTotp(a))
			admin.POST("/auth/totp/confirm", middleware.RequireAdminAuth(getAdminAuth), handler.AdminConfirmTotp(a))
			admin.POST("/auth/totp/disable", middleware.RequireAdminAuth(getAdminAuth), handler.AdminDisableTotp(a))

			// 需管理员登录的管理端点。
			authed := admin.Group("", middleware.RequireAdminAuth(getAdminAuth))
			{
				authed.GET("/stats", handler.AdminStats(a))
				authed.GET("/users", handler.ListUsers(a))
				authed.PUT("/users/:id/status", handler.SetUserStatus(a))
				authed.GET("/templates/reviews", handler.ListReviews(a))
				authed.POST("/templates/:templateId/versions/:versionId/approve", handler.ApproveReview(a))
				authed.POST("/templates/:templateId/versions/:versionId/reject", handler.RejectReview(a))
				authed.GET("/plans", handler.ListPlans(a))
				authed.POST("/plans", handler.CreatePlan(a))
				authed.PUT("/plans/:id", handler.UpdatePlan(a))
				authed.DELETE("/plans/:id", handler.DeletePlan(a))
				authed.GET("/payment-orders", handler.ListPaymentOrders(a))
				authed.GET("/audit-logs", handler.ListAuditLogs(a))
				authed.GET("/settings", handler.GetSettings(a))
				authed.PUT("/settings", handler.UpdateSettings(a))
				authed.POST("/settings/rotate-admin-path", handler.RotateAdminPath(a))

				// 管理员管理(仅超管)。
				admins := authed.Group("/admins", middleware.RequireAdminRole(models.AdminRoleSuper))
				{
					admins.GET("", handler.ListAdmins(a))
					admins.POST("", handler.CreateAdmin(a))
					admins.PUT("/:id/status", handler.SetAdminStatus(a))
					admins.PUT("/:id/password", handler.ResetAdminPassword(a))
				}
			}
		}
	}

	// 兼容层路由(公开, 走兼容 key 鉴权)。
	serverchan.RegisterRoutes(r, a)

	// 支付回调路由(公开, 服务端验签; 不经过 RequireInstalled)。
	r.POST("/api/v1/pay/notify", handler.PayNotify(a))
	r.GET("/api/v1/pay/return", handler.PayReturn(a))

	// 生产模式前端静态托管(根 → 用户中心, /{admin_path}/ → 管理后台)。
	registerWebRoutes(r, a.Cfg.Web.UserDist, a.Cfg.Web.AdminDist, a.Cfg.Admin.RandomPath)

	return r
}

// isInstalled 安装状态检查(由 install 模块通过 config 判断)。
func isInstalled() bool {
	return installed()
}

var installed = func() bool { return true }

// SetInstalledChecker 注入安装状态检查(由 main 调用)。
func SetInstalledChecker(fn func() bool) {
	installed = fn
	middleware.SetInstalledChecker(fn)
}
