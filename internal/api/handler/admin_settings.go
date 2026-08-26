package handler

import (
	"encoding/json"
	"strconv"

	"axmipusher/internal/app"
	"axmipusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetSettings 获取平台设置(敏感字段脱敏)。
func GetSettings(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// SMTP(隐藏密码)。
		smtp := map[string]interface{}{}
		if raw, err := a.Settings.Get("smtp", ""); err == nil && raw != "" {
			json.Unmarshal([]byte(raw), &smtp)
			if _, ok := smtp["password"]; ok {
				smtp["password"] = ""
			}
		}
		// 易支付(隐藏 key)。
		epay := map[string]interface{}{}
		if raw, err := a.Settings.Get("epay", ""); err == nil && raw != "" {
			json.Unmarshal([]byte(raw), &epay)
			if _, ok := epay["key"]; ok {
				epay["key"] = ""
			}
		}
		retention, _ := a.Settings.Get("retention_days", "90")
		adminPath := a.Cfg.Admin.RandomPath
		rateLimit, _ := a.Settings.Get("rate_limit_per_minute", "600")
		response.OK(c, gin.H{
			"smtp":            smtp,
			"epay":            epay,
			"retention_days":  retention,
			"admin_path":      adminPath,
			"rate_limit_per_minute": rateLimit,
		})
	}
}

// UpdateSettings 更新平台设置。
func UpdateSettings(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SMTP           *json.RawMessage `json:"smtp"`
			Epay           *json.RawMessage `json:"epay"`
			RetentionDays  *int             `json:"retention_days"`
			RateLimitPerMinute *int         `json:"rate_limit_per_minute"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
		if req.SMTP != nil {
			if err := a.Settings.Set("smtp", string(*req.SMTP), currentAdminID(c)); err != nil {
				response.ServerError(c, "保存 SMTP 配置失败")
				return
			}
		}
		if req.Epay != nil {
			if err := a.Settings.Set("epay", string(*req.Epay), currentAdminID(c)); err != nil {
				response.ServerError(c, "保存易支付配置失败")
				return
			}
		}
		if req.RetentionDays != nil {
			if err := a.Settings.Set("retention_days", strconv.Itoa(*req.RetentionDays), currentAdminID(c)); err != nil {
				response.ServerError(c, "保存失败")
				return
			}
			a.Cfg.Retention.MessageDays = *req.RetentionDays
		}
		if req.RateLimitPerMinute != nil {
			a.Settings.Set("rate_limit_per_minute", strconv.Itoa(*req.RateLimitPerMinute), currentAdminID(c))
			// 立即生效(限流器支持动态调整额度)。
			a.Cfg.RateLimit.PerMinute = *req.RateLimitPerMinute
			a.Limiter.SetPerMinute(*req.RateLimitPerMinute)
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "settings.update", gin.H{})
		response.OK(c, gin.H{"ok": true})
	}
}

// RotateAdminPath 轮换管理员后台随机路径。
func RotateAdminPath(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		oldPath := a.Cfg.Admin.RandomPath
		// 生成新随机路径并写入配置(持久化)。
		newPath, err := randomHex(8)
		if err != nil {
			response.ServerError(c, "生成随机路径失败")
			return
		}
		// 同步改写 admin dist index.html 的资源前缀(否则新路径下管理后台 404)。
		if a.Cfg.Web.AdminDist != "" {
			if err := rewriteAdminBase(a.Cfg.Web.AdminDist, oldPath, newPath); err != nil {
				response.ServerError(c, "改写前端 base 失败: "+err.Error())
				return
			}
		}
		// 动态注册新前缀路由(轮换后立即生效)。
		a.RegisterAdminSPA(newPath, a.Cfg.Web.AdminDist)
		a.Cfg.Admin.RandomPath = newPath
		if err := a.Cfg.Save(); err != nil {
			response.ServerError(c, "保存配置失败")
			return
		}
		Audit(a.DB, c, currentAdminID(c), currentAdminEmail(c), "settings.rotate_admin_path",
			gin.H{"new_path": newPath})
		response.OK(c, gin.H{"admin_path": newPath})
	}
}
